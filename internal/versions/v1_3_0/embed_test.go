package v1_3_0

import (
	"math/big"
	"testing"

	"github.com/spazzle-io/safekit/internal/versions"
)

func TestV130_ProxyFactory_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical proxy factory address for v1.3.0 on Ethereum
	want := "0xa6B71E26C5e0845f74c812102Ca7114b6a896AB2"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV130_Singleton_L1_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := d.Singleton(big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical Safe singleton address for v1.3.0 on Ethereum
	want := "0xd9Db270c1B5E3Bd161E8c8503c55cEABeE709552"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV130_Singleton_L2_Polygon(t *testing.T) {
	d, err := versions.Get(versions.Version130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Polygon POS — chain 137, L2 variant
	got, err := d.Singleton(big.NewInt(137), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical SafeL2 singleton address for v1.3.0
	want := "0x3E5c63644E683549055b9Be8653de26E0B4CD36E"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV130_Singleton_EIP155Preferred_Optimism(t *testing.T) {
	d, err := versions.Get(versions.Version130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Optimism (chain 10) has ["eip155", "canonical"] in v1.3.0
	// eip155 is the preferred type for this chain
	got, err := d.Singleton(big.NewInt(10), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// eip155 SafeL2 singleton address
	want := "0xfb1bffC9d739B8D520DaF37dF666da4C687191EA"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV130_RegisteredInRegistry(t *testing.T) {
	got, err := versions.Get(versions.Version130)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Version() != versions.Version130 {
		t.Errorf("got version %q, want %q", got.Version(), versions.Version130)
	}
}
