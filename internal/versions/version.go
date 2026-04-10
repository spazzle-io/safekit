// Package versions manages Safe contract deployment metadata for each
// supported Safe version.
//
// Adding a new Safe version means dropping a new sub-package with the
// relevant JSON files and an embed.go that registers itself. Nothing
// else needs to change. See CONTRIBUTING.md for the full steps.
package versions

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Version is a Safe contract version string.
type Version string

const (
	Version130 Version = "1.3.0"
	Version141 Version = "1.4.1"
	Version150 Version = "1.5.0"
)

// ParsedDeployment is the clean, usable result of loading a Safe deployment
// JSON for a specific chain.
type ParsedDeployment struct {
	// Address is the contract address on the requested chain, already
	// resolved from the canonical/eip155 preference in the JSON.
	Address common.Address

	// ABI is parsed and ready for encoding calls.
	ABI abi.ABI
}

// Deployment is the interface every versioned sub-package must implement.
// It's intentionally small — just enough to get addresses and ABIs.
// If you're adding a new Safe version, this is the contract you need to fulfil.
type Deployment interface {
	// Version returns the Safe version this implementation covers.
	Version() Version

	// ProxyFactory returns the parsed factory deployment for the given chain.
	// The factory is what you call to deploy a new Safe — it's the same
	// contract regardless of whether the chain is L1 or L2.
	ProxyFactory(chainID *big.Int) (*ParsedDeployment, error)

	// Singleton returns the parsed Safe singleton for the given chain.
	// isL2 controls which variant is returned — SafeL2.sol for L2 chains
	// which emits richer events, Safe.sol for L1 chains.
	Singleton(chainID *big.Int, isL2 bool) (*ParsedDeployment, error)

	// ProxyCreationCode returns the bytecode used by the factory to deploy
	// each new Safe proxy via CREATE2. This is required for deterministic
	// address prediction.
	ProxyCreationCode() []byte
}

var (
	mu       sync.RWMutex
	registry = map[Version]Deployment{}
)

var (
	// ErrUnknownVersion is returned when the version string is not
	// registered in safekit at all.
	ErrUnknownVersion = errors.New("unknown Safe version")

	// ErrVersionNotOnChain is returned when the version is valid but
	// has not been deployed on the requested chain yet.
	ErrVersionNotOnChain = errors.New("version not deployed on this chain")
)

// Register adds a version implementation to the registry. Each versioned
// sub-package calls this from its init() function so registration is
// automatic on import.
func Register(d Deployment) {
	mu.Lock()
	defer mu.Unlock()

	registry[d.Version()] = d
}

// Get returns the Deployment implementation for the requested version.
// Returns an error if the version hasn't been registered.
func Get(v Version) (Deployment, error) {
	mu.RLock()
	defer mu.RUnlock()

	d, ok := registry[v]
	if !ok {
		return nil, fmt.Errorf("%w: %q: Supported versions are: %v", ErrUnknownVersion, v, supported())
	}

	return d, nil
}

// supported returns a human-readable list of registered versions for use
// in error messages. Called with the read lock already held.
func supported() []Version {
	vs := make([]Version, 0, len(registry))
	for v := range registry {
		vs = append(vs, v)
	}

	return vs
}
