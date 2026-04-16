# Contributing to SafeKit

Thank you for your interest in contributing. This guide covers everything you
need to get started.

## Before you start

For anything beyond a typo fix, please open an issue before submitting a pull
request. This saves everyone time. We can discuss the approach before you
invest effort writing code.

Use the appropriate issue template:
- **Bug report** — include your Go version, SafeKit version, chain, Safe
  version, steps to reproduce, and the full error message
- **Feature request** — describe the use case and what SafeKit does not currently provide.
- **General inquiry** — for anything else

Pull requests should reference an open issue. The PR template will prompt you
for this.

## Setting up

You need Go 1.24+ and a working Go toolchain.

```bash
git clone https://github.com/spazzle-io/safekit.git
cd safekit
go mod download
```

Run the unit tests to confirm everything is working:

```bash
go test -v -race ./...
```

## Code style

SafeKit uses golangci-lint for linting and gofumpt for formatting.
The full linter configuration is in `.golangci.yml` at the root of the repository.

To run the linter locally:

```bash
golangci-lint run ./...
```

To auto-fix formatting issues:

```bash
golangci-lint run --fix ./...
```

## Commit messages

SafeKit uses [conventional commits](https://www.conventionalcommits.org).
Your PR title must follow this format:

```
<type>: <description>

feat: add support for Arbitrum chain
fix: correct salt derivation for nil input
docs: update README quick start example
chore: bump go-ethereum to v1.15.0
refactor: simplify deployer gas estimation
test: add integration test for v1.5.0
ci: pin golangci-lint to v2.11.4
```

The PR title is validated automatically when you open a pull request. The
allowed types are `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, and `ci`.

release-please uses these commit messages to generate the CHANGELOG and
determine version bumps. `feat` bumps the minor version, `fix` bumps the
patch version, and `feat!` or `BREAKING CHANGE` in the footer bumps the
major version.

## Running integration tests

Integration tests deploy real Safes to testnets and verify the predicted
address matches the deployed address. They are tagged with `//go:build integration`
and do not run during `go test ./...`.

Our CI runs unit tests automatically on every pull request. Integration tests
run on a weekly schedule and can be triggered manually by a maintainer when
reviewing a significant pull request. You do not need a funded wallet to
contribute. The maintainers handle integration verification before merging.

If you want to run integration tests locally, you will need a funded wallet
on the target testnet. Set these environment variables before running:

| Variable                  | Description                                                |
|---------------------------|------------------------------------------------------------|
| `SAFEKIT_TEST_RPC_URL`    | RPC endpoint for the target testnet                        |
| `SAFEKIT_TEST_ADMIN_KEY`  | Hex-encoded private key of the funded wallet               |
| `SAFEKIT_TEST_CHAIN_ID`   | Chain ID as a decimal integer e.g. `11155111` for Sepolia  |
| `SAFEKIT_TEST_VERSION`    | Safe version to test e.g. `1.4.1`                          |

```bash
SAFEKIT_TEST_CHAIN_ID=11155111 \
SAFEKIT_TEST_VERSION=1.4.1 \
SAFEKIT_TEST_RPC_URL=https://rpc.sepolia.org \
SAFEKIT_TEST_ADMIN_KEY=<FUNDED_WALLET_PRIVATE_KEY> \
go test -tags integration -timeout 15m -v ./pkg/safe/...
```

### Testing distributed nonce coordination

By default, the distributed nonce manager tests are skipped. To run them, you need
a Redis instance and set `SAFEKIT_TEST_REDIS_URL` environment variable. Everything else stays the same:

```bash
SAFEKIT_TEST_CHAIN_ID=11155111 \
SAFEKIT_TEST_VERSION=1.4.1 \
SAFEKIT_TEST_RPC_URL=https://rpc.sepolia.org \
SAFEKIT_TEST_ADMIN_KEY= \
SAFEKIT_TEST_REDIS_URL=redis://localhost:6379 \
go test -tags integration -timeout 15m -v ./pkg/safe/...
```

## Adding support for a new chain

If Safe contracts are deployed on a chain that SafeKit does not support yet,
adding it is straightforward.

First confirm the chain is in the
[Safe deployments registry](https://github.com/safe-global/safe-deployments).
If it is not there, Safe has not deployed their contracts on that chain and
there is nothing to add yet.

Add the chain to `pkg/chain/known.go`:

```go
var MyChain = &Chain{
    ID:   big.NewInt(12345),
    Name: "my-chain",
    IsL2: true,
}
```

Register it in the `init()` function in `pkg/chain/registry.go`:

Run the unit tests to confirm nothing broke:

```bash
go test ./pkg/chain/...
```

Then [run an integration test](#running-integration-tests) against your chain to confirm Safe deployment works end to end.
Include the transaction hash of a successful deployment in your pull request
so we can verify on-chain.

## Adding a new Safe version

When Safe ships a new version, here is how to add it to SafeKit.

**Step 1 — get the deployment JSON files**

Copy the deployment JSON files for the new version from
[safe-global/safe-deployments](https://github.com/safe-global/safe-deployments/tree/main/src/assets).
The filenames may differ between versions. You need three files; the Safe
singleton, the SafeL2 singleton, and the proxy factory. Place them in
`internal/versions/vX_Y_Z/` renamed to:

- `safe.json`: the Safe singleton
- `safe_l2.json`: the SafeL2 singleton
- `proxy_factory.json`: the proxy factory

Check the existing version directories to see how previous versions named
their source files if you are unsure which file is which.

**Step 2 — get the proxy creation code**

You need the bytecode that the factory uses to deploy each Safe proxy. To get
it, navigate to the proxy factory address on Etherscan (the address is in
`proxy_factory.json` under `networkAddresses` for any supported chain).
Go to the Contract tab, find the `proxyCreationCode` read function, and call
it. Copy the result; it is a hex string starting with `0x`.

You will use this in the next step.

**Step 3 — create the embed file**

Create `internal/versions/vX_Y_Z/embed.go` following the pattern in an
existing version. Paste the proxy creation code from the previous step as a
constant, without the `0x` prefix:

```go
package vX_Y_Z

import (
  _ "embed"
  "encoding/hex"
  "fmt"
  "github.com/spazzle-io/safekit/pkg/version"

  "github.com/spazzle-io/safekit/internal/versions"
)

const proxyCreationCodeHex = "608060405234..." // paste your result here, no 0x prefix

//go:embed safe.json
var safeJSON []byte

//go:embed safe_l2.json
var safeL2JSON []byte

//go:embed proxy_factory.json
var proxyFactoryJSON []byte

func init() {
  var err error
  proxyCreationCode, err := hex.DecodeString(proxyCreationCodeHex)
  if err != nil {
    panic(fmt.Sprintf("invalid proxyCreationCode hex: %v", err))
  }

  versions.Register(versions.NewBaseDeployment(
    version.V150,
    safeJSON,
    safeL2JSON,
    proxyFactoryJSON,
    proxyCreationCode,
  ))
}
```

**Step 4 — add the version constant**

Add the new version constant to `internal/versions/version.go` and
re-export it in `pkg/version/version.go`.

**Step 5 — register the blank import**

Add a blank import for the new version package in `pkg/safe/options.go`:

```go
import (
    _ "github.com/spazzle-io/safekit/internal/versions/vX_Y_Z"
)
```

**Step 6 — add embed tests**

Add `internal/versions/vX_Y_Z/embed_test.go` following the pattern in an
existing version. At minimum test that the JSON files embed correctly and
that the proxy creation code starts with the expected EVM preamble.

**Step 7 — run the tests**

```bash
go test -v -race ./...
```

## License

By contributing to SafeKit you agree that your contributions will be licensed
under the MIT license.
