package v1_4_1

import (
	"math/big"
	"testing"

	"github.com/spazzle-io/safekit/internal/versions"
)

func TestV141_ProxyFactory_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version141)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical proxy factory address for v1.4.1 on Ethereum
	want := "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV141_Singleton_L1_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version141)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := d.Singleton(big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical Safe singleton address for v1.4.1 on Ethereum
	want := "0x41675C099F32341bf84BFc5382aF534df5C7461a"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV141_Singleton_L2_Polygon(t *testing.T) {
	d, err := versions.Get(versions.Version141)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Polygon POS — chain 137, L2 variant
	got, err := d.Singleton(big.NewInt(137), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical SafeL2 singleton address for v1.4.1
	want := "0x29fcB43b46531BcA003ddC8FCB67FFE91900C762"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV141_NoEIP155Variant(t *testing.T) {
	d, err := versions.Get(versions.Version141)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// v1.4.1 has no eip155 variant. Canonical is the only EVM deployment type.
	// Optimism (chain 10) should resolve to canonical directly
	got, err := d.Singleton(big.NewInt(10), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// same canonical SafeL2 address as every other EVM chain in v1.4.1
	want := "0x29fcB43b46531BcA003ddC8FCB67FFE91900C762"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV141_RegisteredInRegistry(t *testing.T) {
	got, err := versions.Get(versions.Version141)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Version() != versions.Version141 {
		t.Errorf("got version %q, want %q", got.Version(), versions.Version141)
	}
}
