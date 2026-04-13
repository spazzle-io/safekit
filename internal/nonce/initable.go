package nonce

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Initable is implemented by NonceManagers that require initialisation with chain-specific context before use.
// safe.New checks for this interface and calls Init automatically.
type Initable interface {
	Init(source Source, chainID *big.Int, address common.Address) error
}
