package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const usage = `ek9s - quickly connect to EKS clusters with k9s

Usage:
  ek9s [config-file]       Launch selector and connect to a cluster
  ek9s scan [flags]        Scan all AWS profiles/regions for EKS clusters

Commands:
  (default)   Open a selector to pick a cluster and launch k9s.
              Press W to toggle readonly/read-write, Enter to connect.
  scan        Parse ~/.aws/config and poll every region in each profile
              for EKS clusters, writing the results to a config file.

Scan flags:
  -o, --output <file>       Output file (default: $XDG_CONFIG_HOME/ek9s/clusters.yaml,
                            or the platform equivalent)
  -a, --account <regex>     Filter profiles by regex (skips non-matching)
  -r, --region <regex>      Filter regions by regex (skips non-matching)
  -c, --cluster <regex>     Filter discovered clusters by regex (drops non-matching)
      --append               Append to existing config file instead of replacing

  Filters use partial matching, e.g. "(api|web)" matches "api-dev-1" and "data-web-prod-1"

Arguments:
  config-file   Path to clusters config. Defaults to clusters.yaml inside
                ek9s's config directory ($XDG_CONFIG_HOME/ek9s, or
                ~/Library/Application Support/ek9s on macOS, or
                ~/.config/ek9s otherwise).

Options:
  -h, --help    Show this help message

Skins:
  Place read_only.skin.yaml and/or read_write.skin.yaml in ek9s's config
  directory to apply a k9s skin matching the selected mode.

Prerequisites:
  aws-vault, aws cli, k9s`

type Config struct {
	Clusters []Cluster `yaml:"clusters"`
}

type Cluster struct {
	Account       string `yaml:"account"`
	Region        string `yaml:"region"`
	Cluster       string `yaml:"cluster"`
	FriendlyName  string `yaml:"friendly_name,omitempty"`
	ReadOnlySkin  string `yaml:"read_only_skin,omitempty"`
	ReadWriteSkin string `yaml:"read_write_skin,omitempty"`
}

// DisplayName returns FriendlyName if set, otherwise the EKS cluster name.
func (c Cluster) DisplayName() string {
	if c.FriendlyName != "" {
		return c.FriendlyName
	}
	return c.Cluster
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			fmt.Println(usage)
			os.Exit(0)
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "scan" {
		opts := parseScanFlags(os.Args[2:])
		runScan(opts)
		return
	}

	configPath := defaultConfigPath()
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	runSelect(configPath)
}

// ek9sConfigDir returns ek9s's config directory, mirroring k9s's resolution:
// $XDG_CONFIG_HOME/ek9s if set, otherwise ~/Library/Application Support/ek9s
// on macOS and ~/.config/ek9s elsewhere.
func ek9sConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "ek9s")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "ek9s")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "ek9s")
	}
	return filepath.Join(home, ".config", "ek9s")
}

func defaultConfigPath() string {
	return filepath.Join(ek9sConfigDir(), "clusters.yaml")
}

type scanOptions struct {
	outputPath    string
	accountFilter *regexp.Regexp
	regionFilter  *regexp.Regexp
	clusterFilter *regexp.Regexp
	append        bool
}

func parseScanFlags(args []string) scanOptions {
	opts := scanOptions{
		outputPath: defaultConfigPath(),
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 < len(args) {
				i++
				opts.outputPath = args[i]
			}
		case "-a", "--account":
			if i+1 < len(args) {
				i++
				r, err := regexp.Compile(args[i])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid account regex %q: %v\n", args[i], err)
					os.Exit(1)
				}
				opts.accountFilter = r
			}
		case "-r", "--region":
			if i+1 < len(args) {
				i++
				r, err := regexp.Compile(args[i])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid region regex %q: %v\n", args[i], err)
					os.Exit(1)
				}
				opts.regionFilter = r
			}
		case "-c", "--cluster":
			if i+1 < len(args) {
				i++
				r, err := regexp.Compile(args[i])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid cluster regex %q: %v\n", args[i], err)
					os.Exit(1)
				}
				opts.clusterFilter = r
			}
		case "--append":
			opts.append = true
		default:
			fmt.Fprintf(os.Stderr, "Unknown scan flag: %s\n", args[i])
			os.Exit(1)
		}
	}

	return opts
}
