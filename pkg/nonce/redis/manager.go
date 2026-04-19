// Package redis provides a distributed, Redis-backed implementation of nonce.Manager.
// It is safe for concurrent use across multiple goroutines and multiple processes or machines,
// as long as they all share the same Redis instance.
//
// Use this when multiple processes share the same signer wallet on the same chain.
// For single-process deployments, use pkg/nonce/local instead.
package redis

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/redis/go-redis/v9"

	internalnonce "github.com/spazzle-io/safekit/internal/nonce"
	pkgnonce "github.com/spazzle-io/safekit/pkg/nonce"
)

const (
	defaultLockTTL      = 30 * time.Second
	defaultPollInterval = 200 * time.Millisecond
)

// Options configures a Redis-backed NonceManager.
type Options struct {
	// Redis is the Redis client used for distributed locking and nonce coordination. Required.
	Redis redis.UniversalClient

	// InstanceID uniquely identifies this process or worker. It is used as the Redis lock token.
	// It must be unique across all workers sharing the same signer on the same chain. If empty, a random ID
	// is generated automatically. Only set this manually if you need meaningful IDs for observability.
	InstanceID string

	// LockTTL is how long the distributed lock is held before it automatically expires.
	// This is a safety net against crashed workers. Defaults to 30s.
	LockTTL time.Duration

	// PollInterval is how often a safe.Client instance will attempt to acquire the lock while waiting. Defaults to 200ms.
	PollInterval time.Duration

	// StaleNonceDelay is how long to wait before re-fetching the pending nonce from the chain after a failed broadcast.
	// Defaults to nonce.DefaultStaleNonceDelay.
	StaleNonceDelay time.Duration
}

// NonceManager is a distributed, Redis-backed nonce.Manager.
type NonceManager struct {
	source          internalnonce.Source
	rdb             redis.UniversalClient
	chainID         *big.Int
	address         common.Address
	instanceID      string
	lockTTL         time.Duration
	pollInterval    time.Duration
	staleNonceDelay time.Duration
	dirty           atomic.Bool
}

// NewNonceManager constructs a NonceManager from the given options.
func NewNonceManager(opts Options) (*NonceManager, error) {
	if opts.Redis == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	if opts.InstanceID == "" {
		instanceID, err := generateInstanceID()
		if err != nil {
			return nil, err
		}
		opts.InstanceID = instanceID
	}

	lockTTL := opts.LockTTL
	if lockTTL <= 0 {
		lockTTL = defaultLockTTL
	}

	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	staleNonceDelay := opts.StaleNonceDelay
	if staleNonceDelay <= 0 {
		staleNonceDelay = internalnonce.DefaultStaleNonceDelay
	}

	nm := &NonceManager{
		rdb:             opts.Redis,
		instanceID:      opts.InstanceID,
		lockTTL:         lockTTL,
		pollInterval:    pollInterval,
		staleNonceDelay: staleNonceDelay,
	}
	nm.dirty.Store(true)

	return nm, nil
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

	m.source = source
	m.chainID = chainID
	m.address = address

	return nil
}

// Acquire blocks until a nonce slot is available, returning the next nonce and a release function
// the caller must invoke exactly once after attempting to broadcast the transaction.
func (m *NonceManager) Acquire(ctx context.Context) (uint64, pkgnonce.Slot, error) {
	if m.source == nil {
		return 0, nil, fmt.Errorf("nonce source required")
	}

	if err := m.acquireLock(ctx); err != nil {
		return 0, nil, err
	}

	n, err := m.nextNonce(ctx)
	if err != nil {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = m.releaseLock(releaseCtx)
		return 0, nil, fmt.Errorf("failed to fetch nonce: %w", err)
	}

	return n, &slot{manager: m}, nil
}

func (m *NonceManager) lockKey() string {
	return fmt.Sprintf("safekit:nonce:%s:%s:lock", m.chainID.String(), strings.ToLower(m.address.Hex()))
}

func (m *NonceManager) counterKey() string {
	return fmt.Sprintf("safekit:nonce:%s:%s", m.chainID.String(), strings.ToLower(m.address.Hex()))
}

func (m *NonceManager) acquireLock(ctx context.Context) error {
	for {
		ok, err := m.rdb.SetNX(ctx, m.lockKey(), m.instanceID, m.lockTTL).Result()
		if err != nil {
			return fmt.Errorf("failed attempt to acquire lock: %w", err)
		}
		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.pollInterval):
		}
	}
}

func (m *NonceManager) releaseLock(ctx context.Context) error {
	n, err := releaseLockScript.Run(ctx, m.rdb, []string{m.lockKey()}, m.instanceID).Int()
	if err != nil {
		return fmt.Errorf("failed to run release lock script: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("failed to release lock. lock not held by requesting instance")
	}

	return nil
}

// nextNonce returns the next nonce to use. If the nonce is dirty it waits StaleNonceDelay then re-fetches
// from the chain. Must be called while holding the distributed lock.
func (m *NonceManager) nextNonce(ctx context.Context) (uint64, error) {
	if m.dirty.Load() {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(m.staleNonceDelay):
		}

		n, err := m.source.PendingNonceAt(ctx, m.address)
		if err != nil {
			return 0, err
		}

		if err := m.rdb.Set(ctx, m.counterKey(), strconv.FormatUint(n+1, 10), 0).Err(); err != nil {
			return 0, fmt.Errorf("redis counter reseed failed: %w", err)
		}

		m.dirty.Store(false)

		return n, nil
	}

	exists, err := m.rdb.Exists(ctx, m.counterKey()).Result()
	if err != nil {
		return 0, fmt.Errorf("redis EXISTS error: %w", err)
	}

	if exists == 0 {
		n, err := m.source.PendingNonceAt(ctx, m.address)
		if err != nil {
			return 0, fmt.Errorf("chain nonce seed failed: %w", err)
		}

		if err := m.rdb.Set(ctx, m.counterKey(), strconv.FormatUint(n+1, 10), 0).Err(); err != nil {
			return 0, fmt.Errorf("redis counter seed failed: %w", err)
		}

		return n, nil
	}

	val, err := m.rdb.Get(ctx, m.counterKey()).Result()
	if err != nil {
		return 0, fmt.Errorf("redis GET error: %w", err)
	}

	current, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("redis counter parse error: %w", err)
	}

	if err := m.rdb.Set(ctx, m.counterKey(), strconv.FormatUint(current+1, 10), 0).Err(); err != nil {
		return 0, fmt.Errorf("redis counter increment failed: %w", err)
	}

	return current, nil
}

func generateInstanceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate instance ID: %w", err)
	}

	return fmt.Sprintf("%x", b), nil
}

var (
	_ pkgnonce.Manager       = (*NonceManager)(nil)
	_ internalnonce.Initable = (*NonceManager)(nil)
)
