package versions

import (
	"math/big"
	"testing"
)

const validL2JSON = `{
	"deployments": {
		"canonical": {
			"address": "0x3E5c63644E683549055b9Be8653de26E0B4CD36E",
			"codeHash": "0xabc123"
		}
	},
	"networkAddresses": {
		"1":   "canonical",
		"137": "canonical"
	},
	"abi": []
}`

var testProxyCreationCode = []byte{0x60, 0x80, 0x60, 0x40}

func newTestDeployment() Deployment {
	return NewBaseDeployment(
		"test.1.0",
		[]byte(validJSON),
		[]byte(validL2JSON),
		[]byte(validJSON),
		testProxyCreationCode,
	)
}

func TestBaseDeployment_Version(t *testing.T) {
	v := Version("test.1.0")
	d := NewBaseDeployment(v, []byte(validJSON), []byte(validL2JSON), []byte(validJSON), testProxyCreationCode)
	if d.Version() != v {
		t.Errorf("got version %q, want %q", d.Version(), v)
	}
}

func TestBaseDeployment_ProxyFactory(t *testing.T) {
	d := newTestDeployment()

	got, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestBaseDeployment_Singleton_L1(t *testing.T) {
	d := newTestDeployment()

	got, err := d.Singleton(big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestBaseDeployment_Singleton_L2(t *testing.T) {
	d := newTestDeployment()

	got, err := d.Singleton(big.NewInt(1), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0x3E5c63644E683549055b9Be8653de26E0B4CD36E"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestBaseDeployment_Singleton_L1vsL2_DifferentAddresses(t *testing.T) {
	d := newTestDeployment()

	l1, err := d.Singleton(big.NewInt(1), false)
	if err != nil {
		t.Fatalf("unexpected error getting L1 singleton: %v", err)
	}

	l2, err := d.Singleton(big.NewInt(1), true)
	if err != nil {
		t.Fatalf("unexpected error getting L2 singleton: %v", err)
	}

	if l1.Address == l2.Address {
		t.Errorf("L1 and L2 singletons should have different addresses, both got %s", l1.Address.Hex())
	}
}

func TestBaseDeployment_UnsupportedChain(t *testing.T) {
	d := newTestDeployment()

	_, err := d.ProxyFactory(big.NewInt(99999999))
	if err == nil {
		t.Fatal("expected error for unsupported chain, got nil")
	}
}

func TestBaseDeployment_Cache_ReturnsSamePointer(t *testing.T) {
	d := newTestDeployment()

	first, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// same pointer confirms the cache is working — JSON was parsed once
	if first != second {
		t.Error("expected cached result to be the same pointer, got different pointers")
	}
}

func TestBaseDeployment_Cache_IndependentPerChain(t *testing.T) {
	d := newTestDeployment()

	chain1, err := d.ProxyFactory(big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error for chain 1: %v", err)
	}

	chain137, err := d.ProxyFactory(big.NewInt(137))
	if err != nil {
		t.Fatalf("unexpected error for chain 137: %v", err)
	}

	if chain1 == chain137 {
		t.Error("expected different cache entries for different chains")
	}
}

func TestBaseDeployment_Cache_ConcurrentAccess(t *testing.T) {
	d := newTestDeployment()

	const goroutines = 50
	results := make(chan error, goroutines)

	for range goroutines {
		go func() {
			_, err := d.ProxyFactory(big.NewInt(1))
			results <- err
		}()
	}

	for range goroutines {
		if err := <-results; err != nil {
			t.Errorf("unexpected error under concurrent access: %v", err)
		}
	}
}

func TestBaseDeployment_NilChainID(t *testing.T) {
	d := newTestDeployment()

	_, err := d.ProxyFactory(nil)
	if err == nil {
		t.Fatal("expected error for nil chain ID, got nil")
	}
}
