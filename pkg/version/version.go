// Package version exposes the supported Safe contract versions.
// Pass these constants to safe.New() to select which version to deploy.
//
// Versions at a glance:
//   - V130: legacy, widely deployed, still fully supported
//   - V141: recommended default, broad chain coverage, battle-tested
//   - V150: latest, chain coverage still expanding
//
// See [Safe's supported networks] for chain coverage per version.
//
// [Safe's supported networks]: https://docs.safe.global/advanced/smart-account-supported-networks
package version

import "github.com/spazzle-io/safekit/internal/versions"

// Version is a Safe contract version string.
type Version = versions.Version

const (
	// V130 is widely deployed and still fully supported.
	// New projects should prefer V141.
	V130 = versions.Version130

	// V141 is the recommended default. Broad chain coverage,
	// audited, and battle-tested in production.
	V141 = versions.Version141

	// V150 is the latest Safe release. Chain coverage is still
	// expanding. Check [Safe's supported networks] before using on a
	// specific chain.
	//
	// [Safe's supported networks]: https://docs.safe.global/advanced/smart-account-supported-networks
	V150 = versions.Version150
)
