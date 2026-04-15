package chain

import (
	"math/big"
	"testing"
)

func TestLookup_BuiltInChains(t *testing.T) {
	tests := []struct {
		name    string
		chainID *big.Int
		want    string
		isL2    bool
	}{
		{"ethereum", big.NewInt(1), "Ethereum", false},
		{"sepolia", big.NewInt(11155111), "Sepolia", false},
		{"polygon", big.NewInt(137), "Polygon", true},
		{"polygon zkevm", big.NewInt(1101), "Polygon zkEVM", true},
		{"polygon amoy", big.NewInt(80002), "Polygon Amoy", true},
		{"arbitrum one", big.NewInt(42161), "Arbitrum One", true},
		{"arbitrum nova", big.NewInt(42170), "Arbitrum Nova", true},
		{"arbitrum sepolia", big.NewInt(421614), "Arbitrum Sepolia", true},
		{"base", big.NewInt(8453), "Base", true},
		{"base sepolia", big.NewInt(84532), "Base Sepolia", true},
		{"optimism", big.NewInt(10), "Optimism", true},
		{"optimism sepolia", big.NewInt(11155420), "Optimism Sepolia", true},
		{"bnb smart chain", big.NewInt(56), "BNB Smart Chain", false},
		{"bnb smart chain testnet", big.NewInt(97), "BNB Smart Chain Testnet", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Lookup(tt.chainID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if c.Name != tt.want {
				t.Errorf("got name %q, want %q", c.Name, tt.want)
			}

			if c.IsL2 != tt.isL2 {
				t.Errorf("got IsL2 %v, want %v", c.IsL2, tt.isL2)
			}
		})
	}
}

func TestLookup_UnknownChain(t *testing.T) {
	_, err := Lookup(big.NewInt(99999999))
	if err == nil {
		t.Fatal("expected error for unknown chain ID, got nil")
	}
}

func TestLookup_NilID(t *testing.T) {
	_, err := Lookup(nil)
	if err == nil {
		t.Fatal("expected error for nil chain ID, got nil")
	}
}

func TestRegister_NewChain(t *testing.T) {
	custom := &Chain{
		ID:   big.NewInt(99999),
		Name: "My Test Chain",
		IsL2: false,
	}

	if err := Register(custom); err != nil {
		t.Fatalf("unexpected error registering chain: %v", err)
	}

	got, err := Lookup(big.NewInt(99999))
	if err != nil {
		t.Fatalf("unexpected error looking up registered chain: %v", err)
	}

	if got.Name != custom.Name {
		t.Errorf("got name %q, want %q", got.Name, custom.Name)
	}
}

func TestRegister_OverridesExisting(t *testing.T) {
	override := &Chain{
		ID:   big.NewInt(1),
		Name: "My Custom Ethereum",
		IsL2: false,
	}

	if err := Register(override); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := Lookup(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Name != "My Custom Ethereum" {
		t.Errorf("got name %q, want %q", got.Name, "My Custom Ethereum")
	}

	// restore so other tests aren't affected
	if err := Register(Ethereum); err != nil {
		t.Fatalf("unexpected error restoring chain: %v", err)
	}
}

func TestRegister_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		chain *Chain
	}{
		{"nil chain", nil},
		{"nil ID", &Chain{Name: "No ID"}},
		{"empty name", &Chain{ID: big.NewInt(1), Name: " \n\t"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Register(tt.chain); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
