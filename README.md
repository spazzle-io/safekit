<p align="center">
  <a href="https://github.com/spazzle-io/safekit">
    <img src="assets/safekit-logo.png" alt="safekit" width="400">
  </a>
</p>

<p align="center">
  <em>Go library for predicting and deploying Gnosis Safe multisig wallets on any EVM chain.</em>
</p>

<p align="center">
  <img src="https://github.com/spazzle-io/safekit/actions/workflows/ci.yml/badge.svg" alt="CI Tests">
  <img src="https://github.com/spazzle-io/safekit/actions/workflows/integration-tests.yml/badge.svg" alt="Integration Tests">
  <a href="https://pkg.go.dev/github.com/spazzle-io/safekit">
    <img src="https://pkg.go.dev/badge/github.com/spazzle-io/safekit.svg" alt="Go Reference">
  </a>
  <a href="https://codecov.io/gh/spazzle-io/safekit">
    <img src="https://codecov.io/gh/spazzle-io/safekit/graph/badge.svg?token=L3AFHQO29M" alt="codecov">
  </a>
  <a href="https://results.pre-commit.ci/latest/github/spazzle-io/safekit/main">
    <img src="https://results.pre-commit.ci/badge/github/spazzle-io/safekit/main.svg" alt="pre-commit.ci status">
  </a>
</p>

---

There is no official Go SDK for Gnosis Safe (now Safe{Wallet}). SafeKit fills that gap.

It lets you predict the address a Safe will be deployed to before it exists on-chain, and deploy it when you are ready. The predicted address is verified against the deployed address on every deployment so you always know they match.

Supports Safe v1.3.0, v1.4.1, and v1.5.0 on any EVM-compatible chain.

## Requirements

- Go 1.24+
- An Ethereum-compatible JSON-RPC endpoint
- A funded admin wallet to pay for gas (never implicitly added as a Safe owner)

## Installation

```bash
go get github.com/spazzle-io/safekit
```

Full API reference is available on [pkg.go.dev](https://pkg.go.dev/github.com/spazzle-io/safekit).

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ethereum/go-ethereum/common"
    "github.com/spazzle-io/safekit/pkg/chain"
    "github.com/spazzle-io/safekit/pkg/safe"
    "github.com/spazzle-io/safekit/pkg/signer"
    "github.com/spazzle-io/safekit/pkg/version"
)

func main() {
    s, err := signer.NewEnvSigner("ADMIN_WALLET_PRIVATE_KEY")
    if err != nil {
        log.Fatal(err)
    }

    eth, err := safe.Dial("RPC_URL")
    if err != nil {
        log.Fatal(err)
    }
    defer eth.Close()

    client, err := safe.New(safe.Options{
        Chain:   chain.Ethereum,
        Client:  eth,
        Signer:  s,
        Version: version.V141,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    owners := []common.Address{
        common.HexToAddress("0x1111111111111111111111111111111111111111"),
        common.HexToAddress("0x2222222222222222222222222222222222222222"),
        common.HexToAddress("0x3333333333333333333333333333333333333333"),
    }

    salt, err := safe.RandomSalt()
    if err != nil {
        log.Fatal(err)
    }

    addr, err := client.PredictAddress(owners, 2, salt)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Safe will be deployed to:", addr.Hex())

    result, err := client.Deploy(context.Background(), owners, 2, salt)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Deployed to:", result.SafeAddress.Hex())
    fmt.Println("Transaction:", result.TxHash.Hex())
}
```

## Predicting addresses

The same owners, threshold, and salt always produce the same address on the same chain and Safe version. This lets you know the wallet address before it exists on-chain, so you can fund it in advance and deploy later.

```go
// predict and store the address before deploying
addr, err := client.PredictAddress(owners, threshold, []byte(userID))

// deploy when ready — result.SafeAddress always matches addr
result, err := client.Deploy(ctx, owners, threshold, []byte(userID))
```

Prediction makes no network calls and costs no gas.

If you do not need a reproducible address, generate a random salt with `safe.RandomSalt()` instead of deriving one from a stable value like a user ID.

## Deployment patterns

`Deploy` submits the transaction and blocks until it is mined. For more control:

```go
// submit without waiting for the transaction to mine
txHash, err := client.SubmitDeployment(ctx, owners, threshold, salt)

// wait for it later
result, err := client.WaitForDeployment(ctx, owners, threshold, salt, txHash)

// or poll yourself
deployed, err := client.IsDeployed(ctx, predictedAddr)
```

## Concurrency

A `safe.Client` is safe for concurrent use. Multiple goroutines may call any method concurrently.

### Single process

By default, SafeKit manages transaction nonces in memory. This works when one process owns the signer wallet on a given chain. Multiple goroutines calling `Deploy` or `SubmitDeployment` on the same client queue safely and each receive a unique nonce.

### Multiple processes

If multiple processes share the same signer on the same chain, use the Redis-backed nonce manager:

```go
import (
    "github.com/redis/go-redis/v9"
    nonceredis "github.com/spazzle-io/safekit/pkg/nonce/redis"
)

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

nm, err := nonceredis.NewNonceManager(nonceredis.Options{
    Redis: rdb,
})
if err != nil {
    log.Fatal(err)
}

client, err := safe.New(safe.Options{
    Chain:        chain.Polygon,
    Client:       eth,
    Signer:       s,
    Version:      version.V141,
    NonceManager: nm,
})
```

All clients pointing at the same Redis instance and using the same signer on the same chain coordinate automatically. Only one client submits a transaction at a time.

> [!WARNING]
> SafeKit assumes no external actor is submitting transactions from the same wallet on the same chain while it is running. If something outside SafeKit uses the wallet concurrently, nonce conflicts may occur.

## Configuration

`safe.New` accepts an `Options` struct:

```go
client, err := safe.New(safe.Options{
    Chain:         chain.Base,
    Client:        eth,
    Signer:        s,
    Version:       version.V141,
    NonceManager:  nm,              // optional. defaults to a local in-memory nonce manager
    DeployTimeout: 3 * time.Minute, // optional. defaults to 2 minutes
    GasMultiplier: 1.3,             // optional. defaults to 1.2
})
```

Use `safe.Dial` if you do not already have an `*ethclient.Client`:

```go
eth, err := safe.Dial("RPC_URL")
if err != nil {
    log.Fatal(err)
}
defer eth.Close()
```

`Signer` is the wallet that pays for gas. For production workloads, prefer a signer backed by a hardware security
module or secrets manager. If your private key is already in memory, use `signer.NewSignerFromHex` instead of `signer.NewEnvSigner`.

## Testing

SafeKit ships a mock client that implements `safe.Client`. It uses real CREATE2 math so predicted and deployed addresses always match, but makes no network calls.

```go
import "github.com/spazzle-io/safekit/testing/mock"

client := mock.NewClient()

addr, err := client.PredictAddress(owners, threshold, salt)
result, err := client.Deploy(ctx, owners, threshold, salt)
// result.SafeAddress == addr, always
```

To test error handling, use `ForceError` or `ForceErrors`:

```go
// single error
client.ForceError(safe.ErrTransactionReverted)
_, err := client.Deploy(ctx, owners, threshold, salt)
// err is safe.ErrTransactionReverted

// sequential errors for testing retry logic
client.ForceErrors(safe.ErrDeployTimeout, safe.ErrDeployTimeout, nil)
// first two calls fail, third succeeds
```

> [!TIP]
> For integration testing against a real chain without spending real funds, run a local chain with [Anvil](https://www.getfoundry.sh/anvil). It starts with pre-funded accounts, a known chain ID, and a deterministic RPC URL, so you can wire it directly into your test config.

## Supported versions

| Version | Status      | Notes                                      |
|---------|-------------|--------------------------------------------|
| v1.3.0  | Supported   | Legacy. Prefer v1.4.1 for new deployments. |
| v1.4.1  | Recommended | Broad chain coverage, battle-tested.       |
| v1.5.0  | Supported   | Latest. Chain coverage still expanding.    |

Check [Safe's supported networks](https://docs.safe.global/advanced/smart-account-supported-networks) for chain coverage per version.

## Supported chains

SafeKit ships with built-in support for the following chains:

| Chain                   | ID       |
|-------------------------|----------|
| Local (Anvil / Hardhat) | 31337    |
| Ethereum                | 1        |
| Sepolia                 | 11155111 |
| Polygon                 | 137      |
| Polygon zkEVM           | 1101     |
| Polygon Amoy            | 80002    |
| Arbitrum One            | 42161    |
| Arbitrum Nova           | 42170    |
| Arbitrum Sepolia        | 421614   |
| Base                    | 8453     |
| Base Sepolia            | 84532    |
| Optimism                | 10       |
| Optimism Sepolia        | 11155420 |
| BNB Smart Chain         | 56       |
| BNB Smart Chain Testnet | 97       |

The full list is in `pkg/chain/known.go`. If your target chain is not listed, register it before calling `safe.New`:

```go
err := chain.Register(&chain.Chain{
    ID:   big.NewInt(12345),
    Name: "my-chain",
    IsL2: true,
})

c, err := chain.Lookup(big.NewInt(12345))

client, err := safe.New(safe.Options{
    Chain: c,
    ...
})
```

Alternatively, if you want the chain included in SafeKit by default, open a pull request following these steps in [CONTRIBUTING.md](CONTRIBUTING.md#adding-support-for-a-new-chain).

The chain must have Safe contracts deployed on it. Check the [Safe deployments registry](https://github.com/safe-global/safe-deployments) to confirm.

If you are running a local fork of a known chain, set `ForksChainID` to tell SafeKit which chain's contract addresses to use.
Transactions are signed with your local chain ID with no replay risk on the source chain.

```go
chain.Register(&chain.Chain{
    ID:           big.NewInt(31337),
    Name:         "local",
    IsL2:         false,
    ForksChainID: big.NewInt(11155111), // use Ethereum Sepolia's contract addresses
})
```

This is the recommended setup for local development with Anvil or Hardhat.

## License

This project is licensed under the terms of the MIT license.
