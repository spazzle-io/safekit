//go:build integration

package safe

import (
	"context"
	"errors"
	"github.com/spazzle-io/safekit/pkg/version"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	_ "github.com/spazzle-io/safekit/internal/versions/v1_3_0"
	_ "github.com/spazzle-io/safekit/internal/versions/v1_4_1"
	_ "github.com/spazzle-io/safekit/internal/versions/v1_5_0"
	"github.com/spazzle-io/safekit/pkg/chain"
	"github.com/spazzle-io/safekit/pkg/signer"
)

// integrationClientFromEnv builds a Client from environment variables.
// Required:
//
//	SAFEKIT_TEST_RPC_URL   — RPC endpoint for the test network
//	SAFEKIT_TEST_ADMIN_KEY — hex-encoded private key for the admin wallet
//	SAFEKIT_TEST_CHAIN_ID  — chain ID as a decimal integer
//	SAFEKIT_TEST_VERSION   — Safe version e.g. "1.4.1"
func integrationClientFromEnv(t *testing.T) *Client {
	t.Helper()

	rpcURL := requireEnv(t, "SAFEKIT_TEST_RPC_URL")
	adminKey := requireEnv(t, "SAFEKIT_TEST_ADMIN_KEY")
	chainIDStr := requireEnv(t, "SAFEKIT_TEST_CHAIN_ID")
	versionStr := requireEnv(t, "SAFEKIT_TEST_VERSION")

	chainIDInt, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid SAFEKIT_TEST_CHAIN_ID %q: %v", chainIDStr, err)
	}

	c, err := chain.Lookup(big.NewInt(chainIDInt))
	if err != nil {
		t.Fatalf("unsupported chain ID %d: %v", chainIDInt, err)
	}

	s, err := signer.NewSignerFromHex(adminKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	client, err := New(Options{
		Chain:         c,
		RPC:           rpcURL,
		Signer:        s,
		Version:       version.Version(versionStr),
		DeployTimeout: 3 * time.Minute,
	})
	if err != nil {
		if errors.Is(err, ErrVersionNotOnChain) {
			t.Skipf("version %s not deployed on chain %d yet. Skipping...", versionStr, chainIDInt)
		}
		t.Fatalf("failed to create client: %v", err)
	}

	t.Cleanup(func() { client.Close() })
	return client
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()

	v := os.Getenv(key)
	if v == "" {
		t.Skipf("%s not set — skipping integration test", key)
	}

	return v
}

func integrationOwners(t *testing.T) []common.Address {
	t.Helper()

	return []common.Address{
		signer.NewMockSigner(1).Address(),
		signer.NewMockSigner(2).Address(),
		signer.NewMockSigner(3).Address(),
	}
}

func TestIntegration_PredictAndDeploy(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	salt, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error generating salt: %v", err)
	}

	predicted, err := client.PredictAddress(owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error predicting: %v", err)
	}
	t.Logf("predicted: %s", predicted.Hex())

	deployed, err := client.IsDeployed(ctx, predicted)
	if err != nil {
		t.Fatalf("unexpected error checking deployment: %v", err)
	}
	if deployed {
		t.Fatal("address should not be deployed before Deploy is called")
	}

	result, err := client.Deploy(ctx, owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error deploying: %v", err)
	}
	t.Logf("deployed: %s (tx: %s, block: %d, gas: %d)",
		result.SafeAddress.Hex(),
		result.TxHash.Hex(),
		result.BlockNumber,
		result.GasUsed,
	)

	if result.SafeAddress != predicted {
		t.Errorf("predicted %s but deployed to %s",
			predicted.Hex(), result.SafeAddress.Hex())
	}

	deployed, err = client.IsDeployed(ctx, result.SafeAddress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deployed {
		t.Error("address should be deployed after Deploy is called")
	}
}

func TestIntegration_AlreadyDeployed(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	// fixed salt makes this idempotent across runs
	salt := []byte("safekit-integration-already-deployed")

	// first deploy may already exist from a previous run — ignore error
	_, _ = client.Deploy(ctx, owners, 2, salt)

	// second deploy must always return ErrAddressAlreadyDeployed
	_, err := client.Deploy(ctx, owners, 2, salt)
	if !errors.Is(err, ErrAddressAlreadyDeployed) {
		t.Errorf("expected ErrAddressAlreadyDeployed, got %v", err)
	}
}

func TestIntegration_SubmitAndWait(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	salt, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	predicted, err := client.PredictAddress(owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	txHash, err := client.SubmitDeployment(ctx, owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error submitting: %v", err)
	}
	t.Logf("submitted tx: %s", txHash.Hex())

	result, err := client.WaitForDeployment(ctx, owners, 2, salt, txHash)
	if err != nil {
		t.Fatalf("unexpected error waiting: %v", err)
	}

	if result.SafeAddress != predicted {
		t.Errorf("predicted %s but deployed to %s",
			predicted.Hex(), result.SafeAddress.Hex())
	}
}
