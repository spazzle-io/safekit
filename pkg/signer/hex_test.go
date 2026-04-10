package signer

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewSignerFromHex_Success(t *testing.T) {
	s, err := NewSignerFromHex(testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	want := common.HexToAddress(testAddress)
	if s.Address() != want {
		t.Errorf("got address %s, want %s", s.Address().Hex(), want.Hex())
	}
}

func TestNewSignerFromHex_WithOxPrefix(t *testing.T) {
	s, err := NewSignerFromHex("0x" + testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error with 0x prefix: %v", err)
	}
	defer s.Close()

	want := common.HexToAddress(testAddress)
	if s.Address() != want {
		t.Errorf("got address %s, want %s", s.Address().Hex(), want.Hex())
	}
}

func TestNewSignerFromHex_SameAddressAsEnvSigner(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY", testPrivateKey)

	envS, err := NewEnvSigner("TEST_SAFE_KEY")
	if err != nil {
		t.Fatalf("unexpected error creating EnvSigner: %v", err)
	}
	defer envS.Close()

	hexS, err := NewSignerFromHex(testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error creating hex signer: %v", err)
	}
	defer hexS.Close()

	if envS.Address() != hexS.Address() {
		t.Errorf("EnvSigner address %s differs from hex signer address %s",
			envS.Address().Hex(), hexS.Address().Hex())
	}
}

func TestNewSignerFromHex_EmptyKey(t *testing.T) {
	_, err := NewSignerFromHex("")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}

func TestNewSignerFromHex_InvalidKey(t *testing.T) {
	_, err := NewSignerFromHex("this-is-not-a-valid-key")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestNewSignerFromHex_SignTx(t *testing.T) {
	s, err := NewSignerFromHex(testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	signed, err := s.SignTx(context.Background(), newTestTx(), big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error signing: %v", err)
	}
	if signed == nil {
		t.Fatal("expected signed transaction, got nil")
	}
}

func TestNewSignerFromHex_Close_Idempotent(t *testing.T) {
	s, err := NewSignerFromHex(testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s.Close()
	s.Close() // should not panic
}

func TestNewSignerFromHex_SignAfterClose(t *testing.T) {
	s, err := NewSignerFromHex(testPrivateKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s.Close()

	_, err = s.SignTx(context.Background(), newTestTx(), big.NewInt(1))
	if err == nil {
		t.Fatal("expected error signing after Close, got nil")
	}
}

func TestNewSignerFromHex_ImplementsSigner(t *testing.T) {
	var _ Signer = &EnvSigner{}
}
