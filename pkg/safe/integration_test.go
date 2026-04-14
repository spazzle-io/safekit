//go:build integration

package safe

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"sync"
	"testing"
)

func TestIntegration_PredictAndDeploy(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	salt, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error generating salt: %v", err)
	}

	predicted, err := client.PredictAddress(owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error predicting address: %v", err)
	}
	t.Logf("predicted: %s", predicted.Hex())

	deployed, err := client.IsDeployed(ctx, predicted)
	if err != nil {
		t.Fatalf("unexpected error checking if address is deployed: %v", err)
	}
	if deployed {
		t.Fatal("address should not be deployed before Deploy is called")
	}

	result, err := client.Deploy(ctx, owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error deploying: %v", err)
	}

	t.Logf("deployed: %s (tx: %s, block: %d, gas: %d)",
		result.SafeAddress.Hex(),
		result.TxHash.Hex(),
		result.BlockNumber,
		result.GasUsed,
	)

	if result.SafeAddress != predicted {
		t.Errorf("predicted %s but deployed to %s", predicted.Hex(), result.SafeAddress.Hex())
	}

	deployed, err = client.IsDeployed(ctx, result.SafeAddress)
	if err != nil {
		t.Fatalf("unexpected error checking if address is deployed: %v", err)
	}
	if !deployed {
		t.Error("address should be deployed after Deploy is called")
	}
}

func TestIntegration_AlreadyDeployed(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	// Fixed salt makes this idempotent across runs.
	salt := []byte("safekit-integration-already-deployed")

	// First deploy may already exist from a previous run. Ignore error.
	_, _ = client.Deploy(ctx, owners, 2, salt)

	// Second deploy must always return ErrAddressAlreadyDeployed.
	_, err := client.Deploy(ctx, owners, 2, salt)
	if !errors.Is(err, ErrAddressAlreadyDeployed) {
		t.Errorf("expected ErrAddressAlreadyDeployed, got %v", err)
	}
}

func TestIntegration_SubmitAndWait(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	salt, err := RandomSalt()
	if err != nil {
		t.Fatalf("unexpected error generating salt: %v", err)
	}

	predicted, err := client.PredictAddress(owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error predicting address: %v", err)
	}
	t.Logf("predicted: %s", predicted.Hex())

	txHash, err := client.SubmitDeployment(ctx, owners, 2, salt)
	if err != nil {
		t.Fatalf("unexpected error submitting deployment: %v", err)
	}
	t.Logf("submitted tx: %s", txHash.Hex())

	result, err := client.WaitForDeployment(ctx, owners, 2, salt, txHash)
	if err != nil {
		t.Fatalf("unexpected error waiting for deployment: %v", err)
	}

	t.Logf("deployed: %s (tx: %s, block: %d, gas: %d)",
		result.SafeAddress.Hex(),
		result.TxHash.Hex(),
		result.BlockNumber,
		result.GasUsed,
	)

	if result.SafeAddress != predicted {
		t.Errorf("predicted %s but deployed to %s", predicted.Hex(), result.SafeAddress.Hex())
	}
}

func TestIntegration_ConcurrentDeploy_LocalNonce(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	const concurrency = 5

	type result struct {
		predicted common.Address
		deployed  common.Address
		err       error
	}

	results := make([]result, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			salt, err := RandomSalt()
			if err != nil {
				results[i] = result{err: fmt.Errorf("goroutine %d: failed to generate salt: %w", i, err)}
				return
			}

			predicted, err := client.PredictAddress(owners, 2, salt)
			if err != nil {
				results[i] = result{err: fmt.Errorf("goroutine %d: failed to predict address: %w", i, err)}
				return
			}

			r, err := client.Deploy(ctx, owners, 2, salt)
			if err != nil {
				results[i] = result{err: fmt.Errorf("goroutine %d: failed to deploy: %w", i, err)}
				return
			}

			results[i] = result{predicted: predicted, deployed: r.SafeAddress}
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Errorf("concurrent deploy %d: %v", i, r.err)
			continue
		}

		if r.deployed != r.predicted {
			t.Errorf("concurrent deploy %d: predicted %s but deployed to %s",
				i, r.predicted.Hex(), r.deployed.Hex())
		} else {
			t.Logf("concurrent deploy %d: deployed %s", i, r.deployed.Hex())
		}
	}
}

func TestIntegration_ConcurrentSubmitAndWait(t *testing.T) {
	client := integrationClientFromEnv(t)
	ctx := context.Background()
	owners := integrationOwners(t)

	const count = 5

	type pending struct {
		salt      []byte
		predicted common.Address
		txHash    common.Hash
	}

	pendings := make([]pending, count)
	for i := 0; i < count; i++ {
		salt, err := RandomSalt()
		if err != nil {
			t.Fatalf("submit %d: failed to generate salt: %v", i, err)
		}

		predicted, err := client.PredictAddress(owners, 2, salt)
		if err != nil {
			t.Fatalf("submit %d: failed to predict address: %v", i, err)
		}

		txHash, err := client.SubmitDeployment(ctx, owners, 2, salt)
		if err != nil {
			t.Fatalf("submit %d: failed to submit: %v", i, err)
		}

		t.Logf("submit %d: tx %s", i, txHash.Hex())
		pendings[i] = pending{salt: salt, predicted: predicted, txHash: txHash}
	}

	type result struct {
		idx  int
		addr common.Address
		err  error
	}

	resultCh := make(chan result, count)

	for i, p := range pendings {
		go func(i int, p pending) {
			r, err := client.WaitForDeployment(ctx, owners, 2, p.salt, p.txHash)
			if err != nil {
				resultCh <- result{idx: i, err: fmt.Errorf("wait %d: %w", i, err)}
				return
			}
			resultCh <- result{idx: i, addr: r.SafeAddress}
		}(i, p)
	}

	for range count {
		r := <-resultCh
		if r.err != nil {
			t.Errorf("%v", r.err)
			continue
		}

		if r.addr != pendings[r.idx].predicted {
			t.Errorf("wait %d: predicted %s but deployed to %s",
				r.idx, pendings[r.idx].predicted.Hex(), r.addr.Hex())
		} else {
			t.Logf("wait %d: deployed %s", r.idx, r.addr.Hex())
		}
	}
}

// Requires SAFEKIT_TEST_REDIS_URL env var to be set. Skips if not.
func TestIntegration_ConcurrentDeploy_RedisNonce(t *testing.T) {
	ctx := context.Background()
	owners := integrationOwners(t)

	env := integrationEnvFromEnv(t)

	const workers = 3
	clients := make([]Client, workers)
	for i := 0; i < workers; i++ {
		clients[i] = integrationRedisClientFromEnv(t, env, fmt.Sprintf("worker-%d", i+1))
	}

	// Each worker deploys 2 Safes. 6 total deployments.
	const deploysPerWorker = 2

	type result struct {
		workerID  int
		deployIdx int
		predicted common.Address
		deployed  common.Address
		err       error
	}

	resultCh := make(chan result, workers*deploysPerWorker)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)

		go func(w int, c Client) {
			defer wg.Done()

			for d := 0; d < deploysPerWorker; d++ {
				salt, err := RandomSalt()
				if err != nil {
					resultCh <- result{
						workerID:  w,
						deployIdx: d,
						err:       fmt.Errorf("worker %d deploy %d: failed to generate salt: %w", w, d, err),
					}
					return
				}

				predicted, err := c.PredictAddress(owners, 2, salt)
				if err != nil {
					resultCh <- result{
						workerID:  w,
						deployIdx: d,
						err:       fmt.Errorf("worker %d deploy %d: failed to predict: %w", w, d, err),
					}
					return
				}

				r, err := c.Deploy(ctx, owners, 2, salt)
				if err != nil {
					resultCh <- result{
						workerID:  w,
						deployIdx: d,
						err:       fmt.Errorf("worker %d deploy %d: failed to deploy: %w", w, d, err),
					}
					return
				}

				resultCh <- result{
					workerID:  w,
					deployIdx: d,
					predicted: predicted,
					deployed:  r.SafeAddress,
				}
			}
		}(w, clients[w])
	}

	wg.Wait()
	close(resultCh)

	for r := range resultCh {
		if r.err != nil {
			t.Errorf("%v", r.err)
			continue
		}

		if r.deployed != r.predicted {
			t.Errorf("worker %d deploy %d: predicted %s but deployed to %s",
				r.workerID, r.deployIdx, r.predicted.Hex(), r.deployed.Hex())
		} else {
			t.Logf("worker %d deploy %d: deployed %s", r.workerID, r.deployIdx, r.deployed.Hex())
		}
	}
}
