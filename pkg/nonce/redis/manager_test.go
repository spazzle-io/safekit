package redis

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
)

type mockSource struct {
	mu     sync.Mutex
	nonces []uint64
	idx    int
	err    error
}

func (m *mockSource) PendingNonceAt(_ context.Context, _ common.Address) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return 0, m.err
	}

	if m.idx >= len(m.nonces) {
		return m.nonces[len(m.nonces)-1], nil
	}

	n := m.nonces[m.idx]
	m.idx++
	return n, nil
}

func (m *mockSource) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.idx
}

func newManager(t *testing.T, mr *miniredis.Miniredis, src *mockSource, instanceID string, opts ...func(*Options)) *NonceManager {
	t.Helper()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	o := Options{
		Redis:           rdb,
		InstanceID:      instanceID,
		LockTTL:         5 * time.Second,
		PollInterval:    10 * time.Millisecond,
		StaleNonceDelay: 10 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(&o)
	}

	m, err := NewNonceManager(o)
	if err != nil {
		t.Fatalf("NewNonceManager: %v", err)
	}

	if err := m.Init(src, big.NewInt(137), common.HexToAddress("0x123")); err != nil {
		t.Fatal(err)
	}

	return m
}

func TestNewNonceManagerValidation_MissingRedis(t *testing.T) {
	if _, err := NewNonceManager(Options{InstanceID: "worker-1"}); err == nil {
		t.Fatal(err)
	}
}

func TestNewNonceManagerValidation_MissingInstanceID(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	nm, err := NewNonceManager(Options{Redis: rdb})
	if err != nil {
		t.Fatalf("want success when InstanceID omitted, got: %v", err)
	}

	if nm.instanceID == "" {
		t.Fatal("want generated instanceID, got empty string")
	}
}

func TestInitValidation(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}
	addr := common.HexToAddress("0x1234")
	chainID := big.NewInt(1)

	newUninitialisedManager := func(t *testing.T) *NonceManager {
		t.Helper()

		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = rdb.Close() })

		m, err := NewNonceManager(Options{
			Redis:      rdb,
			InstanceID: "worker-1",
		})
		if err != nil {
			t.Fatal(err)
		}

		return m
	}

	t.Run("missing source", func(t *testing.T) {
		m := newUninitialisedManager(t)
		if err := m.Init(nil, chainID, addr); err == nil {
			t.Fatal("want error for missing source")
		}
	})

	t.Run("nil chain ID", func(t *testing.T) {
		m := newUninitialisedManager(t)
		if err := m.Init(src, nil, addr); err == nil {
			t.Fatal("want error for nil chain ID")
		}
	})

	t.Run("empty address", func(t *testing.T) {
		m := newUninitialisedManager(t)
		if err := m.Init(src, chainID, common.Address{}); err == nil {
			t.Fatal("want error for empty address")
		}
	})
}

func TestAcquireBeforeInitReturnsError(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	m, err := NewNonceManager(Options{
		Redis:      rdb,
		InstanceID: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = m.Acquire(context.Background())
	if err == nil {
		t.Fatal("want error when Acquire called before Init")
	}
}

func TestNormalAcquireRelease(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{5}}
	m := newManager(t, mr, src, "worker-1")

	for i := uint64(0); i < 3; i++ {
		n, release, err := m.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire %d: %v", i, err)
		}

		if n != 5+i {
			t.Fatalf("Acquire %d: want %d got %d", i, 5+i, n)
		}

		release(nil)
	}

	if src.callCount() != 1 {
		t.Fatalf("want 1 chain call, got %d", src.callCount())
	}
}

func TestTwoInstancesCoordinate(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}

	m1 := newManager(t, mr, src, "worker-1")
	m2 := newManager(t, mr, src, "worker-2")

	const total = 10

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		nonces []uint64
	)

	acquire := func(m *NonceManager) {
		defer wg.Done()

		n, release, err := m.Acquire(context.Background())
		if err != nil {
			t.Errorf("Acquire: %v", err)
			return
		}

		mu.Lock()
		nonces = append(nonces, n)
		mu.Unlock()

		release(nil)
	}

	for i := 0; i < total; i++ {
		wg.Add(1)

		if i%2 == 0 {
			go acquire(m1)
		} else {
			go acquire(m2)
		}
	}

	wg.Wait()

	if len(nonces) != total {
		t.Fatalf("want %d nonces, got %d", total, len(nonces))
	}

	seen := make(map[uint64]bool)
	for _, n := range nonces {
		if seen[n] {
			t.Fatalf("duplicate nonce %d", n)
		}
		seen[n] = true
	}

	for i := uint64(0); i < total; i++ {
		if !seen[i] {
			t.Fatalf("missing nonce %d", i)
		}
	}
}

func TestReleaseErrTriggersRefetch(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{7, 9}}
	m := newManager(t, mr, src, "worker-1")

	n, release, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if n != 7 {
		t.Fatalf("want 7, got %d", n)
	}
	release(errors.New("send failed"))

	n, release, err = m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if n != 9 {
		t.Fatalf("want 9, got %d", n)
	}
	release(nil)

	if src.callCount() != 2 {
		t.Fatalf("want 2 chain calls, got %d", src.callCount())
	}
}

func TestRefetchRespectsDelay(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, mr, src, "worker-1", func(o *Options) {
		o.StaleNonceDelay = 100 * time.Millisecond
	})

	_, release, _ := m.Acquire(context.Background())
	release(errors.New("failed"))

	start := time.Now()
	_, release, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after failure: %v", err)
	}
	release(nil)

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("want at least 100ms delay, got %v", elapsed)
	}
}

func TestLockExpiryRecovery(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}

	m1 := newManager(t, mr, src, "worker-1", func(o *Options) {
		o.LockTTL = 100 * time.Millisecond
	})
	m2 := newManager(t, mr, src, "worker-2", func(o *Options) {
		o.LockTTL = 100 * time.Millisecond
	})

	_, _, err := m1.Acquire(context.Background())
	if err != nil {
		t.Fatalf("m1 Acquire: %v", err)
	}

	mr.FastForward(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, release, err := m2.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire after lock expiry: %v", err)
	}
	release(nil)
}

func TestReleaseLockScript(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, mr, src, "worker-1")

	_, _, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	err = mr.Set(m.lockKey(), "someone-else")
	require.NoError(t, err)

	err = m.releaseLock(context.Background())
	if err == nil {
		t.Fatal("want error releasing lock owned by someone else")
	}

	val, err := mr.Get(m.lockKey())
	require.NoError(t, err)
	if val != "someone-else" {
		t.Fatalf("lock should still be held by someone-else, got %q", val)
	}
}

func TestContextCancellationWhileWaitingForLock(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}

	m1 := newManager(t, mr, src, "worker-1")
	m2 := newManager(t, mr, src, "worker-2")

	_, _, err := m1.Acquire(context.Background())
	if err != nil {
		t.Fatalf("m1 Acquire: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err = m2.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestContextCancellationDuringRefetch(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, mr, src, "worker-1", func(o *Options) {
		o.StaleNonceDelay = 5 * time.Second
	})

	_, release, _ := m.Acquire(context.Background())
	release(errors.New("failed"))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := m.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestCorruptCounterReturnsError(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, mr, src, "worker-1")

	err := mr.Set(m.counterKey(), "not-a-number")
	require.NoError(t, err)

	_, _, err = m.Acquire(context.Background())
	if err == nil {
		t.Fatal("want error for corrupt counter value")
	}

	lockVal, _ := mr.Get(m.lockKey())
	if lockVal != "" {
		t.Fatal("want lock released after nextNonce failure")
	}
}

func TestDefaultsApplied(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	m, err := NewNonceManager(Options{
		Redis:      rdb,
		InstanceID: "worker-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if m.lockTTL != defaultLockTTL {
		t.Fatalf("want lockTTL %v, got %v", defaultLockTTL, m.lockTTL)
	}
	if m.pollInterval != defaultPollInterval {
		t.Fatalf("want pollInterval %v, got %v", defaultPollInterval, m.pollInterval)
	}
	if m.staleNonceDelay <= 0 {
		t.Fatal("want positive staleNonceDelay")
	}
}

func TestCounterKeyNamespacing(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0}}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	m, err := NewNonceManager(Options{Redis: rdb, InstanceID: "worker-1"})
	if err != nil {
		t.Fatalf("NewNonceManager: %v", err)
	}

	addr := common.HexToAddress("0xDea")
	if err := m.Init(src, big.NewInt(137), addr); err != nil {
		t.Fatal(err)
	}

	expectedCounter := "safekit:nonce:137:" + strings.ToLower(addr.Hex())
	expectedLock := expectedCounter + ":lock"

	if m.counterKey() != expectedCounter {
		t.Fatalf("want counter key %q, got %q", expectedCounter, m.counterKey())
	}
	if m.lockKey() != expectedLock {
		t.Fatalf("want lock key %q, got %q", expectedLock, m.lockKey())
	}
}

func TestCounterSeededOnce(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{10}}
	m := newManager(t, mr, src, "worker-1")

	for i := 0; i < 5; i++ {
		_, release, err := m.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire %d: %v", i, err)
		}
		release(nil)
	}

	if src.callCount() != 1 {
		t.Fatalf("want 1 chain call, got %d", src.callCount())
	}

	val, err := redis.NewClient(&redis.Options{Addr: mr.Addr()}).Get(context.Background(), m.counterKey()).Result()
	if err != nil {
		t.Fatalf("GET counter: %v", err)
	}

	n, _ := strconv.ParseUint(val, 10, 64)
	if n != 15 {
		t.Fatalf("want counter 15, got %d", n)
	}
}
