# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

In addition to the standard sections, this changelog uses a **### Breaking**
section to explicitly document backwards-incompatible changes that would
otherwise appear under ### Changed. Entries under ### Breaking trigger a major
version bump in automated release recommendation logic.

## [Unreleased]

### Breaking

### Added

### Changed

### Removed

### Fixed

## [0.1.0]

### Added

- Interactive cluster selector TUI with fuzzy search powered by Bubble Tea
- Toggle between readonly and read-write mode with `w` key
- Automatic `kubeconfig` setup and `k9s` launch via `aws-vault`
- `scan` command to discover EKS clusters across all AWS profiles and regions
- Concurrent cluster scanning with semaphore-limited parallelism (max 10)
- Progress bar TUI during scanning
- Account, region, and cluster regex filters for scan
- `--append` flag to merge scan results with existing config
- Cluster deduplication by (account, region, cluster) identity
- YAML config format with `clusters.yaml` as default path
- CLI help via `-h` / `--help`
- Unit tests covering config parsing, CLI flag parsing, AWS profile parsing,
  cluster deduplication, fuzzy search filtering, and TUI selector behaviour
- GitHub Actions workflow to run tests on pushes to main and pull requests
- GitHub Actions workflow to build and release binaries (linux/darwin,
  amd64/arm64) on version tags
- Test status badge in README

[Unreleased]: https://github.com/DelphicOkami/ek9s/compare/v0.1.0...main
[0.1.0]: https://github.com/DelphicOkami/ek9s/compare/8a37e7b983d06ea64d1cea2ce7abee0c4a3afb8f...v0.1.0