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

## [1.0.0] - 2026-04-28

### Breaking

- Default config path moved from `./clusters.yaml` to `<config-dir>/clusters.yaml`,
  where `<config-dir>` is `$XDG_CONFIG_HOME/ek9s`, `~/Library/Application Support/ek9s`
  on macOS, or `~/.config/ek9s` elsewhere. Users relying on the working-directory
  default need to move the file or pass it explicitly.
- Skin files (`read_only.skin.yaml` / `read_write.skin.yaml`) are now read from the
  config directory instead of the directory next to the `ek9s` binary.

### Added

- `friendly_name` field on cluster config; shown in the selector instead of the
  EKS cluster name when set, and included in the fuzzy-search filter
- Per-cluster `read_only_skin` / `read_write_skin` fields to override the default
  skin files; absolute paths are used as-is, relative paths resolve against
  ek9s's config directory

### Changed

- Default config and skin files now resolve from ek9s's config directory; the
  `scan` command writes to the same location and creates the directory if missing
- Skin files are no longer rewritten into the k9s skins directory when the
  destination already matches the source, eliminating a redundant write on every
  launch

### Removed

### Fixed

## [0.2.0]

### Added

- Support for changing theme based on read / read-write mode selection
- Documented work around for OSx's binary quarantine

### Changed

- Replace manual release with goreleaser

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

[Unreleased]: https://github.com/DelphicOkami/ek9s/compare/v0.2.0...main
[0.2.0]: https://github.com/DelphicOkami/ek9s/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/DelphicOkami/ek9s/compare/8a37e7b983d06ea64d1cea2ce7abee0c4a3afb8f...v0.1.0
