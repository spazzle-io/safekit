// Package safe is the primary entry point for safekit. It provides a Client
// for predicting Safe addresses and deploying Safe multisig wallets on
// EVM-compatible chains.
//
// Basic usage:
//
//	client, err := safe.New(safe.Options{
//	    Chain:   chain.Polygon,
//	    RPC:     os.Getenv("RPC_URL"),
//	    Signer:  mySigner,
//	    Version: version.V141,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	addr, err := client.PredictAddress(owners, threshold, salt)
//	result, err := client.Deploy(ctx, owners, threshold, salt)
package safe

import (
	"context"
	"errors"
	"fmt"

	"github.com/spazzle-io/safekit/internal/deployer"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spazzle-io/safekit/internal/versions"
	"github.com/spazzle-io/safekit/pkg/chain"
)

// Deployer is the interface implemented by Client.
// Depend on this interface in your application code rather than
// *Client directly — it makes testing with mock.Client straightforward.
type Deployer interface {
	PredictAddress(owners []common.Address, threshold uint8, salt []byte) (common.Address, error)
	IsDeployed(ctx context.Context, addr common.Address) (bool, error)
	Deploy(ctx context.Context, owners []common.Address, threshold uint8, salt []byte) (*DeployResult, error)
	SubmitDeployment(ctx context.Context, owners []common.Address, threshold uint8, salt []byte) (common.Hash, error)
	WaitForDeployment(ctx context.Context, owners []common.Address, threshold uint8, salt []byte, txHash common.Hash) (*DeployResult, error)
	Close()
}

var _ Deployer = (*Client)(nil)

// Client provides Safe address prediction and deployment for a specific
// chain and Safe version. Create one with New() and close it with Close()
// when done.
//
// A Client is safe for concurrent use. Multiple goroutines may call any Client method simultaneously.
//
// Each Client manages the nonce sequence for its signer wallet on its chain. If any other process,
// script, or Client instance submits transactions from the same wallet on the same chain while an existing Client
// is running, nonce conflicts may occur. For safe concurrent operation, ensure only one Client per wallet
// per chain is active at any time.
type Client struct {
	chain      *chain.Chain
	deployer   *deployer.Deployer
	deployment versions.Deployment
	opts       *Options
}

// New creates a new Client from the given options.
// It establishes a connection to the RPC endpoint and loads the
// Safe contract metadata for the configured version and chain.
func New(opts Options) (*Client, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	deployment, err := versions.Get(opts.Version)
	if err != nil {
		if errors.Is(err, versions.ErrUnknownVersion) {
			return nil, fmt.Errorf("%w: %w", ErrUnknownVersion, err)
		}
		return nil, err
	}

	if _, err := deployment.ProxyFactory(opts.Chain.ID); err != nil {
		if errors.Is(err, versions.ErrVersionNotOnChain) {
			return nil, fmt.Errorf("%w: version %s is not deployed on chain %s",
				ErrVersionNotOnChain, opts.Version, opts.Chain.Name)
		}
		return nil, err
	}

	if _, err := deployment.Singleton(opts.Chain.ID, opts.Chain.IsL2); err != nil {
		if errors.Is(err, versions.ErrVersionNotOnChain) {
			return nil, fmt.Errorf("%w: version %s is not deployed on chain %s",
				ErrVersionNotOnChain, opts.Version, opts.Chain.Name)
		}
		return nil, err
	}

	eth, err := opts.dialRPC()
	if err != nil {
		return nil, err
	}

	return &Client{
		chain:      opts.Chain,
		deployer:   deployer.NewDeployer(eth, opts.Signer),
		deployment: deployment,
		opts:       &opts,
	}, nil
}

// Close releases the RPC connection and zeroes the signer's key material.
// Always call this when done with a Client.
func (c *Client) Close() {
	if c.deployer.Client != nil {
		c.deployer.Client.Close()
	}

	if c.deployer.Signer != nil {
		c.deployer.Signer.Close()
	}
}
