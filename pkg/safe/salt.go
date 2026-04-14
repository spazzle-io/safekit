package safe

import (
	"crypto/rand"
	"fmt"
)

// RandomSalt generates a cryptographically random 32-byte salt suitable
// for use with Client.Deploy and Client.PredictAddress.
//
// Use this when you don't need a reproducible address. For deterministic addresses
// (e.g. one Safe per user derived from a user ID), provide your own salt instead.
func RandomSalt() ([]byte, error) {
	return generateRandomSalt()
}

func generateRandomSalt() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random salt: %w", err)
	}

	return b, nil
}
