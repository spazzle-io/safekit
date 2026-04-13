package safe

import "context"

// ReleaseFunc must be called exactly once after every successful Acquire.
// Pass nil if SendTransaction succeeded as the nonce is likely consumed.
// Pass a non-nil error if SendTransaction failed and the nonce will be re-fetched
// on the next Acquire.
type ReleaseFunc func(broadcastErr error)

// NonceManager serialises nonce assignment across concurrent deployments.
//
// Two implementations are provided out of the box:
//   - pkg/safe/local: single-process, in-memory (default when none is provided to safe.New)
//   - pkg/safe/redis: distributed, Redis-backed (for multiple processes sharing a signer)
type NonceManager interface {
	// Acquire blocks until a nonce slot is available, returning the next nonce and a
	// release function the caller must invoke exactly once after attempting to broadcast the transaction.
	Acquire(ctx context.Context) (nonce uint64, release ReleaseFunc, err error)
}
