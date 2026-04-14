// Package mock provides an in-memory Safe client for use in tests.
// It implements safe.Client without making any network calls.
//
// Usage:
//
//	client := mock.NewClient()
//	addr, err := client.PredictAddress(owners, threshold, salt)
package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/pkg/safe"
)

// deployedSafe holds the state of a mock-deployed Safe.
type deployedSafe struct {
	owners    []common.Address
	threshold uint8
	txHash    common.Hash
	block     uint64
	gasUsed   uint64
}

// Client is an in-memory Safe client that implements safe.Client.
// Predicted addresses are computed using the real CREATE2 math.
// Deployments are recorded in memory. No network, no gas, no chain.
type Client struct {
	mu       sync.RWMutex
	deployed map[common.Address]*deployedSafe
	blockNum uint64
}

// NewClient creates a new mock Client with no deployed Safes.
func NewClient() *Client {
	return &Client{
		deployed: make(map[common.Address]*deployedSafe),
		blockNum: 1,
	}
}

// PredictAddress computes the deterministic Safe address for the given
// configuration. Uses the real CREATE2 math; same as the production client.
func (c *Client) PredictAddress(
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (common.Address, error) {
	return predictAddress(owners, threshold, salt)
}

// IsDeployed returns true if a Safe has been deployed at the given address via this mock client.
func (c *Client) IsDeployed(_ context.Context, addr common.Address) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.deployed[addr]
	return ok, nil
}

// Deploy records a Safe deployment in memory and returns a synthetic result.
func (c *Client) Deploy(
	_ context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (*safe.DeployResult, error) {
	addr, err := predictAddress(owners, threshold, salt)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.deployed[addr]; exists {
		return nil, fmt.Errorf("%w: %s", safe.ErrAddressAlreadyDeployed, addr.Hex())
	}

	txHash := syntheticTxHash(addr, c.blockNum)
	c.blockNum++

	c.deployed[addr] = &deployedSafe{
		owners:    owners,
		threshold: threshold,
		txHash:    txHash,
		block:     c.blockNum,
		gasUsed:   200_000,
	}

	return &safe.DeployResult{
		SafeAddress: addr,
		TxHash:      txHash,
		BlockNumber: c.blockNum,
		GasUsed:     200_000,
	}, nil
}

// SubmitDeployment records the deployment and returns a synthetic tx hash.
// In the mock, submission and mining are instantaneous.
func (c *Client) SubmitDeployment(
	ctx context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (common.Hash, error) {
	result, err := c.Deploy(ctx, owners, threshold, salt)
	if err != nil {
		return common.Hash{}, err
	}

	return result.TxHash, nil
}

// WaitForDeployment returns the result of a previously submitted deployment.
// In the mock, the result is available immediately.
func (c *Client) WaitForDeployment(
	_ context.Context,
	owners []common.Address,
	threshold uint8,
	salt []byte,
	txHash common.Hash,
) (*safe.DeployResult, error) {
	addr, err := predictAddress(owners, threshold, salt)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	s, ok := c.deployed[addr]
	if !ok {
		return nil, fmt.Errorf("no deployment found for address %s", addr.Hex())
	}
	if s.txHash != txHash {
		return nil, fmt.Errorf("tx hash mismatch for address %s", addr.Hex())
	}

	return &safe.DeployResult{
		SafeAddress: addr,
		TxHash:      s.txHash,
		BlockNumber: s.block,
		GasUsed:     s.gasUsed,
	}, nil
}

// Close is a no-op for the mock client.
func (c *Client) Close() {}

// Reset clears all deployed Safes.
func (c *Client) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deployed = make(map[common.Address]*deployedSafe)
	c.blockNum = 1
}

// DeployedAddresses returns all addresses that have been deployed via this mock client.
func (c *Client) DeployedAddresses() []common.Address {
	c.mu.RLock()
	defer c.mu.RUnlock()

	addrs := make([]common.Address, 0, len(c.deployed))
	for addr := range c.deployed {
		addrs = append(addrs, addr)
	}

	return addrs
}
