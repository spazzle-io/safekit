// Package predict computes the deterministic address a Safe will be deployed
// to before it exists on-chain. This is pure computation with no network calls,
// no transactions, no gas. The same inputs for the same chain always produce the same address.
package predict

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/spazzle-io/safekit/internal/versions"
)

// Input holds everything needed to predict a Safe's address.
// The same Input on the same chain always produces the same address.
type Input struct {
	// Owners is the list of addresses that will control the Safe.
	// Must contain at least one address, no duplicates, no zero address.
	Owners []common.Address

	// Threshold is the minimum number of owner signatures required to
	// execute a transaction. Must be >= 1 and <= len(Owners).
	Threshold uint8

	// Salt is arbitrary bytes used to differentiate deployments that share
	// the same owners and threshold. The same salt + owners + threshold
	// always produces the same address.
	Salt []byte
}

// Address computes the deterministic address for a Safe with the given
// configuration on the given chain and Safe version.
//
// This is a pure function. It makes no network calls and has no side effects.
func Address(
	input Input,
	deployment versions.Deployment,
	chainID *big.Int,
	isL2 bool,
) (common.Address, error) {
	if chainID == nil {
		return common.Address{}, fmt.Errorf("chainID cannot be nil")
	}

	factory, err := deployment.ProxyFactory(chainID)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get proxy factory: %w", err)
	}

	singleton, err := deployment.Singleton(chainID, isL2)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get singleton: %w", err)
	}

	initializer, err := BuildInitializer(input.Owners, input.Threshold)
	if err != nil {
		return common.Address{}, err
	}

	saltNonce := DeriveSaltNonce(input.Salt)

	salt := crypto.Keccak256(
		initializerHash(initializer),
		saltNonce[:],
	)

	initCode := buildInitCode(deployment.ProxyCreationCode(), singleton.Address)

	address := computeCreate2Address(factory.Address, salt, initCode)

	return address, nil
}

func DeriveSaltNonce(salt []byte) [32]byte {
	var nonce [32]byte
	hash := crypto.Keccak256(salt)
	copy(nonce[:], hash)

	return nonce
}

// buildInitCode concatenates the proxy creation bytecode with the
// ABI-encoded singleton address to form the CREATE2 initCode.
func buildInitCode(proxyCreationCode []byte, singleton common.Address) []byte {
	// ABI encoding of an address is left-padded to 32 bytes
	paddedSingleton := common.LeftPadBytes(singleton.Bytes(), 32)
	initCode := make([]byte, len(proxyCreationCode)+len(paddedSingleton))
	copy(initCode, proxyCreationCode)
	copy(initCode[len(proxyCreationCode):], paddedSingleton)

	return initCode
}

// computeCreate2Address runs the EIP-1014 CREATE2 address derivation formula:
//
//	keccak256(0xff ++ factory ++ salt ++ keccak256(initCode))[12:]
func computeCreate2Address(factory common.Address, salt []byte, initCode []byte) common.Address {
	raw := crypto.Keccak256(
		[]byte{0xff},
		factory.Bytes(),
		salt,
		crypto.Keccak256(initCode),
	)

	// the address is the last 20 bytes of the 32-byte hash
	return common.BytesToAddress(raw[12:])
}
