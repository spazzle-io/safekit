package versions

import (
	"math/big"
	"sync"

	"github.com/spazzle-io/safekit/pkg/chain"
)

// BaseDeployment implements the Deployment interface for a specific Safe version.
type BaseDeployment struct {
	version           Version
	safeJSON          []byte
	safeL2JSON        []byte
	proxyFactoryJSON  []byte
	proxyCreationCode []byte

	mu      sync.RWMutex
	factory map[string]*ParsedDeployment
	safe    map[string]*ParsedDeployment
	safeL2  map[string]*ParsedDeployment
}

// NewBaseDeployment creates a BaseDeployment for a specific Safe version.
func NewBaseDeployment(
	v Version,
	jsonSafe []byte,
	jsonSafeL2 []byte,
	jsonFactory []byte,
	proxyCreationCode []byte,
) Deployment {
	return &BaseDeployment{
		version:           v,
		safeJSON:          jsonSafe,
		safeL2JSON:        jsonSafeL2,
		proxyFactoryJSON:  jsonFactory,
		proxyCreationCode: proxyCreationCode,
		factory:           make(map[string]*ParsedDeployment),
		safe:              make(map[string]*ParsedDeployment),
		safeL2:            make(map[string]*ParsedDeployment),
	}
}

func (d *BaseDeployment) Version() Version {
	return d.version
}

func (d *BaseDeployment) ProxyFactory(chainID *big.Int) (*ParsedDeployment, error) {
	return d.loadCached(d.factory, d.proxyFactoryJSON, d.resolveChainID(chainID))
}

func (d *BaseDeployment) Singleton(chainID *big.Int, isL2 bool) (*ParsedDeployment, error) {
	if isL2 {
		return d.loadCached(d.safeL2, d.safeL2JSON, d.resolveChainID(chainID))
	}
	return d.loadCached(d.safe, d.safeJSON, d.resolveChainID(chainID))
}

func (d *BaseDeployment) ProxyCreationCode() []byte {
	return d.proxyCreationCode
}

func (d *BaseDeployment) resolveChainID(chainID *big.Int) *big.Int {
	c, err := chain.Lookup(chainID)
	if err != nil {
		return chainID
	}

	if c.ForksChainID() != nil {
		return c.ForksChainID()
	}

	return c.ID
}

func (d *BaseDeployment) loadCached(
	cache map[string]*ParsedDeployment,
	jsonBytes []byte,
	chainID *big.Int,
) (*ParsedDeployment, error) {
	key := chainID.String()

	d.mu.RLock()
	if p, ok := cache[key]; ok {
		d.mu.RUnlock()
		return p, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	if p, ok := cache[key]; ok {
		return p, nil
	}

	parsed, err := Load(jsonBytes, chainID)
	if err != nil {
		return nil, err
	}

	cache[key] = parsed
	return parsed, nil
}
