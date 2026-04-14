package redis

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func acquireSlot(t *testing.T, m *NonceManager) (uint64, *slot) {
	t.Helper()
	n, s, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return n, s.(*slot)
}

func counterVal(t *testing.T, mr *miniredis.Miniredis, m *NonceManager) uint64 {
	t.Helper()

	val, err := mr.Get(m.counterKey())
	if err != nil {
		t.Fatalf("GET counter: %v", err)
	}

	n, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		t.Fatalf("parse counter: %v", err)
	}

	return n
}

func lockHeld(mr *miniredis.Miniredis, m *NonceManager) bool {
	val, err := mr.Get(m.lockKey())
	return err == nil && val != ""
}

func TestCommitIdempotent(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{5}}
	m := newTestManager(t, mr, src, "worker-1")

	_, s := acquireSlot(t, m)

	s.Commit()
	if lockHeld(mr, m) {
		t.Fatal("want lock released after Commit")
	}

	s.Commit() // second call must be a no-op
	if lockHeld(mr, m) {
		t.Fatal("want lock released after Commit")
	}

	// Should be able to acquire again cleanly.
	_, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after double Commit: %v", err)
	}
	s2.Commit()
}

func TestReuseIdempotent(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{5}}
	m := newTestManager(t, mr, src, "worker-1")

	n, s := acquireSlot(t, m)
	if n != 5 {
		t.Fatalf("want 5, got %d", n)
	}

	if cv := counterVal(t, mr, m); cv != 6 {
		t.Fatalf("want counter 6 after Acquire, got %d", cv)
	}

	s.Reuse()
	if cv := counterVal(t, mr, m); cv != 5 {
		t.Fatalf("want counter 5 after Reuse, got %d", cv)
	}

	s.Reuse() // second call must be a no-op
	if cv := counterVal(t, mr, m); cv != 5 {
		t.Fatalf("want counter still 5 after double Reuse, got %d — decremented twice", cv)
	}

	n2, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after double Reuse: %v", err)
	}
	if n2 != 5 {
		t.Fatalf("want 5 after Reuse, got %d", n2)
	}
	s2.Commit()
}

func TestReclaimIdempotent(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newTestManager(t, mr, src, "worker-1", func(o *Options) {
		o.StaleNonceDelay = 10 * time.Millisecond
	})

	_, s := acquireSlot(t, m)

	s.Reclaim()
	if lockHeld(mr, m) {
		t.Fatal("want lock released after Reclaim")
	}

	s.Reclaim() // second call must be a no-op
	if lockHeld(mr, m) {
		t.Fatal("want lock released after Reclaim")
	}

	_, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after double Reclaim: %v", err)
	}
	s2.Commit()
}

func TestMixedDoubleCall(t *testing.T) {
	mr := miniredis.RunT(t)
	src := &mockSource{nonces: []uint64{7}}
	m := newTestManager(t, mr, src, "worker-1")

	n, s := acquireSlot(t, m)
	if n != 7 {
		t.Fatalf("want 7, got %d", n)
	}

	s.Commit() // nonce 7 committed, counter stays at 8
	s.Reuse()  // must be a no-op. Must NOT decrement counter to 7

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	val, err := rdb.Get(context.Background(), m.counterKey()).Result()
	if err != nil {
		t.Fatalf("GET counter: %v", err)
	}
	cv, _ := strconv.ParseUint(val, 10, 64)
	if cv != 8 {
		t.Fatalf("want counter 8 after Commit+Reuse, got %d. Reuse incorrectly ran after Commit", cv)
	}

	n2, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after Commit+Reuse: %v", err)
	}
	if n2 != 8 {
		t.Fatalf("want 8, got %d", n2)
	}
	s2.Commit()
}
