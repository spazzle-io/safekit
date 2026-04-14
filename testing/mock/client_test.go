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

func TestMockClient_ImplementsSafeClient(t *testing.T) {
	var _ safe.Client = NewClient()
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

	addrs := c.DeployedAddresses()
	if len(addrs) != 1 {
		t.Errorf("expected 1 address; got %d", len(addrs))
	}

	c.Reset()

	addrs = c.DeployedAddresses()
	if len(addrs) != 0 {
		t.Errorf("expected no deployed addresses after Reset, got %d", len(addrs))
	}
}

func TestMockClient_SubmitAndWait(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	predicted, err := c.PredictAddress(testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error predicting: %v", err)
	}

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

	if result.SafeAddress != predicted {
		t.Errorf("predicted %s but got %s", predicted.Hex(), result.SafeAddress.Hex())
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

func TestForceError_Deploy(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceError(safe.ErrTransactionReverted)

	_, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrTransactionReverted) {
		t.Fatalf("want ErrTransactionReverted, got %v", err)
	}
}

func TestForceError_SubmitDeployment(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceError(safe.ErrTransactionReverted)

	_, err := c.SubmitDeployment(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrTransactionReverted) {
		t.Fatalf("want ErrTransactionReverted, got %v", err)
	}
}

func TestForceError_WaitForDeployment(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	txHash, err := c.SubmitDeployment(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("unexpected error submitting: %v", err)
	}

	c.ForceError(safe.ErrDeployTimeout)

	_, err = c.WaitForDeployment(ctx, testOwners, testThreshold, testSalt, txHash)
	if !errors.Is(err, safe.ErrDeployTimeout) {
		t.Fatalf("want ErrDeployTimeout, got %v", err)
	}
}

func TestForceError_ConsumedAfterOneUse(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceError(safe.ErrTransactionReverted)

	_, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrTransactionReverted) {
		t.Fatalf("want ErrTransactionReverted on first call, got %v", err)
	}

	result, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("want success on second call, got %v", err)
	}
	if result == nil {
		t.Fatal("want non-nil result on second call")
	}
}

func TestForceErrors_Sequential(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceErrors(safe.ErrDeployTimeout, nil, safe.ErrTransactionReverted, nil)

	_, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrDeployTimeout) {
		t.Fatalf("first call: want ErrDeployTimeout, got %v", err)
	}

	result, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("second call: want success, got %v", err)
	}
	if result == nil {
		t.Fatal("second call: want non-nil result")
	}

	_, err = c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrTransactionReverted) {
		t.Fatalf("third call: want ErrTransactionReverted, got %v", err)
	}

	result, err = c.Deploy(ctx, testOwners, testThreshold, []byte("different salt"))
	if err != nil {
		t.Fatalf("fourth call: want success, got %v", err)
	}
	if result == nil {
		t.Fatal("fourth call: want non-nil result")
	}
}

func TestForceErrors_QueueAppend(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceErrors(safe.ErrDeployTimeout)
	c.ForceErrors(safe.ErrTransactionReverted)

	_, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrDeployTimeout) {
		t.Fatalf("first call: want ErrDeployTimeout, got %v", err)
	}

	_, err = c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrTransactionReverted) {
		t.Fatalf("second call: want ErrTransactionReverted, got %v", err)
	}
}

func TestForceError_ResetClearsForcedErrors(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceErrors(safe.ErrTransactionReverted, safe.ErrDeployTimeout)
	c.Reset()

	result, err := c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("want success after Reset, got %v", err)
	}
	if result == nil {
		t.Fatal("want non-nil result after Reset")
	}
}

func TestForceError_DoesNotAffectPredictOrIsDeployed(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	c.ForceError(safe.ErrTransactionReverted)

	_, err := c.PredictAddress(testOwners, testThreshold, testSalt)
	if err != nil {
		t.Fatalf("PredictAddress should not be affected by ForceError, got %v", err)
	}

	_, err = c.IsDeployed(ctx, common.Address{})
	if err != nil {
		t.Fatalf("IsDeployed should not be affected by ForceError, got %v", err)
	}

	_, err = c.Deploy(ctx, testOwners, testThreshold, testSalt)
	if !errors.Is(err, safe.ErrTransactionReverted) {
		t.Fatalf("want ErrTransactionReverted still pending, got %v", err)
	}
}
