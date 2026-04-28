# ek9s

[![Test](https://github.com/DelphicOkami/ek9s/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/DelphicOkami/ek9s/actions/workflows/test.yml)

**Easy k9s** - a terminal UI for quickly connecting to EKS clusters with [k9s](https://k9scli.io/).

ek9s gives you a fuzzy-searchable list of your EKS clusters and connects you in one step, handling `kubeconfig` setup and `k9s` launch via `aws-vault`.

## Prerequisites

- [aws-vault](https://github.com/99designs/aws-vault)
- [AWS CLI](https://aws.amazon.com/cli/)
- [k9s](https://k9scli.io/)
- Go 1.26.1+

## Install

### From GitHub Releases

Download the latest binary for your platform from the [Releases page](https://github.com/DelphicOkami/ek9s/releases).

```bash
mv ek9s-<platform>-<arch> /usr/local/bin/ek9s
chmod +x /usr/local/bin/ek9s
```

#### macOS: removing the quarantine flag

macOS blocks unsigned binaries downloaded from the internet. After downloading, run:

```bash
xattr -d com.apple.quarantine /usr/local/bin/ek9s
```

### From source

```bash
go build -o ek9s .
mv ek9s /usr/local/bin/
```

## Usage

### Select and connect to a cluster

```bash
ek9s                    # uses clusters.yaml from ek9s's config directory
ek9s /path/to/config.yaml
```

The default config directory follows the same pattern as k9s:

- `$XDG_CONFIG_HOME/ek9s/` if `XDG_CONFIG_HOME` is set
- `~/Library/Application Support/ek9s/` on macOS
- `~/.config/ek9s/` elsewhere

ek9s reads `clusters.yaml` and (optionally) `read_only.skin.yaml` / `read_write.skin.yaml` from this directory.

This opens an interactive selector powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea):

- Type `/` to filter clusters with fuzzy search
- **Enter** to connect in readonly mode
- **Ctrl+W** to connect in read-write mode

On selection, ek9s runs:

```
aws-vault exec <account> --region <region> -- aws eks update-kubeconfig --name <cluster>
aws-vault exec <account> --region <region> -- k9s [--readonly]
```

### Scan for clusters

Automatically discover EKS clusters across all AWS profiles and regions:

```bash
ek9s scan
```

This parses `~/.aws/config` for profiles, polls every EKS-supported region in parallel, and writes the results to `clusters.yaml`.

#### Scan flags

| Flag | Description |
|------|-------------|
| `-o, --output <file>` | Output file (default: `clusters.yaml`) |
| `-a, --account <regex>` | Filter AWS profiles by regex (non-matching profiles are skipped) |
| `-r, --region <regex>` | Filter regions by regex (non-matching regions are skipped) |
| `-c, --cluster <regex>` | Filter discovered cluster names by regex (non-matching are dropped) |

Filters use partial matching with Go regex syntax.

#### Scan examples

```bash
# Scan only platform and data accounts, us-east regions
ek9s scan -a "(platform|data)" -r "us-east"

# Scan everything but only keep clusters matching "prod"
ek9s scan -c "prod"

# Scan dev accounts, write to a specific file
ek9s scan -a "dev" -o dev-clusters.yaml
```

### Per-mode k9s skins

ek9s can apply a different k9s skin depending on the selected mode. Drop either or both of these files into ek9s's config directory (see [Usage](#usage) above for the path):

- `read_only.skin.yaml` — applied when launching in readonly mode
- `read_write.skin.yaml` — applied when launching in read-write mode

At launch, the matching file is copied into your k9s skins directory (`$XDG_CONFIG_HOME/k9s/skins/` if set; otherwise `~/Library/Application Support/k9s/skins/` on macOS, `~/.config/k9s/skins/` elsewhere) as `ek9s-readonly.yaml` or `ek9s-readwrite.yaml`, and ek9s sets `K9S_SKIN` so k9s picks it up. If the file isn't present, k9s uses its default skin. See the [k9s skins docs](https://k9scli.io/topics/skins/) for the file format.

## Config format

```yaml
clusters:
  - account: "acme-platform-test.AdministratorAccess"
    region: "us-east-2"
    cluster: "platform-test-1"
  - account: "acme-platform-prod.AdministratorAccess"
    region: "us-east-1"
    cluster: "platform-prod-1"
    friendly_name: "platform prod (us-east)"
```

| Field | Description |
|-------|-------------|
| `account` | AWS Vault profile name |
| `region` | AWS region |
| `cluster` | EKS cluster name |
| `friendly_name` | Optional. Shown in the selector instead of `cluster` — useful when the same cluster name exists across multiple accounts. |
| `read_only_skin` | Optional. Path to a k9s skin file used when launching this cluster in readonly mode. Absolute paths are used as-is; relative paths resolve against ek9s's config directory. Falls back to `read_only.skin.yaml`. |
| `read_write_skin` | Optional. Same as above for read-write mode. Falls back to `read_write.skin.yaml`. |
