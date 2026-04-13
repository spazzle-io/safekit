package nonce

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// DefaultStaleNonceDelay is how long a NonceManager waits before re-fetching the pending nonce from the
// chain after a failed broadcast. This gives the node time to settle its mempool state before we query it again.
const DefaultStaleNonceDelay = 500 * time.Millisecond

// NonceSource is the minimal interface required to fetch the pending nonce for an account.
// *ethclient.Client satisfies this interface.
type NonceSource interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
}
