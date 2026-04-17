package safe

import (
	"fmt"
	"time"

	"github.com/spazzle-io/safekit/pkg/nonce"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/spazzle-io/safekit/internal/versions"
	"github.com/spazzle-io/safekit/pkg/chain"
	"github.com/spazzle-io/safekit/pkg/signer"

	_ "github.com/spazzle-io/safekit/internal/versions/v1_3_0"
	_ "github.com/spazzle-io/safekit/internal/versions/v1_4_1"
	_ "github.com/spazzle-io/safekit/internal/versions/v1_5_0"
)

const (
	defaultGasMultiplier       = 1.2
	defaultDeployTimeout       = 2 * time.Minute
	defaultReceiptPollInterval = 2 * time.Second
)

// Options configures a Safe client.
type Options struct {
	// Chain is the EVM chain to operate on.
	// Use the constants from pkg/chain e.g. chain.Ethereum, chain.Polygon.
	// Required.
	Chain *chain.Chain

	// Client is the Ethereum JSON-RPC client used to interact with the chain.
	Client *ethclient.Client

	// Signer is the admin wallet that pays for gas.
	// It is never added as an owner of deployed Safes.
	// Required.
	Signer signer.Signer

	// Version is the Safe contract version to deploy.
	// Use the constants from pkg/version e.g. version.V141.
	// Required.
	Version versions.Version

	// NonceManager controls how nonces are assigned across concurrent deployments.
	// If nil, a local in-memory manager is created automatically with sensible defaults.
	// Optional.
	NonceManager nonce.Manager

	// DeployTimeout is how long Deploy will wait for a transaction to be mined.
	// Defaults to 2 minutes.
	DeployTimeout time.Duration

	// GasMultiplier is applied to the estimated gas to give headroom.
	// Defaults to 1.2. Increase if deployments fail with out-of-gas errors on congested chains.
	GasMultiplier float64

	// ReceiptPollInterval is how often safekit checks whether a submitted transaction has been mined.
	// Defaults to 2 seconds. Increase if fast deployment confirmation is not required to reduce RPC usage.
	ReceiptPollInterval time.Duration
}

func (o *Options) validate() error {
	if o.Chain == nil {
		return fmt.Errorf("chain is required")
	}

	if o.Client == nil {
		return fmt.Errorf("client is required")
	}

	if o.Signer == nil {
		return fmt.Errorf("signer is required")
	}

	if o.Version == "" {
		return fmt.Errorf("version is required")
	}

	return nil
}

func (o *Options) deployTimeout() time.Duration {
	if o.DeployTimeout <= 0 {
		return defaultDeployTimeout
	}
	return o.DeployTimeout
}

func (o *Options) gasMultiplier() float64 {
	if o.GasMultiplier <= 0 {
		return defaultGasMultiplier
	}
	return o.GasMultiplier
}

func (o *Options) receiptPollInterval() time.Duration {
	if o.ReceiptPollInterval <= 0 {
		return defaultReceiptPollInterval
	}
	return o.ReceiptPollInterval
}
