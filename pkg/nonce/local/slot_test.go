package local

import (
	"context"
	"testing"
	"time"
)

func acquireSlot(t *testing.T, m *NonceManager) (uint64, *slot) {
	t.Helper()

	n, s, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return n, s.(*slot)
}

func TestCommitIdempotent(t *testing.T) {
	src := &mockSource{nonces: []uint64{0}}
	m := newManager(t, src, 0)

	_, s := acquireSlot(t, m)
	s.Commit()
	s.Commit() // second call must be a no-op

	// Drain the one legitimate token by acquiring.
	_, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after double Commit: %v", err)
	}
	// intentionally do not resolve s2 slot.

	// If double Commit added two tokens, a third Acquire would succeed.
	// It should block and time out because the channel is now empty.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err = m.Acquire(ctx)
	if err == nil {
		t.Fatal("third Acquire succeeded. We must have 2 tokens in the chan")
	}

	// resolve s2
	s2.Commit()
}

func TestReuseIdempotent(t *testing.T) {
	src := &mockSource{nonces: []uint64{5}}
	m := newManager(t, src, 0)

	n, s := acquireSlot(t, m)
	if n != 5 {
		t.Fatalf("want 5, got %d", n)
	}

	s.Reuse()
	s.Reuse() // second call must be a no-op

	// Next Acquire should get 5 again.
	n2, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after double Reuse: %v", err)
	}
	if n2 != 5 {
		t.Fatalf("want 5 after Reuse, got %d — counter was decremented more than once", n2)
	}
	s2.Commit()
}

func TestReclaimIdempotent(t *testing.T) {
	src := &mockSource{nonces: []uint64{0, 1}}
	m := newManager(t, src, 0)

	_, s := acquireSlot(t, m)
	s.Reclaim()
	s.Reclaim() // must be a no-op

	// Drain the one legitimate token.
	_, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after double Reclaim: %v", err)
	}
	// intentionally do not resolve s2 slot.

	// If double Reclaim added two tokens, this would succeed.
	// It should block and time out.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err = m.Acquire(ctx)
	if err == nil {
		t.Fatal("third Acquire succeeded. We must have 2 tokens in the chan")
	}

	s2.Commit()
}

func TestMixedDoubleCall(t *testing.T) {
	src := &mockSource{nonces: []uint64{7}}
	m := newManager(t, src, 0)

	n, s := acquireSlot(t, m)
	if n != 7 {
		t.Fatalf("want 7, got %d", n)
	}

	s.Commit() // nonce 7 committed, counter moves to 8
	s.Reuse()  // must be a no-op. Counter must NOT decrement back to 7

	// Next Acquire should get 8, not 7.
	n2, s2, err := m.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire after Commit+Reuse: %v", err)
	}
	if n2 != 8 {
		t.Fatalf("want 8, got %d — Reuse incorrectly decremented after Commit", n2)
	}
	s2.Commit()
}
