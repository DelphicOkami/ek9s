package main

import (
	"fmt"
	"os"
	"regexp"
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
  -o, --output <file>       Output file (default: clusters.yaml)
  -a, --account <regex>     Filter profiles by regex (skips non-matching)
  -r, --region <regex>      Filter regions by regex (skips non-matching)
  -c, --cluster <regex>     Filter discovered clusters by regex (drops non-matching)
      --append               Append to existing config file instead of replacing

  Filters use partial matching, e.g. "(api|web)" matches "api-dev-1" and "data-web-prod-1"

Arguments:
  config-file   Path to clusters config (default: clusters.yaml)

Options:
  -h, --help    Show this help message

Prerequisites:
  aws-vault, aws cli, k9s`

type Config struct {
	Clusters []Cluster `yaml:"clusters"`
}

type Cluster struct {
	Account string `yaml:"account"`
	Region  string `yaml:"region"`
	Cluster string `yaml:"cluster"`
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

	configPath := "clusters.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	runSelect(configPath)
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
		outputPath: "clusters.yaml",
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
