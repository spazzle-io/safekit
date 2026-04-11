package signer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// MockSigner is a deterministic signer for use in tests. It derives its key
// from a fixed seed so the address is always the same for the same seed,
// which makes test assertions stable across runs.
//
// The signatures it produces are cryptographically real but the key holds no
// funds on any live network. Never use a MockSigner in production.
//
// Usage:
//
//	s := signer.NewMockSigner(0) // always the same address for seed 0
//	defer s.Close()
type MockSigner struct {
	key     *ecdsa.PrivateKey
	address common.Address
}

// NewMockSigner creates a MockSigner from an integer seed. The same seed
// always produces the same private key and address, making it safe to
// hardcode expected addresses in test assertions.
func NewMockSigner(seed uint64) Signer {
	seedBytes := make([]byte, 8)
	for i := range 8 {
		seedBytes[7-i] = byte((seed >> (8 * i)) & 0xff)
	}

	keyBytes := crypto.Keccak256(seedBytes)
	key, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		// keccak256 output is always a valid ECDSA scalar — if this
		// ever panics something is deeply wrong with the environment
		panic(fmt.Sprintf("signer: MockSigner seed %d produced invalid key: %v", seed, err))
	}

	return &MockSigner{
		key:     key,
		address: crypto.PubkeyToAddress(key.PublicKey),
	}
}

// Address returns the public address derived from this signer's seed.
func (s *MockSigner) Address() common.Address {
	return s.address
}

// SignTx signs the transaction using the deterministic key for this seed.
func (s *MockSigner) SignTx(_ context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	signed, err := types.SignTx(tx, types.NewLondonSigner(chainID), s.key)
	if err != nil {
		return nil, fmt.Errorf("mock failed to sign transaction: %w", err)
	}

	return signed, nil
}

// Close is a no-op for MockSigner.
func (s *MockSigner) Close() {}
