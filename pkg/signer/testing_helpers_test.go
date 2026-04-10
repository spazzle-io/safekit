package signer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// newTestTx returns a minimal EIP-1559 transaction suitable for signing tests.
func newTestTx() *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     0,
		GasTipCap: big.NewInt(1e9),  // 1 gwei tip
		GasFeeCap: big.NewInt(10e9), // 10 gwei max fee
		Gas:       21000,
		To:        ptrAddress(common.HexToAddress("0x000000000000000000000000000000000000dEaD")),
		Value:     big.NewInt(0),
	})
}

func ptrAddress(a common.Address) *common.Address {
	return &a
}
