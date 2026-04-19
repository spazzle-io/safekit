// Package txmanager handles the transaction lifecycle for Safe proxy deployments,
// including gas estimation, nonce coordination, signing, broadcasting, and receipt confirmation.
package txmanager

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	pkgnonce "github.com/spazzle-io/safekit/pkg/nonce"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/spazzle-io/safekit/internal/predict"
	"github.com/spazzle-io/safekit/internal/versions"
	"github.com/spazzle-io/safekit/pkg/signer"
)

const (
	// defaultDeployTimeout is how long Deploy will wait for a transaction to be mined before giving up.
	defaultDeployTimeout = 2 * time.Minute

	// defaultReceiptPollInterval is how often waitForReceipt polls for a transaction receipt.
	defaultReceiptPollInterval = 2 * time.Second
)

type TxManager struct {
	client              *ethclient.Client
	signer              signer.Signer
	nonceManager        pkgnonce.Manager
	receiptPollInterval time.Duration
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

func New(
	client *ethclient.Client,
	signer signer.Signer,
	nm pkgnonce.Manager,
	receiptPollInterval time.Duration,
) *TxManager {
	if receiptPollInterval <= 0 {
		receiptPollInterval = defaultReceiptPollInterval
	}

	return &TxManager{
		client:              client,
		signer:              signer,
		nonceManager:        nm,
		receiptPollInterval: receiptPollInterval,
	}
}

// Submit builds and submits the deployment transaction, returning the transaction hash immediately without
// waiting for it to be mined. Use Wait to retrieve the result once mined.
func (t *TxManager) Submit(
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

	n, slot, err := t.nonceManager.Acquire(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to acquire nonce slot: %w", err)
	}

	deployed, err := t.IsDeployed(ctx, predictedAddr)
	if err != nil {
		slot.Reclaim()
		return common.Hash{}, fmt.Errorf("failed to determine if contract is deployed: %w", err)
	}

	if deployed {
		slot.Reuse()
		return common.Hash{}, fmt.Errorf("%w: %s", ErrAddressAlreadyDeployed, predictedAddr.Hex())
	}

	gasLimit, err := estimateGas(ctx, t.client, t.signer.Address(), factory.Address, calldata, opts.gasMultiplier())
	if err != nil {
		slot.Reuse()
		return common.Hash{}, fmt.Errorf("gas estimation failed: %w", err)
	}

	tip, err := suggestGasTipCap(ctx, t.client)
	if err != nil {
		slot.Reuse()
		return common.Hash{}, err
	}

	head, err := t.client.HeaderByNumber(ctx, nil)
	if err != nil {
		slot.Reuse()
		return common.Hash{}, fmt.Errorf("failed to get latest block header: %w", err)
	}

	var gasPrice *big.Int
	if head.BaseFee == nil {
		gasPrice, err = suggestGasPrice(ctx, t.client)
		if err != nil {
			slot.Reuse()
			return common.Hash{}, err
		}
	}

	tx := buildTxWithNonce(chainID, factory.Address, calldata, n, gasLimit, tip, head.BaseFee, gasPrice)

	signed, err := t.signer.SignTx(ctx, tx, chainID)
	if err != nil {
		slot.Reuse()
		return common.Hash{}, err
	}

	sendErr := t.client.SendTransaction(ctx, signed)
	if sendErr != nil {
		slot.Reclaim()
		return common.Hash{}, fmt.Errorf("failed to submit transaction: %w", sendErr)
	}

	slot.Commit()
	return signed.Hash(), nil
}

// Wait waits for a previously submitted deployment transaction to be mined,
// verifies the deployed address matches the prediction, and returns the result.
func (t *TxManager) Wait(
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

	receipt, err := t.waitForReceipt(ctx, txHash)
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
func (t *TxManager) Deploy(
	ctx context.Context,
	deployment versions.Deployment,
	chainID *big.Int,
	isL2 bool,
	input predict.Input,
	opts *Options,
) (*Result, error) {
	txHash, err := t.Submit(ctx, deployment, chainID, isL2, input, opts)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, opts.timeout())
	defer cancel()

	result, err := t.Wait(timeoutCtx, deployment, chainID, isL2, input, txHash)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", ErrDeployTimeout, txHash.Hex())
		}
		return nil, err
	}

	return result, nil
}

func (t *TxManager) IsDeployed(ctx context.Context, addr common.Address) (bool, error) {
	header, err := t.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get latest block header: %w", err)
	}

	code, err := t.client.CodeAt(ctx, addr, header.Number)
	if err != nil {
		return false, err
	}

	return len(code) > 0, nil
}

func (t *TxManager) Close() {
	t.signer.Close()
}

func (t *TxManager) waitForReceipt(
	ctx context.Context,
	txHash common.Hash,
) (*types.Receipt, error) {
	ticker := time.NewTicker(t.receiptPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			receipt, err := t.client.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
		}
	}
}

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
			// if indexed, address is in Topics[1]
			return common.BytesToAddress(log.Topics[1].Bytes()), nil
		}

		// if not indexed, address is in log.Data
		if len(log.Data) < 32 {
			return common.Address{}, fmt.Errorf(
				"ProxyCreation log data too short: %d bytes, want at least 32",
				len(log.Data),
			)
		}
		return common.BytesToAddress(log.Data[12:32]), nil
	}

	return common.Address{}, errors.New("ProxyCreation event not found in transaction logs")
}
