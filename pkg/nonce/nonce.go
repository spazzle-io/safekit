package nonce

import "context"

// ReleaseFunc must be called exactly once after every successful Acquire.
// Pass nil if SendTransaction succeeded as the nonce is likely consumed.
// Pass a non-nil error if SendTransaction failed and the nonce will be re-fetched on the next Acquire.
type ReleaseFunc func(broadcastErr error)

// Manager serialises nonce assignment across concurrent deployments.
type Manager interface {
	// Acquire blocks until a nonce slot is available, returning the next nonce and a release function
	// the caller must invoke exactly once after attempting to broadcast the transaction.
	Acquire(ctx context.Context) (nonce uint64, release ReleaseFunc, err error)
}
