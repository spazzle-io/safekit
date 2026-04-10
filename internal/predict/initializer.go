package predict

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var zeroAddress = common.Address{}

// setupABI is the ABI for Safe's setup() initializer function.
var setupABI abi.ABI

func init() {
	const setupABIJSON = `[{
		"type": "function",
		"name": "setup",
		"inputs": [
			{"name": "owners",          "type": "address[]"},
			{"name": "threshold",       "type": "uint256"},
			{"name": "to",              "type": "address"},
			{"name": "data",            "type": "bytes"},
			{"name": "fallbackHandler", "type": "address"},
			{"name": "paymentToken",    "type": "address"},
			{"name": "payment",         "type": "uint256"},
			{"name": "paymentReceiver", "type": "address"}
		],
		"outputs": []
	}]`

	var err error
	setupABI, err = abi.JSON(strings.NewReader(setupABIJSON))
	if err != nil {
		panic(fmt.Sprintf("failed to parse setup ABI: %v", err))
	}
}

// BuildInitializer ABI-encodes the Safe setup() call with the given owners and threshold.
func BuildInitializer(owners []common.Address, threshold uint8) ([]byte, error) {
	data, err := setupABI.Pack(
		"setup",
		owners,
		new(big.Int).SetUint64(uint64(threshold)),
		zeroAddress,   // to: no delegate call on setup
		[]byte{},      // data: empty, no delegate call
		zeroAddress,   // fallbackHandler: none
		zeroAddress,   // paymentToken: no payment
		big.NewInt(0), // payment: zero
		zeroAddress,   // paymentReceiver: none
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode setup calldata: %w", err)
	}

	return data, nil
}

// initializerHash returns keccak256 of the ABI-encoded setup() calldata.
func initializerHash(initializer []byte) []byte {
	return crypto.Keccak256(initializer)
}
