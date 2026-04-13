package safe

import (
	"fmt"

	"github.com/spazzle-io/safekit/internal/txmanager"

	"github.com/spazzle-io/safekit/internal/versions"
)

// DeploymentMismatchError is returned when the deployed Safe address does not match the predicted address.
// This indicates a bug in safekit. Please open an issue at github.com/spazzle-io/safekit with the details.
type DeploymentMismatchError = txmanager.DeploymentMismatchError

var (
	// ErrUnknownVersion is returned when the version string passed to New() is not a version safekit knows about.
	ErrUnknownVersion = versions.ErrUnknownVersion

	// ErrVersionNotOnChain is returned when the requested Safe version exists but has not been deployed on the
	// configured chain yet. This is expected for newer versions with limited chain coverage. Check
	// [Safe's supported networks] for the latest coverage.
	//
	// [Safe's supported networks]: https://docs.safe.global/advanced/smart-account-supported-networks
	ErrVersionNotOnChain = versions.ErrVersionNotOnChain

	// ErrAddressAlreadyDeployed is returned when the predicted Safe address already has a contract deployed at it.
	ErrAddressAlreadyDeployed = txmanager.ErrAddressAlreadyDeployed

	// ErrDeployTimeout is returned when the deployment transaction is not mined within the configured timeout.
	ErrDeployTimeout = txmanager.ErrDeployTimeout

	// ErrTransactionReverted is returned when the deployment transaction was mined but reverted on-chain.
	ErrTransactionReverted = txmanager.ErrTransactionReverted

	// ErrInvalidThreshold is returned when the threshold exceeds the number of owners or is zero.
	ErrInvalidThreshold = fmt.Errorf("invalid threshold")

	// ErrEmptyOwners is returned when the owners list is empty.
	ErrEmptyOwners = fmt.Errorf("owners list cannot be empty")

	// ErrDuplicateOwner is returned when the owners list contains duplicate addresses.
	ErrDuplicateOwner = fmt.Errorf("duplicate owner address")

	// ErrZeroAddressOwner is returned when the owners list contains the zero address.
	ErrZeroAddressOwner = fmt.Errorf("zero address cannot be an owner")
)
