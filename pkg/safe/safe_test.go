package safe

import (
	"errors"
	"testing"

	"github.com/spazzle-io/safekit/internal/txmanager"

	"github.com/spazzle-io/safekit/pkg/version"

	"github.com/ethereum/go-ethereum/common"

	"github.com/spazzle-io/safekit/internal/versions"
	_ "github.com/spazzle-io/safekit/internal/versions/v1_4_1"
	"github.com/spazzle-io/safekit/pkg/chain"
	"github.com/spazzle-io/safekit/pkg/signer"
)

var testOwners = []common.Address{
	common.HexToAddress("0x1111111111111111111111111111111111111111"),
	common.HexToAddress("0x2222222222222222222222222222222222222222"),
	common.HexToAddress("0x3333333333333333333333333333333333333333"),
}

func newTestClient(t *testing.T) *Client {
	t.Helper()

	deployment, err := versions.Get(version.V141)
	if err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}

	return &Client{
		chain:      chain.Ethereum,
		deployer:   txmanager.New(nil, signer.NewMockSigner(0)),
		deployment: deployment,
		opts:       &Options{},
	}
}

func TestNew_MissingChain(t *testing.T) {
	_, err := New(Options{
		RPC:     "http://localhost:8545",
		Signer:  signer.NewMockSigner(0),
		Version: version.V141,
	})
	if err == nil {
		t.Fatal("expected error for missing Chain, got nil")
	}
}

func TestNew_MissingRPC(t *testing.T) {
	_, err := New(Options{
		Chain:   chain.Ethereum,
		Signer:  signer.NewMockSigner(0),
		Version: version.V141,
	})
	if err == nil {
		t.Fatal("expected error for missing RPC, got nil")
	}
}

func TestNew_MissingSigner(t *testing.T) {
	_, err := New(Options{
		Chain:   chain.Ethereum,
		RPC:     "http://localhost:8545",
		Version: version.V141,
	})
	if err == nil {
		t.Fatal("expected error for missing Signer, got nil")
	}
}

func TestNew_MissingVersion(t *testing.T) {
	_, err := New(Options{
		Chain:  chain.Ethereum,
		RPC:    "http://localhost:8545",
		Signer: signer.NewMockSigner(0),
	})
	if err == nil {
		t.Fatal("expected error for missing Version, got nil")
	}
}

func TestRandomSalt_Length(t *testing.T) {
	salt, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(salt) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(salt))
	}
}

func TestRandomSalt_Unique(t *testing.T) {
	a, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(a) == string(b) {
		t.Error("RandomSalt produced identical values on consecutive calls")
	}
}

func TestValidateSafeConfig_EmptyOwners(t *testing.T) {
	err := validateSafeConfig([]common.Address{}, 1)
	if !errors.Is(err, ErrEmptyOwners) {
		t.Errorf("expected ErrEmptyOwners, got %v", err)
	}
}

func TestValidateSafeConfig_ZeroAddressOwner(t *testing.T) {
	err := validateSafeConfig([]common.Address{{}}, 1)
	if !errors.Is(err, ErrZeroAddressOwner) {
		t.Errorf("expected ErrZeroAddressOwner, got %v", err)
	}

	err = validateSafeConfig([]common.Address{zeroAddress}, 1)
	if !errors.Is(err, ErrZeroAddressOwner) {
		t.Errorf("expected ErrZeroAddressOwner, got %v", err)
	}

	err = validateSafeConfig([]common.Address{{}, zeroAddress}, 1)
	if !errors.Is(err, ErrZeroAddressOwner) {
		t.Errorf("expected ErrZeroAddressOwner, got %v", err)
	}
}

func TestValidateSafeConfig_DuplicateOwner(t *testing.T) {
	owners := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}
	err := validateSafeConfig(owners, 1)
	if !errors.Is(err, ErrDuplicateOwner) {
		t.Errorf("expected ErrDuplicateOwner, got %v", err)
	}
}

func TestValidateSafeConfig_ThresholdZero(t *testing.T) {
	err := validateSafeConfig(testOwners, 0)
	if !errors.Is(err, ErrInvalidThreshold) {
		t.Errorf("expected ErrInvalidThreshold, got %v", err)
	}
}

func TestValidateSafeConfig_ThresholdExceedsOwners(t *testing.T) {
	err := validateSafeConfig(testOwners, 10)
	if !errors.Is(err, ErrInvalidThreshold) {
		t.Errorf("expected ErrInvalidThreshold, got %v", err)
	}
}

func TestValidateSafeConfig_Valid(t *testing.T) {
	err := validateSafeConfig(testOwners, 2)
	if err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}
}

func TestPredictAddress_Deterministic(t *testing.T) {
	c := newTestClient(t)

	a, err := c.PredictAddress(testOwners, 2, []byte("salt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := c.PredictAddress(testOwners, 2, []byte("salt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a != b {
		t.Errorf("same inputs produced different addresses: %s vs %s",
			a.Hex(), b.Hex())
	}
}

func TestPredictAddress_DifferentSalt(t *testing.T) {
	c := newTestClient(t)

	a, err := c.PredictAddress(testOwners, 2, []byte("salt-a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := c.PredictAddress(testOwners, 2, []byte("salt-b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a == b {
		t.Error("different salts produced the same address")
	}
}

func TestPredictAddress_NilSalt(t *testing.T) {
	c := newTestClient(t)

	_, err := c.PredictAddress(testOwners, 2, nil)
	if err != nil {
		t.Fatalf("nil salt should be valid, got error: %v", err)
	}
}

func TestPredictAddress_NilAndEmptySaltMatch(t *testing.T) {
	c := newTestClient(t)

	a, err := c.PredictAddress(testOwners, 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, err := c.PredictAddress(testOwners, 2, []byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a != b {
		t.Error("nil and empty salt should produce the same address")
	}
}

func TestPredictAddress_ValidationErrors(t *testing.T) {
	c := newTestClient(t)

	tests := []struct {
		name      string
		owners    []common.Address
		threshold uint8
		wantErr   error
	}{
		{
			name:      "empty owners",
			owners:    []common.Address{},
			threshold: 1,
			wantErr:   ErrEmptyOwners,
		},
		{
			name:      "zero address owner",
			owners:    []common.Address{{}},
			threshold: 1,
			wantErr:   ErrZeroAddressOwner,
		},
		{
			name: "duplicate owner",
			owners: []common.Address{
				common.HexToAddress("0x1111111111111111111111111111111111111111"),
				common.HexToAddress("0x1111111111111111111111111111111111111111"),
			},
			threshold: 1,
			wantErr:   ErrDuplicateOwner,
		},
		{
			name:      "threshold zero",
			owners:    testOwners,
			threshold: 0,
			wantErr:   ErrInvalidThreshold,
		},
		{
			name:      "threshold exceeds owners",
			owners:    testOwners,
			threshold: 10,
			wantErr:   ErrInvalidThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.PredictAddress(tt.owners, tt.threshold, []byte("salt"))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Close_Idempotent(t *testing.T) {
	c := newTestClient(t)

	// calling Close on a client with nil eth should not panic
	c.Close()
	c.Close()
}

func TestClient_ImplementsDeployer(t *testing.T) {
	var _ Deployer = (*Client)(nil)
}
