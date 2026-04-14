package safe

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/internal/predict"
	"github.com/spazzle-io/safekit/internal/txmanager"
)

// DeployResult is returned after a successful Safe deployment.
type DeployResult struct {
	// SafeAddress is the address of the newly deployed Safe proxy.
	SafeAddress common.Address

	// TxHash is the hash of the deployment transaction.
	TxHash common.Hash

	// BlockNumber is the block in which the deployment was mined.
	BlockNumber uint64

	// GasUsed is the actual gas consumed by the deployment transaction.
	GasUsed uint64
}

func (c *client) Deploy(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (*DeployResult, error) {
	if err := validateSafeConfig(owners, threshold); err != nil {
		return nil, err
	}

	result, err := c.txManager.Deploy(
		ctx,
		c.deployment,
		c.chain.ID,
		c.chain.IsL2,
		predict.Input{
			Owners:    owners,
			Threshold: threshold,
			Salt:      salt,
		},
		&txmanager.Options{
			GasMultiplier: c.opts.gasMultiplier(),
			Timeout:       c.opts.deployTimeout(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("safe: %w", err)
	}

	return &DeployResult{
		SafeAddress: result.SafeAddress,
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		GasUsed:     result.GasUsed,
	}, nil
}

func (c *client) SubmitDeployment(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (common.Hash, error) {
	if err := validateSafeConfig(owners, threshold); err != nil {
		return common.Hash{}, err
	}

	txHash, err := c.txManager.Submit(
		ctx,
		c.deployment,
		c.chain.ID,
		c.chain.IsL2,
		predict.Input{
			Owners:    owners,
			Threshold: threshold,
			Salt:      salt,
		},
		&txmanager.Options{
			GasMultiplier: c.opts.gasMultiplier(),
		},
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("safe: %w", err)
	}
	return txHash, nil
}

func (c *client) WaitForDeployment(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
	txHash common.Hash,
) (*DeployResult, error) {
	result, err := c.txManager.Wait(
		ctx,
		c.deployment,
		c.chain.ID,
		c.chain.IsL2,
		predict.Input{
			Owners:    owners,
			Threshold: threshold,
			Salt:      salt,
		},
		txHash,
	)
	if err != nil {
		return nil, fmt.Errorf("safe: %w", err)
	}

	return &DeployResult{
		SafeAddress: result.SafeAddress,
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		GasUsed:     result.GasUsed,
	}, nil
}
