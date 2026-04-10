package signer

import (
	"context"
	"math/big"
	"testing"
)

func TestNewMockSigner_Deterministic(t *testing.T) {
	// same seed must always produce the same address
	a := NewMockSigner(0)
	b := NewMockSigner(0)

	if a.Address() != b.Address() {
		t.Errorf("same seed produced different addresses: %s vs %s",
			a.Address().Hex(), b.Address().Hex())
	}
}

func TestNewMockSigner_DifferentSeeds(t *testing.T) {
	// different seeds must produce different addresses
	a := NewMockSigner(0)
	b := NewMockSigner(1)

	if a.Address() == b.Address() {
		t.Errorf("different seeds produced the same address: %s", a.Address().Hex())
	}
}

func TestMockSigner_SignTx(t *testing.T) {
	s := NewMockSigner(0)
	defer s.Close()

	signed, err := s.SignTx(context.Background(), newTestTx(), big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error signing tx: %v", err)
	}

	if signed == nil {
		t.Fatal("expected signed transaction, got nil")
	}
}

func TestMockSigner_Close_Idempotent(t *testing.T) {
	s := NewMockSigner(0)

	// calling Close multiple times on a mock should never panic
	s.Close()
	s.Close()
}
