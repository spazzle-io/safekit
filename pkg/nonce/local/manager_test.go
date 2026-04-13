package local

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
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

func newManager(t *testing.T, src *mockSource, delay time.Duration) *NonceManager {
	t.Helper()

	m, err := NewNonceManager(Options{StaleNonceDelay: delay})
	require.NoError(t, err)

	if err := m.Init(src, big.NewInt(1), common.HexToAddress("0x123")); err != nil {
		t.Fatal(err)
	}

	return m
}

func TestSequentialAcquireRelease(t *testing.T) {
	src := &mockSource{nonces: []uint64{5}}
	m := newManager(t, src, 0)

	for i := uint64(0); i < 5; i++ {
		n, release, err := m.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire %d: unexpected error: %v", i, err)
		}
		if n != 5+i {
			t.Fatalf("Acquire %d: want nonce %d got %d", i, 5+i, n)
		}
		release(nil)
	}

	if src.callCount() != 1 {
		t.Fatalf("want 1 chain call, got %d", src.callCount())
	}
}

func TestConcurrentAcquireProducesSequentialNonces(t *testing.T) {
	const goroutines = 10

	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, src, 0)

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		nonces []uint64
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			n, release, err := m.Acquire(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			mu.Lock()
			nonces = append(nonces, n)
			mu.Unlock()
			release(nil)
		}()
	}

	wg.Wait()

	if len(nonces) != goroutines {
		t.Fatalf("want %d nonces, got %d", goroutines, len(nonces))
	}

	seen := make(map[uint64]bool)
	for _, n := range nonces {
		if seen[n] {
			t.Fatalf("duplicate nonce %d", n)
		}

		seen[n] = true
	}

	for i := uint64(0); i < goroutines; i++ {
		if !seen[i] {
			t.Fatalf("missing nonce %d", i)
		}
	}
}

func TestReleaseErrTriggersRefetch(t *testing.T) {
	src := &mockSource{nonces: []uint64{7, 8}}
	m := newManager(t, src, 10*time.Millisecond)

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
	if n != 8 {
		t.Fatalf("want 8, got %d", n)
	}
	release(nil)

	if src.callCount() != 2 {
		t.Fatalf("want 2 chain calls, got %d", src.callCount())
	}
}

func TestRefetchRespectsDelay(t *testing.T) {
	delay := 100 * time.Millisecond
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, src, delay)

	n, release, _ := m.Acquire(context.Background())
	_ = n
	release(errors.New("failed"))

	start := time.Now()
	_, release, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire after failure: %v", err)
	}
	release(nil)

	elapsed := time.Since(start)
	if elapsed < delay {
		t.Fatalf("want at least %v delay, got %v", delay, elapsed)
	}
}

func TestContextCancellationWhileWaiting(t *testing.T) {
	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, src, 0)

	_, _, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	// Intentionally do not release slot

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err = m.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestContextCancellationDuringRefetch(t *testing.T) {
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, src, 5*time.Second) // long delay

	n, release, _ := m.Acquire(context.Background())
	_ = n
	release(errors.New("failed"))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := m.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestZeroDelayAppliesDefault(t *testing.T) {
	m, err := NewNonceManager(Options{})
	require.NoError(t, err)

	if m.staleNonceDelay <= 0 {
		t.Fatal("want positive stale nonce delay, got zero or negative")
	}
}

func TestInitValidation(t *testing.T) {
	m, err := NewNonceManager(Options{})
	require.NoError(t, err)

	err = m.Init(nil, big.NewInt(1), common.HexToAddress("0x123"))
	require.Error(t, err)

	err = m.Init(&mockSource{nonces: []uint64{5}}, big.NewInt(1), common.Address{})
	require.Error(t, err)
}

func TestNonceNotRefetchedOnSuccess(t *testing.T) {
	src := &mockSource{nonces: []uint64{3}}
	m := newManager(t, src, 0)

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
}

func TestSlotReturnedOnFetchError(t *testing.T) {
	src := &mockSource{err: errors.New("rpc error")}
	m := newManager(t, src, 0)

	_, _, err := m.Acquire(context.Background())
	if err == nil {
		t.Fatal("want error from failed fetch")
	}

	src.mu.Lock()
	src.err = nil
	src.nonces = []uint64{0}
	src.mu.Unlock()

	var acquired atomic.Bool
	done := make(chan struct{})

	go func() {
		_, release, err := m.Acquire(context.Background())
		if err == nil {
			release(nil)
			acquired.Store(true)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second Acquire blocked")
	}

	if !acquired.Load() {
		t.Fatal("second Acquire did not succeed")
	}
}
