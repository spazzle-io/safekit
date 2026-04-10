package v1_5_0

import (
	"math/big"
	"testing"

	"github.com/spazzle-io/safekit/internal/versions"
)

func TestV150_ProxyFactory_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical proxy factory address for v1.5.0 on Ethereum
	want := "0x14F2982D601c9458F93bd70B218933A6f8165e7b"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV150_Singleton_L1_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := d.Singleton(big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// canonical Safe singleton address for v1.5.0 on Ethereum
	want := "0xFf51A5898e281Db6DfC7855790607438dF2ca44b"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV150_Singleton_L2_Ethereum(t *testing.T) {
	d, err := versions.Get(versions.Version150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ethereum is listed in the v1.5.0 L2 JSON too
	got, err := d.Singleton(big.NewInt(1), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0xEdd160fEBBD92E350D4D398fb636302fccd67C7e"
	if got.Address.Hex() != want {
		t.Errorf("got %s, want %s", got.Address.Hex(), want)
	}
}

func TestV150_UnsupportedChain_ReturnsError(t *testing.T) {
	d, err := versions.Get(versions.Version150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// v1.5.0 has limited chain coverage. Chains only in v1.4.1 should
	// return a clear error rather than silently failing or returning a
	// wrong address.
	_, err = d.ProxyFactory(big.NewInt(56)) // BNB Smart Chain
	if err == nil {
		t.Fatal("expected error for chain not yet supported in v1.5.0, got nil")
	}
}

func TestV150_RegisteredInRegistry(t *testing.T) {
	got, err := versions.Get(versions.Version150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Version() != versions.Version150 {
		t.Errorf("got version %q, want %q", got.Version(), versions.Version150)
	}
}
