package safe

import (
	"crypto/rand"
	"fmt"
)

// generateRandomSalt generates a cryptographically random 32-byte salt.
func generateRandomSalt() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random salt: %w", err)
	}

	return b, nil
}
