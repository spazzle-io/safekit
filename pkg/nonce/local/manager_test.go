package local

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

	m := NewNonceManager(Options{StaleNonceDelay: delay})

	if err := m.Init(src, big.NewInt(1), common.HexToAddress("0x123")); err != nil {
		t.Fatal(err)
	}

	return m
}

func TestAcquireBeforeInitReturnsError(t *testing.T) {
	m := NewNonceManager(Options{})

	_, _, err := m.Acquire(context.Background())
	if err == nil {
		t.Fatal("want error when Acquire called before Init")
	}
}

func TestSequentialCommit(t *testing.T) {
	src := &mockSource{nonces: []uint64{5}}
	m := newManager(t, src, 0)

	for i := uint64(0); i < 5; i++ {
		n, slot, err := m.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire %d: %v", i, err)
		}
		if n != 5+i {
			t.Fatalf("Acquire %d: want nonce %d got %d", i, 5+i, n)
		}
		slot.Commit()
	}

	if src.callCount() != 1 {
		t.Fatalf("want 1 chain call, got %d", src.callCount())
	}
}

func TestReuseReturnsSameNonce(t *testing.T) {
	src := &mockSource{nonces: []uint64{7}}
	m := newManager(t, src, 0)

	n, slot, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if n != 7 {
		t.Fatalf("want 7, got %d", n)
	}
	slot.Reuse()

	n, slot, err = m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if n != 7 {
		t.Fatalf("want 7 again after Reuse, got %d", n)
	}
	slot.Commit()

	if src.callCount() != 1 {
		t.Fatalf("want 1 chain call, got %d", src.callCount())
	}
}

func TestReclaimTriggersRefetch(t *testing.T) {
	src := &mockSource{nonces: []uint64{7, 8}}
	m := newManager(t, src, 10*time.Millisecond)

	n, slot, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if n != 7 {
		t.Fatalf("want 7, got %d", n)
	}
	slot.Reclaim()

	n, slot, err = m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if n != 8 {
		t.Fatalf("want 8 after Reclaim, got %d", n)
	}
	slot.Commit()

	if src.callCount() != 2 {
		t.Fatalf("want 2 chain calls, got %d", src.callCount())
	}
}

func TestReclaimRespectsDelay(t *testing.T) {
	delay := 100 * time.Millisecond
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, src, delay)

	_, slot, _ := m.Acquire(context.Background())
	slot.Reclaim()

	start := time.Now()
	_, slot, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after Reclaim: %v", err)
	}
	slot.Commit()

	if elapsed := time.Since(start); elapsed < delay {
		t.Fatalf("want at least %v delay, got %v", delay, elapsed)
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
			n, slot, err := m.Acquire(context.Background())
			if err != nil {
				t.Errorf("Acquire: %v", err)
				return
			}
			mu.Lock()
			nonces = append(nonces, n)
			mu.Unlock()
			slot.Commit()
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

func TestContextCancellationWhileWaiting(t *testing.T) {
	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, src, 0)

	_, _, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err = m.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestContextCancellationDuringRefetch(t *testing.T) {
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, src, 5*time.Second)

	_, slot, _ := m.Acquire(context.Background())
	slot.Reclaim()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := m.Acquire(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestZeroDelayAppliesDefault(t *testing.T) {
	m := NewNonceManager(Options{})
	if m.staleNonceDelay <= 0 {
		t.Fatal("want positive stale nonce delay, got zero or negative")
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
		_, slot, err := m.Acquire(context.Background())
		if err == nil {
			slot.Commit()
			acquired.Store(true)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second Acquire blocked — slot was not returned after fetch error")
	}

	if !acquired.Load() {
		t.Fatal("second Acquire did not succeed")
	}
}

func TestReuseAfterMultipleCommits(t *testing.T) {
	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, src, 0)

	for i := uint64(0); i < 3; i++ {
		n, slot, err := m.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire %d: %v", i, err)
		}
		if n != i {
			t.Fatalf("want %d, got %d", i, n)
		}
		slot.Commit()
	}

	n, slot, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire for Reuse: %v", err)
	}
	if n != 3 {
		t.Fatalf("want 3, got %d", n)
	}
	slot.Reuse()

	n, slot, err = m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after Reuse: %v", err)
	}
	if n != 3 {
		t.Fatalf("want 3 after Reuse, got %d", n)
	}
	slot.Commit()
}
