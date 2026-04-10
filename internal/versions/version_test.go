package versions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type stubDeployment struct {
	version           Version
	proxyCreationCode []byte
}

func (s *stubDeployment) Version() Version { return s.version }

func (s *stubDeployment) ProxyFactory(_ *big.Int) (*ParsedDeployment, error) {
	return &ParsedDeployment{
		Address: common.HexToAddress("0x1234"),
		ABI:     abi.ABI{},
	}, nil
}

func (s *stubDeployment) Singleton(_ *big.Int, _ bool) (*ParsedDeployment, error) {
	return &ParsedDeployment{
		Address: common.HexToAddress("0x5678"),
		ABI:     abi.ABI{},
	}, nil
}

func (s *stubDeployment) ProxyCreationCode() []byte {
	return s.proxyCreationCode
}

func TestRegister_And_Get(t *testing.T) {
	v := Version("test.0.1")
	Register(&stubDeployment{
		version:           v,
		proxyCreationCode: []byte{0x60, 0x80},
	})

	got, err := Get(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Version() != v {
		t.Errorf("got version %q, want %q", got.Version(), v)
	}
}

func TestGet_UnknownVersion(t *testing.T) {
	_, err := Get("99.99.99")
	if err == nil {
		t.Fatal("expected error for unknown version, got nil")
	}
	if !errors.Is(err, ErrUnknownVersion) {
		t.Errorf("expected ErrUnknownVersion, got %v", err)
	}
}

func TestRegister_Overwrites(t *testing.T) {
	v := Version("test.0.2")

	first := &stubDeployment{version: v, proxyCreationCode: []byte{0x01}}
	second := &stubDeployment{version: v, proxyCreationCode: []byte{0x02}}

	Register(first)
	Register(second)

	got, err := Get(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// second registration should win
	if got != second {
		t.Error("expected second registration to overwrite first")
	}
}
