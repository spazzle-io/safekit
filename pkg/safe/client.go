// Package safe is the primary entry point for safekit. It provides a client
// for predicting Safe addresses and deploying Safe multisig wallets on
// EVM-compatible chains.
//
// Basic usage:
//
//	eth, err := safe.Dial(os.Getenv("RPC_URL"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer eth.Close()
//
//	client, err := safe.New(safe.Options{
//	    Chain:   chain.Polygon,
//	    Client:  eth,
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
	internalnonce "github.com/spazzle-io/safekit/internal/nonce"
	"github.com/spazzle-io/safekit/pkg/nonce/local"

	"github.com/spazzle-io/safekit/internal/txmanager"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spazzle-io/safekit/internal/versions"
	"github.com/spazzle-io/safekit/pkg/chain"
)

// Client is the interface implemented by the safekit client.
//
// Client provides Safe address prediction and deployment for a specific chain and Safe version.
// Create one with New() and close it with Close() when done.
//
// A client is safe for concurrent use. Multiple goroutines may call any client method simultaneously.
//
// Each client manages the nonce sequence for its signer on its chain. By default, a single-process local
// nonce manager is used. If multiple processes share the same signer on the same chain, provide a distributed
// NonceManager via Options. See pkg/nonce/redis for details.
type Client interface {
	// PredictAddress computes the deterministic address a Safe will be deployed to without making a network call.
	// The same inputs on the same chain always produce the same address.
	//
	// Use this to get an address you can fund or reference before the Safe exists on-chain.
	// Call IsDeployed to check whether it has been deployed yet.
	PredictAddress(owners []common.Address, threshold uint8, salt []byte) (common.Address, error)

	// IsDeployed returns true if a contract is already deployed at the given address.
	// Use this to check whether a predicted Safe address has been deployed yet.
	IsDeployed(ctx context.Context, addr common.Address) (bool, error)

	// Deploy predicts, deploys, and verifies a new Safe in one call.
	// It blocks until the deployment tx is mined, the context is cancelled, or the deployment timeout is reached.
	//
	// Returns context.Canceled if the context is cancelled before mining.
	// Returns ErrDeployTimeout if the transaction is not mined within the timeout.
	//
	// If you want to separate submission from confirmation, use SubmitDeployment to submit the transaction and get a
	// hash immediately, then WaitForDeployment to block until it mines. If you want fully non-blocking behaviour,
	// use SubmitDeployment and then poll with IsDeployed at your own pace rather than calling WaitForDeployment.
	//
	// The predicted address is verified against the actual deployed address.
	// If they differ, a [DeploymentMismatchError] is returned. This indicates a bug in safekit.
	// Please open an issue at github.com/spazzle-io/safekit with the details.
	//
	// The signer configured in Options pays for gas but is never added as an owner.
	// Pass the admin's address in owners explicitly if you want it to be an owner.
	Deploy(ctx context.Context, owners []common.Address, threshold uint8, salt []byte) (*DeployResult, error)

	// SubmitDeployment submits the deployment transaction and returns the transaction hash immediately without waiting
	// for it to be mined. This is useful when integrating with async job queues where you want to track the transaction
	// hash separately from the mining confirmation.
	//
	// Use [Client.WaitForDeployment] to block until the transaction mines, or poll [Client.IsDeployed] at your own pace
	// for fully non-blocking behaviour.
	//
	// The signer pays for gas but is never added as an owner unless explicitly added to the owners slice.
	SubmitDeployment(ctx context.Context, owners []common.Address, threshold uint8, salt []byte) (common.Hash, error)

	// WaitForDeployment waits for a previously submitted deployment transaction to be mined and returns the result.
	// It blocks until the deployment tx is mined, the context is cancelled, or the deployment timeout is reached.
	//
	// The predicted address is verified against the actual deployed address. If they differ,
	// a DeploymentMismatchError is returned. This indicates a bug in safekit.
	// Please open an issue at github.com/spazzle-io/safekit with the details.
	WaitForDeployment(ctx context.Context, owners []common.Address, threshold uint8, salt []byte, txHash common.Hash) (*DeployResult, error)

	// Close closes the signer.
	Close()
}

type client struct {
	chain      *chain.Chain
	txManager  *txmanager.TxManager
	deployment versions.Deployment
	opts       *Options
}

// New creates a new Client from the given options.
// It loads the Safe contract metadata for the configured version and chain, and initialises the nonce manager.
func New(opts Options) (Client, error) {
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

	nm := opts.NonceManager
	if nm == nil {
		nm = local.NewNonceManager(local.Options{})
	}

	if initable, ok := nm.(internalnonce.Initable); ok {
		if err := initable.Init(opts.Client, opts.Chain.ID, opts.Signer.Address()); err != nil {
			return nil, fmt.Errorf("failed to initialise nonce manager: %w", err)
		}
	}

	return &client{
		chain:      opts.Chain,
		txManager:  txmanager.New(opts.Client, opts.Signer, nm),
		deployment: deployment,
		opts:       &opts,
	}, nil
}

func (c *client) Close() {
	c.txManager.Close()
}
