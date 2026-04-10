package safe

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var zeroAddress = common.Address{}

func validateSafeConfig(owners []common.Address, threshold uint8) error {
	if len(owners) == 0 {
		return ErrEmptyOwners
	}

	seen := make(map[common.Address]struct{}, len(owners))
	for _, o := range owners {
		if o == zeroAddress {
			return ErrZeroAddressOwner
		}
		if _, exists := seen[o]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateOwner, o.Hex())
		}
		seen[o] = struct{}{}
	}

	if threshold == 0 || int(threshold) > len(owners) {
		return fmt.Errorf("%w: %d of %d", ErrInvalidThreshold, threshold, len(owners))
	}

	return nil
}
