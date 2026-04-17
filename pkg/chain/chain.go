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

// Fork returns a copy of c configured as a fork of source.
// Use this when running a local development chain that forks a known network.
//
// Example: an Anvil instance forking Sepolia:
//
//	local, _ := chain.Lookup(big.NewInt(31337))
//	client, err := safe.New(safe.Options{
//	    Chain: local.Fork(chain.EthereumSepolia),
//	    ...
//	})
func (c *Chain) Fork(source *Chain) *Chain {
	cpy := *c
	cpy.forksChainID = source.ID
	return &cpy
}

// ForksChainID returns the chain ID whose contract addresses this chain uses for Safe deployments,
// or nil if no fork is configured.
func (c *Chain) ForksChainID() *big.Int {
	return c.forksChainID
}
