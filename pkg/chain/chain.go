// Package chain provides EVM chain definitions and a registry for looking
// them up. You bring your own RPC — this package only cares about the
// constants that make each chain unique for Safe deployments.
package chain

import "math/big"

// Chain represents an EVM-compatible network that safekit can deploy to.
// It carries only the constants needed for Safe operations.
type Chain struct {
	// ID is the EIP-155 chain identifier.
	ID *big.Int

	// Name is human-readable chain name.
	Name string

	// IsL2 determines which Safe singleton variant gets used during
	// deployment. L2 chains use SafeL2.sol. Others use Safe.sol.
	IsL2 bool

	// forksChainID is the chain ID whose Safe contract addresses should be used for this chain.
	// This is useful for local development chains that fork a known network.
	//
	// If nil, the chain's own ID is used for contract lookups.
	forksChainID *big.Int
}

// ForksChainID returns the chain ID whose contract addresses this chain uses for Safe deployments,
// or nil if no fork is configured.
func (c *Chain) ForksChainID() *big.Int {
	return c.forksChainID
}
