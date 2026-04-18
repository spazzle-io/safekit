# Changelog

## [1.4.0](https://github.com/spazzle-io/safekit/compare/v1.3.0...v1.4.0) (2026-04-18)


### Features

* support forking of known chains ([#24](https://github.com/spazzle-io/safekit/issues/24)) ([03421aa](https://github.com/spazzle-io/safekit/commit/03421aa241c27f1f2bf3a894ff48099007aa86b7))


### Bug Fixes

* use same safe.Client across all integration tests to prevent nonce re-seeding ([#25](https://github.com/spazzle-io/safekit/issues/25)) ([d9913c3](https://github.com/spazzle-io/safekit/commit/d9913c38f3349b602fceea54b8b43d6439187e9a))

## [1.3.0](https://github.com/spazzle-io/safekit/compare/v1.2.0...v1.3.0) (2026-04-16)


### Features

* add local development chain (chain ID 31337) as a known chain ([#22](https://github.com/spazzle-io/safekit/issues/22)) ([3dd5c57](https://github.com/spazzle-io/safekit/commit/3dd5c57c83721a2ba4c93a7321e244eb0c5855f4))

## [1.2.0](https://github.com/spazzle-io/safekit/compare/v1.1.0...v1.2.0) (2026-04-15)


### Features

* add nonce manager and tx manager ([#15](https://github.com/spazzle-io/safekit/issues/15)) ([55722e4](https://github.com/spazzle-io/safekit/commit/55722e4da2fda924902e108cbf7706a5831e2842))


### Bug Fixes

* correct ProxyCreation log data length validation ([#17](https://github.com/spazzle-io/safekit/issues/17)) ([3530b12](https://github.com/spazzle-io/safekit/commit/3530b1294915fa59a734b49afca15154c933bc7a))

## [1.1.0](https://github.com/spazzle-io/safekit/compare/v1.0.1...v1.1.0) (2026-04-13)


### Features

* allow concurrent deployments ([#12](https://github.com/spazzle-io/safekit/issues/12)) ([143df76](https://github.com/spazzle-io/safekit/commit/143df76009034a26dafe029f245af4ced0b57daf))


### Bug Fixes

* fix concurrent deployments on same client ([#14](https://github.com/spazzle-io/safekit/issues/14)) ([2c0ee1e](https://github.com/spazzle-io/safekit/commit/2c0ee1e092ebe27416264cae47e29497005eb9b7))

## [1.0.1](https://github.com/spazzle-io/safekit/compare/v1.0.0...v1.0.1) (2026-04-11)


### Bug Fixes

* **test:** fix integration tests ([#7](https://github.com/spazzle-io/safekit/issues/7)) ([cbfc18b](https://github.com/spazzle-io/safekit/commit/cbfc18b2005e937cb88650c7f1abf0f3ffdf5d4b))
