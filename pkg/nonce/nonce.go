package nonce

import "context"

// Slot represents an acquired nonce slot. The caller must invoke exactly one of Commit, Reuse, or Reclaim after
// every successful Acquire.
type Slot interface {
	// Commit signals the transaction was successfully broadcast.
	// The nonce is consumed and the counter advances to N+1.
	Commit()

	// Reuse signals the nonce was acquired but never broadcast.
	// The counter stays at N. The next Acquire receives N directly without re-fetching from the chain.
	Reuse()

	// Reclaim signals the broadcast was attempted but failed.
	// The counter is marked dirty and re-fetched from the chain on the next Acquire after a short delay.
	Reclaim()
}

// Manager serialises nonce assignment across concurrent deployments.
type Manager interface {
	// Acquire blocks until a nonce slot is available, returning the next nonce and a Slot the caller must resolve exactly once.
	Acquire(ctx context.Context) (nonce uint64, slot Slot, err error)
}
