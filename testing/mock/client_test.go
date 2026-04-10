package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/pkg/safe"
)

var (
	testOwners = []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}
	testThreshold = uint8(1)
	testSalt      = []byte("test-salt")
)

func TestMockClient_ImplementsDeployer(t *testing.T) {
	var _ safe.Deployer = NewClient()
}

func TestMockClient_PredictAddress_Deterministic(t *testing.T) {
	c := NewClient()

	a, err := c.PredictAddress(testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := c.PredictAddress(testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a != b {
		t.Errorf("same inputs produced different addresses: %s vs %s", a.Hex(), b.Hex())
	}
}

func TestMockClient_Deploy_PredictMatchesDeploy(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	predicted, err := c.PredictAddress(testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error predicting: %v", err)
	}

	result, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error deploying: %v", err)
	}

	if result.SafeAddress != predicted {
		t.Errorf("predicted %s but deployed to %s", predicted.Hex(), result.SafeAddress.Hex())
	}
}

func TestMockClient_Deploy_AlreadyDeployed(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	_, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error on first deploy: %v", err)
	}

	_, err = c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrAddressAlreadyDeployed) {
		t.Errorf("expected ErrAddressAlreadyDeployed, got %v", err)
	}
}

func TestMockClient_IsDeployed(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	addr, err := c.PredictAddress(testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deployed, err := c.IsDeployed(ctx, addr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deployed {
		t.Error("address should not be deployed before Deploy is called")
	}

	_, err = c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error deploying: %v", err)
	}

	deployed, err = c.IsDeployed(ctx, addr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deployed {
		t.Error("address should be deployed after Deploy is called")
	}
}

func TestMockClient_Reset(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	_, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c.Reset()

	addrs := c.DeployedAddresses()
	if len(addrs) != 0 {
		t.Errorf("expected no deployed addresses after Reset, got %d", len(addrs))
	}
}

func TestMockClient_SubmitAndWait(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	txHash, err := c.SubmitDeployment(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error submitting: %v", err)
	}

	result, err := c.WaitForDeployment(ctx, testOwners, testThreshold, testSalt, txHash)
	if err != nil {
		t.Fatalf("unexpected error waiting: %v", err)
	}

	if result.TxHash != txHash {
		t.Errorf("tx hash mismatch: got %s, want %s", result.TxHash.Hex(), txHash.Hex())
	}
}

func TestMockClient_DeployedAddresses(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	addrs := c.DeployedAddresses()
	if len(addrs) != 0 {
		t.Errorf("expected no deployed addresses, got %d", len(addrs))
	}

	_, err := c.Deploy(ctx, testOwners, testThreshold, []byte("salt-one"))
	if err != nil {
		t.Fatalf("unexpected error on first deploy: %v", err)
	}

	_, err = c.Deploy(ctx, testOwners, testThreshold, []byte("salt-two"))
	if err != nil {
		t.Fatalf("unexpected error on second deploy: %v", err)
	}

	addrs = c.DeployedAddresses()
	if len(addrs) != 2 {
		t.Errorf("expected 2 deployed addresses, got %d", len(addrs))
	}

	addrOne, err := c.PredictAddress(testOwners, testThreshold, []byte("salt-one"))
	if err != nil {
		t.Fatalf("unexpected error predicting: %v", err)
	}

	addrTwo, err := c.PredictAddress(testOwners, testThreshold, []byte("salt-two"))
	if err != nil {
		t.Fatalf("unexpected error predicting: %v", err)
	}

	found := make(map[common.Address]bool)
	for _, a := range addrs {
		found[a] = true
	}

	if !found[addrOne] {
		t.Errorf("expected %s in DeployedAddresses", addrOne.Hex())
	}
	if !found[addrTwo] {
		t.Errorf("expected %s in DeployedAddresses", addrTwo.Hex())
	}
}
