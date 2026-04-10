package versions

import (
	"errors"
	"math/big"
	"testing"
)

const validJSON = `{
	"deployments": {
		"canonical": {
			"address": "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67",
			"codeHash": "0xabc123"
		},
		"eip155": {
			"address": "0xC22834581EbC8527d974F8a1c97E1bEA4EF910BC",
			"codeHash": "0xabc123"
		}
	},
	"networkAddresses": {
		"1":   ["canonical", "eip155"],
		"137": "canonical",
		"999": ["eip155"]
	},
	"abi": []
}`

const zkSyncOnlyJSON = `{
	"deployments": {
		"zksync": {
			"address": "0x1234567890123456789012345678901234567890",
			"codeHash": "0xabc123"
		}
	},
	"networkAddresses": {
		"1": "zksync"
	},
	"abi": []
}`

const zkSyncWithFallbackJSON = `{
    "deployments": {
        "zksync": {
            "address": "0x1234567890123456789012345678901234567890",
            "codeHash": "0xabc123"
        },
        "canonical": {
            "address": "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67",
            "codeHash": "0xabc123"
        }
    },
    "networkAddresses": {
        "1": ["zksync", "canonical"]
    },
    "abi": []
}`

func TestLoad_CanonicalAddress(t *testing.T) {
	got, err := Load([]byte(validJSON), big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestLoad_SingleStringAddress(t *testing.T) {
	got, err := Load([]byte(validJSON), big.NewInt(137))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestLoad_EIP155PreferredAddress(t *testing.T) {
	got, err := Load([]byte(validJSON), big.NewInt(999))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0xC22834581EbC8527d974F8a1c97E1bEA4EF910BC"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestLoad_UnsupportedChain(t *testing.T) {
	_, err := Load([]byte(validJSON), big.NewInt(99999))
	if err == nil {
		t.Fatal("expected error for unsupported chain, got nil")
	}
	if !errors.Is(err, ErrVersionNotOnChain) {
		t.Errorf("expected ErrVersionNotOnChain, got %v", err)
	}
}

func TestLoad_ZkSyncRejected(t *testing.T) {
	_, err := Load([]byte(zkSyncOnlyJSON), big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for zksync-only chain, got nil")
	}
	if !errors.Is(err, ErrVersionNotOnChain) {
		t.Errorf("expected ErrVersionNotOnChain, got %v", err)
	}
}

func TestLoad_ZkSyncWithFallback(t *testing.T) {
	got, err := Load([]byte(zkSyncWithFallbackJSON), big.NewInt(1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67"
	if got.Address.Hex() != want {
		t.Errorf("got address %s, want %s", got.Address.Hex(), want)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	_, err := Load([]byte(`not json`), big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_EmptyJSON(t *testing.T) {
	_, err := Load([]byte(`{}`), big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for empty JSON, got nil")
	}
}
