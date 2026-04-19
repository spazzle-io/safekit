//go:build integration

package safe

import (
	"errors"
	"fmt"
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

// Integration tests require on the following:
//
// Required env vars:
//
//	SAFEKIT_TEST_RPC_URL   - RPC endpoint for the test network
//	SAFEKIT_TEST_ADMIN_KEY - hex-encoded private key for the admin wallet
//	SAFEKIT_TEST_CHAIN_ID  - chain ID as a decimal integer
//	SAFEKIT_TEST_VERSION   - Safe version e.g. "1.4.1"
//
// Optional env vars:
//
//  SAFEKIT_TEST_REDIS_URL - Redis connection URL. To be set when testing the redis nonce manager.

var (
	sharedEnv    *integrationEnv
	sharedClient Client
)

type integrationEnv struct {
	eth     *ethclient.Client
	signer  signer.Signer
	chain   *chain.Chain
	version version.Version
}

func TestMain(m *testing.M) {
	if err := setupSharedEnv(); err != nil {
		fmt.Println("skipping integration tests:", err)
		os.Exit(0)
	}

	code := m.Run()

	teardownSharedEnv()
	os.Exit(code)
}

func setupSharedEnv() error {
	rpcURL := os.Getenv("SAFEKIT_TEST_RPC_URL")
	adminKey := os.Getenv("SAFEKIT_TEST_ADMIN_KEY")
	chainIDStr := os.Getenv("SAFEKIT_TEST_CHAIN_ID")
	versionStr := os.Getenv("SAFEKIT_TEST_VERSION")

	if rpcURL == "" || adminKey == "" || chainIDStr == "" || versionStr == "" {
		return fmt.Errorf("one or more required env vars not set")
	}

	chainIDInt, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid SAFEKIT_TEST_CHAIN_ID %q: %w", chainIDStr, err)
	}

	c, err := chain.Lookup(big.NewInt(chainIDInt))
	if err != nil {
		return fmt.Errorf("unsupported chain ID %d: %w", chainIDInt, err)
	}

	s, err := signer.NewSignerFromHex(adminKey)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}

	eth, err := Dial(rpcURL)
	if err != nil {
		s.Close()
		return fmt.Errorf("failed to dial RPC: %w", err)
	}

	sharedEnv = &integrationEnv{
		eth:     eth,
		signer:  s,
		chain:   c,
		version: version.Version(versionStr),
	}

	client, err := New(Options{
		Chain:   sharedEnv.chain,
		Client:  sharedEnv.eth,
		Signer:  sharedEnv.signer,
		Version: sharedEnv.version,
	})
	if err != nil {
		teardownSharedEnv()
		return fmt.Errorf("failed to create client: %w", err)
	}

	sharedClient = client

	return nil
}

func teardownSharedEnv() {
	if sharedClient != nil {
		sharedClient.Close()
	}

	if sharedEnv != nil {
		sharedEnv.eth.Close()
	}
}

func integrationClientFromEnv(t *testing.T) Client {
	t.Helper()

	if sharedClient == nil {
		t.Skip("shared integration client not initialised")
	}

	return sharedClient
}

func integrationEnvFromEnv(t *testing.T) *integrationEnv {
	t.Helper()

	if sharedEnv == nil {
		t.Skip("shared integration env not initialised")
	}

	return sharedEnv
}

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
		Redis:           rdb,
		InstanceID:      instanceID,
		StaleNonceDelay: staleNonceDelay(env.chain),
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

func staleNonceDelay(chain *chain.Chain) time.Duration {
	d := 500 * time.Millisecond

	if chain.ID == big.NewInt(84532) {
		d = 1 * time.Second
	}
	if chain.ID == big.NewInt(11155111) {
		d = 2 * time.Second
	}

	return d
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
