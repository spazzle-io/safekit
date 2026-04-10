package deployer

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// ErrAddressAlreadyDeployed is returned when the predicted Safe address already has a contract deployed at it.
	ErrAddressAlreadyDeployed = errors.New("address already has a deployed contract")

	// ErrDeployTimeout is returned when the deployment transaction is not mined within the configured timeout.
	ErrDeployTimeout = errors.New("deployment timed out waiting to be mined")

	// ErrTransactionReverted is returned when the deployment transaction was mined but reverted on-chain.
	ErrTransactionReverted = errors.New("transaction reverted on-chain")
)

// DeploymentMismatchError is returned when the deployed Safe address does
// not match the predicted address.
type DeploymentMismatchError struct {
	PredictedAddress common.Address
	ActualAddress    common.Address
	TxHash           common.Hash
	BlockNumber      uint64
}

func (e *DeploymentMismatchError) Error() string {
	return fmt.Sprintf(
		"Address mismatch. Predicted %s but deployed to %s (tx: %s, block: %d)."+
			"This is a bug in safekit, please open an issue at github.com/spazzle-io/safekit",
		e.PredictedAddress.Hex(),
		e.ActualAddress.Hex(),
		e.TxHash.Hex(),
		e.BlockNumber,
	)
}
