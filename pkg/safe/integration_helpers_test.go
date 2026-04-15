//go:build integration

package safe

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
	"github.com/spazzle-io/safekit/pkg/chain"
	nonceredis "github.com/spazzle-io/safekit/pkg/nonce/redis"
	"github.com/spazzle-io/safekit/pkg/signer"
	"github.com/spazzle-io/safekit/pkg/version"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

type integrationEnv struct {
	eth     *ethclient.Client
	signer  signer.Signer
	chain   *chain.Chain
	version version.Version
}

// integrationClientFromEnv builds a Client from environment variables.
//
// Required env vars:
//
//	SAFEKIT_TEST_RPC_URL   — RPC endpoint for the test network
//	SAFEKIT_TEST_ADMIN_KEY — hex-encoded private key for the admin wallet
//	SAFEKIT_TEST_CHAIN_ID  — chain ID as a decimal integer
//	SAFEKIT_TEST_VERSION   — Safe version e.g. "1.4.1"
func integrationClientFromEnv(t *testing.T) Client {
	t.Helper()

	env := integrationEnvFromEnv(t)

	client, err := New(Options{
		Chain:         env.chain,
		Client:        env.eth,
		Signer:        env.signer,
		Version:       env.version,
		DeployTimeout: 2 * time.Minute,
	})
	if err != nil {
		if isVersionNotOnChain(err) {
			t.Skipf("version %s not deployed on chain %s. Skipping", env.version, env.chain.Name)
		}
		t.Fatalf("failed to create client: %v", err)
	}

	t.Cleanup(func() { client.Close() })
	return client
}

func integrationEnvFromEnv(t *testing.T) *integrationEnv {
	t.Helper()

	rpcURL := requireEnv(t, "SAFEKIT_TEST_RPC_URL")
	adminKey := requireEnv(t, "SAFEKIT_TEST_ADMIN_KEY")
	chainIDStr := requireEnv(t, "SAFEKIT_TEST_CHAIN_ID")
	versionStr := requireEnv(t, "SAFEKIT_TEST_VERSION")

	chainIDInt, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid SAFEKIT_TEST_CHAIN_ID %q: %v", chainIDStr, err)
	}

	c, err := chain.Lookup(big.NewInt(chainIDInt))
	if err != nil {
		t.Fatalf("unsupported chain ID %d: %v", chainIDInt, err)
	}

	s, err := signer.NewSignerFromHex(adminKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	eth, err := Dial(rpcURL)
	if err != nil {
		t.Fatalf("failed to dial RPC: %v", err)
	}
	t.Cleanup(func() { eth.Close() })

	return &integrationEnv{
		eth:     eth,
		signer:  s,
		chain:   c,
		version: version.Version(versionStr),
	}
}

// integrationRedisClientFromEnv builds a Client using a Redis-backed nonce manager.
// Requires SAFEKIT_TEST_REDIS_URL in addition to the standard env vars.
// Skips the test if SAFEKIT_TEST_REDIS_URL is not set.
func integrationRedisClientFromEnv(t *testing.T, env *integrationEnv, instanceID string) Client {
	t.Helper()

	redisURL := requireEnv(t, "SAFEKIT_TEST_REDIS_URL")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("invalid SAFEKIT_TEST_REDIS_URL %q: %v", redisURL, err)
	}

	rdb := redis.NewClient(opt)
	t.Cleanup(func() { _ = rdb.Close() })

	nm, err := nonceredis.NewNonceManager(nonceredis.Options{
		Redis:      rdb,
		InstanceID: instanceID,
	})
	if err != nil {
		t.Fatalf("failed to create redis nonce manager: %v", err)
	}

	client, err := New(Options{
		Chain:         env.chain,
		Client:        env.eth,
		Signer:        env.signer,
		Version:       env.version,
		NonceManager:  nm,
		DeployTimeout: 3 * time.Minute,
	})
	if err != nil {
		if isVersionNotOnChain(err) {
			t.Skipf("version %s not deployed on chain %s. Skipping", env.version, env.chain.Name)
		}
		t.Fatalf("failed to create redis client: %v", err)
	}

	t.Cleanup(func() { client.Close() })
	return client
}

// requireEnv returns the value of the given environment variable, skipping the test if it is not set.
func requireEnv(t *testing.T, key string) string {
	t.Helper()

	v := os.Getenv(key)
	if v == "" {
		t.Skipf("%s env var not set. Skipping integration test", key)
	}

	return v
}

// integrationOwners returns a fixed set of owner addresses for use in tests.
func integrationOwners(t *testing.T) []common.Address {
	t.Helper()

	return []common.Address{
		signer.NewMockSigner(1).Address(),
		signer.NewMockSigner(2).Address(),
		signer.NewMockSigner(3).Address(),
	}
}

func isVersionNotOnChain(err error) bool {
	return err != nil && errors.Is(err, ErrVersionNotOnChain)
}
