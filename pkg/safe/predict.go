package safe

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/internal/predict"
)

// PredictAddress computes the deterministic address a Safe will be deployed
// to without making a network call. The same inputs on the same chain always
// produce the same address.
//
// Use this to get an address you can fund or reference before the Safe
// exists on-chain. Call IsDeployed to check whether it has been deployed yet.
func (c *Client) PredictAddress(
	owners []common.Address,
	threshold uint8,
	salt []byte,
) (common.Address, error) {
	if err := validateSafeConfig(owners, threshold); err != nil {
		return common.Address{}, err
	}

	addr, err := predict.Address(
		predict.Input{
			Owners:    owners,
			Threshold: threshold,
			Salt:      salt,
		},
		c.deployment,
		c.chain.ID,
		c.chain.IsL2,
	)
	if err != nil {
		return common.Address{}, err
	}

	return addr, nil
}
