// Package txmanager handles the transaction lifecycle for Safe proxy deployments,
// including gas estimation, nonce coordination, signing, broadcasting, and receipt confirmation.
package txmanager

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/spazzle-io/safekit/internal/predict"
	"github.com/spazzle-io/safekit/internal/versions"
	"github.com/spazzle-io/safekit/pkg/signer"
)

// defaultDeployTimeout is how long Deploy will wait for a transaction to be mined before giving up.
const defaultDeployTimeout = 5 * time.Minute

type TxManager struct {
	Client   *ethclient.Client
	Signer   signer.Signer
	nonceMu  sync.Mutex
	nonce    uint64
	hasNonce bool
}

// Result is returned after a successful Safe deployment.
type Result struct {
	// SafeAddress is the address of the newly deployed Safe proxy.
	SafeAddress common.Address

	// TxHash is the hash of the deployment transaction.
	TxHash common.Hash

	// BlockNumber is the block in which the deployment was mined.
	BlockNumber uint64

	// GasUsed is the actual gas consumed by the deployment transaction.
	GasUsed uint64
}

// Options configures a deployment.
type Options struct {
	// GasMultiplier is applied to the estimated gas.
	// Increase if deployments fail with "out of gas" on congested chains.
	GasMultiplier float64

	// Timeout is how long to wait for the transaction to be mined.
	Timeout time.Duration
}

func (o *Options) gasMultiplier() float64 {
	if o == nil || o.GasMultiplier <= 0 {
		return defaultGasMultiplier
	}
	return o.GasMultiplier
}

func (o *Options) timeout() time.Duration {
	if o == nil || o.Timeout <= 0 {
		return defaultDeployTimeout
	}
	return o.Timeout
}

func New(client *ethclient.Client, signer signer.Signer) *TxManager {
	return &TxManager{
		Client: client,
		Signer: signer,
	}
}

// Submit builds and submits the deployment transaction, returning the
// transaction hash immediately without waiting for it to be mined.
// Use Wait to retrieve the result once mined.
func (d *TxManager) Submit(
	ctx context.Context,
	deployment versions.Deployment,
	chainID *big.Int,
	isL2 bool,
	input predict.Input,
	opts *Options,
) (common.Hash, error) {
	calldata, err := buildCalldata(input, deployment, chainID, isL2)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to build calldata: %w", err)
	}

	factory, err := deployment.ProxyFactory(chainID)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get proxy factory: %w", err)
	}

	predictedAddr, err := predict.Address(input, deployment, chainID, isL2)
	if err != nil {
		return common.Hash{}, fmt.Errorf("address prediction failed: %w", err)
	}

	gasLimit, err := estimateGas(ctx, d.Client, d.Signer.Address(), factory.Address, calldata, opts.gasMultiplier())
	if err != nil {
		if alreadyDeployed, checkErr := isDeployed(ctx, d.Client, predictedAddr); checkErr == nil && alreadyDeployed {
			return common.Hash{}, fmt.Errorf("%w: %s", ErrAddressAlreadyDeployed, predictedAddr.Hex())
		}
		return common.Hash{}, err
	}

	tip, err := suggestGasTipCap(ctx, d.Client)
	if err != nil {
		return common.Hash{}, err
	}

	head, err := d.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get latest block header: %w", err)
	}

	var gasPrice *big.Int
	if head.BaseFee == nil {
		gasPrice, err = suggestGasPrice(ctx, d.Client)
		if err != nil {
			return common.Hash{}, err
		}
	}

	d.nonceMu.Lock()
	defer d.nonceMu.Unlock()

	nonce, err := d.nextNonce(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	tx := buildTxWithNonce(chainID, factory.Address, calldata, nonce, gasLimit, tip, head.BaseFee, gasPrice)

	signed, err := d.Signer.SignTx(ctx, tx, chainID)
	if err != nil {
		d.hasNonce = false
		return common.Hash{}, err
	}

	if err := d.Client.SendTransaction(ctx, signed); err != nil {
		d.hasNonce = false
		return common.Hash{}, fmt.Errorf("failed to submit transaction: %w", err)
	}

	return signed.Hash(), nil
}

// Wait waits for a previously submitted deployment transaction to be mined,
// verifies the deployed address matches the prediction, and returns the result.
func (d *TxManager) Wait(
	ctx context.Context,
	deployment versions.Deployment,
	chainID *big.Int,
	isL2 bool,
	input predict.Input,
	txHash common.Hash,
) (*Result, error) {
	predictedAddr, err := predict.Address(input, deployment, chainID, isL2)
	if err != nil {
		return nil, fmt.Errorf("address prediction failed: %w", err)
	}

	receipt, err := waitForReceipt(ctx, d.Client, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return nil, fmt.Errorf("%w: %s", ErrTransactionReverted, txHash.Hex())
	}

	actualAddr, err := extractDeployedAddress(receipt, deployment, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract deployed address: %w", err)
	}

	if actualAddr != predictedAddr {
		return nil, &DeploymentMismatchError{
			PredictedAddress: predictedAddr,
			ActualAddress:    actualAddr,
			TxHash:           txHash,
			BlockNumber:      receipt.BlockNumber.Uint64(),
		}
	}

	return &Result{
		SafeAddress: actualAddr,
		TxHash:      txHash,
		BlockNumber: receipt.BlockNumber.Uint64(),
		GasUsed:     receipt.GasUsed,
	}, nil
}

// Deploy submits a deployment transaction and waits for it to be mined.
// Use Submit and Wait if you need to deploy in a non-blocking fashion.
func (d *TxManager) Deploy(
	ctx context.Context,
	deployment versions.Deployment,
	chainID *big.Int,
	isL2 bool,
	input predict.Input,
	opts *Options,
) (*Result, error) {
	predictedAddr, err := predict.Address(input, deployment, chainID, isL2)
	if err != nil {
		return nil, fmt.Errorf("address prediction failed: %w", err)
	}

	deployed, err := isDeployed(ctx, d.Client, predictedAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to check deployment status: %w", err)
	}
	if deployed {
		return nil, fmt.Errorf("%w: %s", ErrAddressAlreadyDeployed, predictedAddr.Hex())
	}

	txHash, err := d.Submit(ctx, deployment, chainID, isL2, input, opts)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, opts.timeout())
	defer cancel()

	return d.Wait(timeoutCtx, deployment, chainID, isL2, input, txHash)
}

func (d *TxManager) nextNonce(ctx context.Context) (uint64, error) {
	if !d.hasNonce {
		n, err := d.Client.PendingNonceAt(ctx, d.Signer.Address())
		if err != nil {
			return 0, err
		}

		d.nonce = n
		d.hasNonce = true

		return d.nonce, nil
	}

	d.nonce++
	return d.nonce, nil
}

func waitForReceipt(
	ctx context.Context,
	client *ethclient.Client,
	txHash common.Hash,
) (*types.Receipt, error) {
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %s", ErrDeployTimeout, txHash.Hex())
		case <-time.After(2 * time.Second):
			// continue polling
		}
	}
}

// buildCalldata ABI-encodes the createProxyWithNonce call for the factory.
func buildCalldata(
	input predict.Input,
	deployment versions.Deployment,
	chainID *big.Int,
	isL2 bool,
) ([]byte, error) {
	factory, err := deployment.ProxyFactory(chainID)
	if err != nil {
		return nil, err
	}

	singleton, err := deployment.Singleton(chainID, isL2)
	if err != nil {
		return nil, err
	}

	initializer, err := buildInitializer(input)
	if err != nil {
		return nil, err
	}

	saltNonce := deriveSaltNonce(input.Salt)
	saltNonceInt := new(big.Int).SetBytes(saltNonce[:])

	calldata, err := factory.ABI.Pack(
		"createProxyWithNonce",
		singleton.Address,
		initializer,
		saltNonceInt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack createProxyWithNonce: %w", err)
	}

	return calldata, nil
}

func buildInitializer(input predict.Input) ([]byte, error) {
	return predict.BuildInitializer(input.Owners, input.Threshold)
}

func deriveSaltNonce(salt []byte) [32]byte {
	return predict.DeriveSaltNonce(salt)
}

func buildTxWithNonce(
	chainID *big.Int,
	to common.Address,
	data []byte,
	nonce uint64,
	gasLimit uint64,
	tip *big.Int,
	baseFee *big.Int,
	gasPrice *big.Int,
) *types.Transaction {
	if baseFee == nil {
		return types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &to,
			Value:    big.NewInt(0),
			Data:     data,
		})
	}

	maxFee := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		tip,
	)

	return types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Gas:       gasLimit,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      data,
	})
}

// isDeployed returns true if a contract is already deployed at the given address.
func isDeployed(ctx context.Context, client *ethclient.Client, addr common.Address) (bool, error) {
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, err
	}

	return len(code) > 0, nil
}

// extractDeployedAddress reads the ProxyCreation event from the deployment receipt to
// get the actual address the factory deployed to.
func extractDeployedAddress(
	receipt *types.Receipt,
	deployment versions.Deployment,
	chainID *big.Int,
) (common.Address, error) {
	factory, err := deployment.ProxyFactory(chainID)
	if err != nil {
		return common.Address{}, err
	}

	proxyCreationEvent, ok := factory.ABI.Events["ProxyCreation"]
	if !ok {
		return common.Address{}, fmt.Errorf("ProxyCreation event not found in factory ABI")
	}

	for _, log := range receipt.Logs {
		if len(log.Topics) == 0 {
			continue
		}
		if log.Topics[0] != proxyCreationEvent.ID {
			continue
		}

		if len(log.Topics) > 1 {
			// indexed: address is in Topics[1]
			return common.BytesToAddress(log.Topics[1].Bytes()), nil
		}

		// non-indexed: address is in log.Data
		if len(log.Data) >= 32 {
			return common.BytesToAddress(log.Data[12:32]), nil
		}
	}

	return common.Address{}, errors.New("ProxyCreation event not found in transaction logs")
}
