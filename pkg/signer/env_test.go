package signer

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// a known test private key and its corresponding address.
// DO NOT use these keys for anything real.
const (
	testPrivateKey = "3289cccbc9c65fb424a36e726c48ecb7a0f8d8e61dfdff1dc869107978e53ec9"
	testAddress    = "a53e8c5aba180dc39e2b2319521fbb517a13ae19"
)

func TestNewEnvSigner_Success(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY", testPrivateKey)

	s, err := NewEnvSigner("TEST_SAFE_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	want := common.HexToAddress(testAddress)
	if s.Address() != want {
		t.Errorf("got address %s, want %s", s.Address().Hex(), want.Hex())
	}
}

func TestNewEnvSigner_WithOxPrefix(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY_0X", "0x"+testPrivateKey)

	s, err := NewEnvSigner("TEST_SAFE_KEY_0X")
	if err != nil {
		t.Fatalf("unexpected error with 0x prefix: %v", err)
	}
	defer s.Close()

	want := common.HexToAddress(testAddress)
	if s.Address() != want {
		t.Errorf("got address %s, want %s", s.Address().Hex(), want.Hex())
	}
}

func TestNewEnvSigner_MissingEnvVar(t *testing.T) {
	_, err := NewEnvSigner("SAFEKIT_THIS_VAR_DOES_NOT_EXIST")
	if err == nil {
		t.Fatal("expected error for missing env var, got nil")
	}
}

func TestNewEnvSigner_InvalidKey(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY_BAD", "this-is-not-a-valid-private-key")

	_, err := NewEnvSigner("TEST_SAFE_KEY_BAD")
	if err == nil {
		t.Fatal("expected error for invalid private key, got nil")
	}
}

func TestEnvSigner_SignTx(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY", testPrivateKey)

	s, err := NewEnvSigner("TEST_SAFE_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	tx := newTestTx()
	signed, err := s.SignTx(context.Background(), tx, big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error signing tx: %v", err)
	}

	if signed == nil {
		t.Fatal("expected signed transaction, got nil")
	}
}

func TestEnvSigner_SignTx_AfterClose(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY", testPrivateKey)

	s, err := NewEnvSigner("TEST_SAFE_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s.Close()

	_, err = s.SignTx(context.Background(), newTestTx(), big.NewInt(1))
	if err == nil {
		t.Fatal("expected error signing after Close, got nil")
	}
}

func TestEnvSigner_Close_Idempotent(t *testing.T) {
	t.Setenv("TEST_SAFE_KEY", testPrivateKey)

	s, err := NewEnvSigner("TEST_SAFE_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// calling Close twice should not panic
	s.Close()
	s.Close()
}
