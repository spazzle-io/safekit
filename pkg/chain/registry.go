package chain

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]*Chain{}
)

func init() {
	for _, c := range []*Chain{
		// Local
		Local,

		// Ethereum
		Ethereum,
		Sepolia,

		// Polygon
		Polygon,
		PolygonZkEVM,
		PolygonAmoy,

		// Arbitrum
		ArbitrumOne,
		ArbitrumNova,
		ArbitrumSepolia,

		// Base
		Base,
		BaseSepolia,

		// Optimism
		Optimism,
		OptimismSepolia,

		// BNB Smart Chain
		BNBSmartChain,
		BNBSmartChainTestnet,
	} {
		registry[c.ID.String()] = c
	}
}

// Register adds a chain to the registry. If a chain with the same ID already
// exists it gets overwritten — so you can override one of our built-in chains
// with your own metadata if you need to.
//
// Call this once at startup before doing anything else with safekit.
//
//	chain.Register(&chain.Chain{
//	    ID:   big.NewInt(12345),
//	    Name: "My Private Chain",
//	    IsL2: false,
//	})
func Register(c *Chain) error {
	if c == nil {
		return fmt.Errorf("cannot register a nil chain")
	}

	if c.ID == nil {
		return fmt.Errorf("cannot register a chain with a nil ID")
	}

	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("cannot register a chain with an empty name")
	}

	mu.Lock()
	defer mu.Unlock()

	registry[c.ID.String()] = c
	return nil
}

// Lookup returns the chain for a given ID, or an error if it isn't registered.
// Use chain.Register to add chains that aren't built in.
func Lookup(id *big.Int) (*Chain, error) {
	if id == nil {
		return nil, fmt.Errorf("lookup called with nil ID")
	}

	mu.RLock()
	defer mu.RUnlock()

	c, ok := registry[id.String()]
	if !ok {
		return nil, fmt.Errorf("chain ID %s is not registered — use chain.Register() to add it", id)
	}

	return c, nil
}

// Fork registers c as a fork of source. Safe contract addresses are resolved from source
// while all other chain properties remain those of c.
//
// Use this when running a local development chain that forks a known network.
func Fork(c *Chain, source *Chain) (*Chain, error) {
	forked := *c
	forked.forksChainID = source.ID

	if err := Register(&forked); err != nil {
		return nil, err
	}

	return &forked, nil
}

// Deregister removes a chain from the registry by its chain ID.
// It is a no-op if the chain is not registered.
func Deregister(chainID *big.Int) {
	mu.Lock()
	defer mu.Unlock()

	delete(registry, chainID.String())
}
