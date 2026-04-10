package signer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// EnvSigner loads a private key from an environment variable at startup,
// parses it immediately, and never stores the raw string. It's the right
// choice when your key lives in an environment variable — which for most
// backend services means it came from a secrets manager at deploy time.
//
// Usage:
//
//	s, err := signer.NewEnvSigner("SAFE_ADMIN_KEY")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
type EnvSigner struct {
	key     *ecdsa.PrivateKey
	address common.Address
}

// NewEnvSigner reads a hex-encoded private key from the named environment
// variable and returns a ready-to-use signer.
func NewEnvSigner(envVar string) (Signer, error) {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return nil, fmt.Errorf("environment variable %q is not set", envVar)
	}

	raw = strings.TrimPrefix(raw, "0x")

	key, err := crypto.HexToECDSA(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from %q: %w", envVar, err)
	}

	return &EnvSigner{
		key:     key,
		address: crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}

// Address returns the public address derived from this signer's private key.
func (s *EnvSigner) Address() common.Address {
	return s.address
}

// SignTx signs the transaction for the given chain ID using EIP-1559 signing
// Returns the signed transaction ready to broadcast.
func (s *EnvSigner) SignTx(_ context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	if s.key == nil {
		return nil, fmt.Errorf("EnvSigner has been closed and can no longer sign")
	}

	signed, err := types.SignTx(tx, types.NewLondonSigner(chainID), s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return signed, nil
}

// Close zeroes the private key material from memory. Call this when you're
// done with the signer — typically via defer right after construction.
// After Close, SignTx will return an error.
func (s *EnvSigner) Close() {
	if s.key != nil && s.key.D != nil {
		clear(s.key.D.Bits())
		s.key = nil
	}
}
