package predict

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/spazzle-io/safekit/internal/versions"

	"github.com/ethereum/go-ethereum/common"
)

var testOwners = []common.Address{
	common.HexToAddress("0x1111111111111111111111111111111111111111"),
	common.HexToAddress("0x2222222222222222222222222222222222222222"),
	common.HexToAddress("0x3333333333333333333333333333333333333333"),
}

func TestAddress_DeterministicOutput(t *testing.T) {
	deployment := newTestDeployment()

	first, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("test-salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("test-salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first != second {
		t.Errorf("same inputs produced different addresses: %s vs %s",
			first.Hex(), second.Hex())
	}
}

func TestAddress_DifferentSaltProducesDifferentAddress(t *testing.T) {
	deployment := newTestDeployment()

	a, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("salt-a"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("salt-b"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a == b {
		t.Error("different salts produced the same address")
	}
}

func TestAddress_DifferentThresholdProducesDifferentAddress(t *testing.T) {
	deployment := newTestDeployment()

	a, err := Address(Input{
		Owners:    testOwners,
		Threshold: 1,
		Salt:      []byte("salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a == b {
		t.Error("different thresholds produced the same address")
	}
}

func TestAddress_DifferentOwnersProducesDifferentAddress(t *testing.T) {
	deployment := newTestDeployment()

	ownersA := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}
	ownersB := []common.Address{
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	a, err := Address(Input{
		Owners:    ownersA,
		Threshold: 1,
		Salt:      []byte("salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := Address(Input{
		Owners:    ownersB,
		Threshold: 1,
		Salt:      []byte("salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a == b {
		t.Error("different owners produced the same address")
	}
}

func TestAddress_L1vsL2ProducesDifferentAddress(t *testing.T) {
	deployment := newTestDeployment()

	l1, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("salt"),
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l2, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("salt"),
	}, deployment, big.NewInt(1), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if l1 == l2 {
		t.Error("L1 and L2 produced the same address")
	}
}

func TestAddress_NilChainID(t *testing.T) {
	deployment := newTestDeployment()

	_, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte("salt"),
	}, deployment, nil, false)
	if err == nil {
		t.Fatal("expected error for nil chainID, got nil")
	}
}

func TestAddress_EmptySaltIsValid(t *testing.T) {
	deployment := newTestDeployment()

	_, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte{},
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("empty salt should be valid, got error: %v", err)
	}
}

func TestAddress_NilSaltIsValid(t *testing.T) {
	deployment := newTestDeployment()

	_, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      nil,
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("nil salt should be valid, got error: %v", err)
	}
}

func TestAddress_NilAndEmptySaltProduceSameAddress(t *testing.T) {
	deployment := newTestDeployment()

	nilSalt, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      nil,
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emptySalt, err := Address(Input{
		Owners:    testOwners,
		Threshold: 2,
		Salt:      []byte{},
	}, deployment, big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if nilSalt != emptySalt {
		t.Error("nil and empty salt should produce the same address")
	}
}

type testDeployment struct{}

func newTestDeployment() versions.Deployment {
	return &testDeployment{}
}

func (d *testDeployment) Version() versions.Version {
	return "test.1.0"
}

func (d *testDeployment) ProxyFactory(_ *big.Int) (*versions.ParsedDeployment, error) {
	return &versions.ParsedDeployment{
		Address: common.HexToAddress("0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"),
	}, nil
}

func (d *testDeployment) Singleton(_ *big.Int, isL2 bool) (*versions.ParsedDeployment, error) {
	if isL2 {
		return &versions.ParsedDeployment{
			Address: common.HexToAddress("0x29fcB43b46531BcA003ddC8FCB67FFE91900C762"),
		}, nil
	}

	return &versions.ParsedDeployment{
		Address: common.HexToAddress("0x41675C099F32341bf84BFc5382aF534df5C7461a"),
	}, nil
}

func (d *testDeployment) ProxyCreationCode() []byte {
	const codeHex = "608060405234801561001057600080fd5b506040516101e63803806101e68339818101604052602081101561003357600080fd5b8101908080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614156100ca576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260228152602001806101c46022913960400191505060405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505060ab806101196000396000f3fe608060405273ffffffffffffffffffffffffffffffffffffffff600054167fa619486e0000000000000000000000000000000000000000000000000000000060003514156050578060005260206000f35b3660008037600080366000845af43d6000803e60008114156070573d6000fd5b3d6000f3fea264697066735822122003d1488ee65e08fa41e58e888a9865554c535f2c77126a82cb4c0f917f31441364736f6c63430007060033496e76616c69642073696e676c65746f6e20616464726573732070726f7669646564"
	code, _ := hex.DecodeString(codeHex)

	return code
}
