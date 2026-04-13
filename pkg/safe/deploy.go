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

// Deploy predicts, deploys, and verifies a new Safe in one call.
// It blocks until the deployment transaction is mined or the context is cancelled.
//
// If you want to separate submission from confirmation, use
// SubmitDeployment to submit the transaction and get a hash immediately,
// then WaitForDeployment to block until it mines. If you want fully
// non-blocking behaviour, use SubmitDeployment and then poll with
// IsDeployed at your own pace rather than calling WaitForDeployment.
//
// The predicted address is verified against the actual deployed address.
// If they differ, a [DeploymentMismatchError] is returned. This indicates
// a bug in safekit. Please open an issue at github.com/spazzle-io/safekit with the details.
//
// The admin signer configured in Options pays for gas but is never added
// as an owner. Pass the admin's address in owners explicitly if you want it to be an owner.
func (c *Client) Deploy(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (*DeployResult, error) {
	if err := validateSafeConfig(owners, threshold); err != nil {
		return nil, err
	}

	result, err := c.deployer.Deploy(
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

// SubmitDeployment submits the deployment transaction and returns the
// transaction hash immediately without waiting for it to be mined.
// This is useful when integrating with async job queues where you want
// to track the transaction hash separately from the mining confirmation.
//
// Use [Client.WaitForDeployment] to block until the transaction mines,
// or poll [Client.IsDeployed] at your own pace for fully non-blocking behaviour.
//
// The admin signer pays for gas but is never added as an owner unless explicitly added
// to the owners list.
func (c *Client) SubmitDeployment(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (common.Hash, error) {
	if err := validateSafeConfig(owners, threshold); err != nil {
		return common.Hash{}, err
	}

	txHash, err := c.deployer.Submit(
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

// WaitForDeployment waits for a previously submitted deployment transaction
// to be mined and returns the result.
//
// The predicted address is verified against the actual deployed address.
// If they differ, a DeploymentMismatchError is returned.
func (c *Client) WaitForDeployment(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
	txHash common.Hash,
) (*DeployResult, error) {
	result, err := c.deployer.Wait(
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

// RandomSalt generates a cryptographically random 32-byte salt suitable
// for use with Deploy and PredictAddress.
//
// Use this when you don't need a reproducible address. For deterministic
// addresses (e.g. one Safe per user derived from a user ID), provide your
// own salt instead.
func RandomSalt() ([]byte, error) {
	return generateRandomSalt()
}
