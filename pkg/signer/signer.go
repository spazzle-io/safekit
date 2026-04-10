// Package signer handles transaction signing for the Safe admin wallet.
// This is the only place in safekit that ever touches a private key.
// Everything else works with addresses and already-signed transactions.
package signer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Signer signs Ethereum transactions on behalf of a single address.
type Signer interface {
	// Address returns the public address this signer controls.
	Address() common.Address

	// SignTx signs a transaction for the given chain ID and returns
	// the signed version ready to broadcast.
	SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// Close cleans up any key material held by the signer.
	// Always call this when done.
	Close()
}
