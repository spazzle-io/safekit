package deployer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// defaultGasMultiplier is applied to the estimated gas to give headroom
// above the estimate.
const defaultGasMultiplier = 1.2

// estimateGas estimates the gas required for a contract call and applies
// the configured multiplier to give headroom above the raw estimate.
func estimateGas(
	ctx context.Context,
	client *ethclient.Client,
	from common.Address,
	to common.Address,
	data []byte,
	multiplier float64,
) (uint64, error) {
	if multiplier <= 0 {
		multiplier = defaultGasMultiplier
	}

	msg := ethereum.CallMsg{
		From: from,
		To:   &to,
		Data: data,
	}

	estimated, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("gas estimation failed: %w", err)
	}

	withHeadroom := uint64(float64(estimated) * multiplier)
	return withHeadroom, nil
}

// suggestGasPrice returns the current suggested gas price for legacy
// transactions, used as a fallback on chains that don't support EIP-1559.
func suggestGasPrice(ctx context.Context, client *ethclient.Client) (*big.Int, error) {
	price, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	return price, nil
}

// suggestGasTipCap returns the current suggested priority fee (tip) for EIP-1559 transactions.
func suggestGasTipCap(ctx context.Context, client *ethclient.Client) (*big.Int, error) {
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas tip: %w", err)
	}

	return tip, nil
}
