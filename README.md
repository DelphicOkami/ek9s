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

```bash
go build -o ek9s .
```

Move the binary somewhere on your `$PATH`:

```bash
mv ek9s /usr/local/bin/
```

## Usage

### Select and connect to a cluster

```bash
ek9s                    # uses clusters.yaml in current directory
ek9s /path/to/config.yaml
```

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

## Config format

```yaml
clusters:
  - account: "acme-platform-test.AdministratorAccess"
    region: "us-east-2"
    cluster: "platform-test-1"
  - account: "acme-platform-prod.AdministratorAccess"
    region: "us-east-1"
    cluster: "platform-prod-1"
```

| Field | Description |
|-------|-------------|
| `account` | AWS Vault profile name |
| `region` | AWS region |
| `cluster` | EKS cluster name |
