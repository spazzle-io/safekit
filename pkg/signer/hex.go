package signer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// NewSignerFromHex creates a Signer from a hex-encoded private key string.
// Both 0x-prefixed and unprefixed strings are accepted.
//
// The key exists as a plain string in memory for the duration of this call.
// For production use, prefer a signer that fetches the key directly from
// your secrets manager; such as one backed by AWS KMS or HashiCorp Vault,
// so the raw key never touches application memory at all.
//
// Use NewSignerFromHex when the key is already in memory, for example when reading
// from a secrets manager SDK that returns strings, or in integration tests.
func NewSignerFromHex(hexKey string) (Signer, error) {
	raw := strings.TrimPrefix(hexKey, "0x")
	key, err := crypto.HexToECDSA(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &EnvSigner{
		key:     key,
		address: crypto.PubkeyToAddress(key.PublicKey),
	}, nil
}
