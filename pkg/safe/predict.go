package safe

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/internal/predict"
)

func (c *client) PredictAddress(
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

func (c *client) IsDeployed(ctx context.Context, addr common.Address) (bool, error) {
	return c.txManager.IsDeployed(ctx, addr)
}
