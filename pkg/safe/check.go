package safe

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// IsDeployed returns true if a contract is already deployed at the given
// address. Use this to check whether a predicted Safe address has been
// deployed yet.
func (c *Client) IsDeployed(ctx context.Context, addr common.Address) (bool, error) {
	code, err := c.deployer.Client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check deployment status: %w", err)
	}

	return len(code) > 0, nil
}
