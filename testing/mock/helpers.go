package mock

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/spazzle-io/safekit/pkg/version"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/internal/predict"
	"github.com/spazzle-io/safekit/internal/versions"

	// use v1.4.1 as the mock's default version
	_ "github.com/spazzle-io/safekit/internal/versions/v1_4_1"
)

// predictAddress computes the real CREATE2 address for the given config.
// Uses v1.4.1 on Ethereum mainnet as the mock's fixed context.
func predictAddress(owners []common.Address, threshold uint8, salt []byte) (common.Address, error) {
	deployment, err := versions.Get(version.V141)
	if err != nil {
		return common.Address{}, err
	}

	return predict.Address(
		predict.Input{
			Owners:    owners,
			Threshold: threshold,
			Salt:      salt,
		},
		deployment,
		big.NewInt(1),
		false,
	)
}

// syntheticTxHash generates a deterministic fake transaction hash from
// a Safe address and block number. Unique per deployment, stable across test runs.
func syntheticTxHash(addr common.Address, block uint64) common.Hash {
	h := sha256.New()
	h.Write(addr.Bytes())

	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, block)

	h.Write(blockBytes)
	return common.BytesToHash(h.Sum(nil))
}
