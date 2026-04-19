// Package local provides a single-process, in-memory implementation of nonce.Manager.
// It is the default NonceManager used by safe.Client when no NonceManager is provided to safe.New.
//
// It is safe for concurrent use by multiple goroutines within one process that share a safe.Client.
// For deployments where multiple processes share the same signer on the same chain, use pkg/nonce/redis instead.
package local

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	internalnonce "github.com/spazzle-io/safekit/internal/nonce"
	"github.com/spazzle-io/safekit/pkg/nonce"
)

// Options configures a local NonceManager.
type Options struct {
	// StaleNonceDelay is how long to wait before re-fetching the pending nonce from the chain after a failed broadcast.
	// Defaults to nonce.DefaultStaleNonceDelay.
	StaleNonceDelay time.Duration
}

// NonceManager is a single-process, in-memory nonce.Manager.
type NonceManager struct {
	source          internalnonce.Source
	address         common.Address
	staleNonceDelay time.Duration

	mu       sync.Mutex
	nonce    uint64
	hasNonce bool
	dirty    bool

	inflightCh chan struct{}
}

// NewNonceManager constructs a NonceManager from the given options.
func NewNonceManager(opts Options) *NonceManager {
	delay := opts.StaleNonceDelay
	if delay <= 0 {
		delay = internalnonce.DefaultStaleNonceDelay
	}

	ch := make(chan struct{}, 1)
	ch <- struct{}{}

	return &NonceManager{
		staleNonceDelay: delay,
		inflightCh:      ch,
	}
}

// Init injects chain-specific context required for nonce management.
// It is called automatically by safe.New and should not be called directly.
// Init must complete before Acquire is called. This is guaranteed when the nonce manager is passed to safe.New.
func (m *NonceManager) Init(source internalnonce.Source, chainID *big.Int, address common.Address) error {
	if source == nil {
		return fmt.Errorf("source is required")
	}
	if chainID == nil {
		return fmt.Errorf("chain ID is required")
	}
	if (address == common.Address{}) {
		return fmt.Errorf("address is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.source = source
	m.address = address

	return nil
}

// Acquire blocks until a nonce slot is available, returning the next nonce and a release function
// the caller must invoke exactly once after attempting to broadcast the transaction.
func (m *NonceManager) Acquire(ctx context.Context) (uint64, nonce.Slot, error) {
	m.mu.Lock()
	ready := m.source != nil
	m.mu.Unlock()

	if !ready {
		return 0, nil, fmt.Errorf("nonce source required")
	}

	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case <-m.inflightCh:
	}

	n, err := m.nextNonce(ctx)
	if err != nil {
		m.inflightCh <- struct{}{}
		return 0, nil, fmt.Errorf("failed to fetch nonce: %w", err)
	}

	return n, &slot{manager: m}, nil
}

// Reset is a no-op for the local in-memory NonceManager.
func (m *NonceManager) Reset(_ context.Context) error {
	return nil
}

// nextNonce returns the next nonce to use. If the nonce is dirty it waits StaleNonceDelay then re-fetches
// from the chain. Must be called while holding the inflight slot.
func (m *NonceManager) nextNonce(ctx context.Context) (uint64, error) {
	m.mu.Lock()
	dirty := m.dirty
	hasNonce := m.hasNonce
	m.mu.Unlock()

	if dirty {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(m.staleNonceDelay):
		}

		n, err := m.source.PendingNonceAt(ctx, m.address)
		if err != nil {
			return 0, err
		}

		m.mu.Lock()
		m.nonce = n
		m.hasNonce = true
		m.dirty = false
		m.mu.Unlock()

		return n, nil
	}

	if !hasNonce {
		n, err := m.source.PendingNonceAt(ctx, m.address)
		if err != nil {
			return 0, err
		}

		m.mu.Lock()
		m.nonce = n
		m.hasNonce = true
		m.mu.Unlock()

		return n, nil
	}

	m.mu.Lock()
	m.nonce++
	n := m.nonce
	m.mu.Unlock()

	return n, nil
}

var (
	_ nonce.Manager          = (*NonceManager)(nil)
	_ internalnonce.Initable = (*NonceManager)(nil)
)
