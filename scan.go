package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

type describeRegionsOutput struct {
	Regions []struct {
		RegionName string `json:"RegionName"`
	} `json:"Regions"`
}

// fetchEKSRegions uses the first available profile to query AWS for all
// regions where EKS is available.
func fetchEKSRegions(profile string) ([]string, error) {
	cmd := exec.Command("aws-vault", "exec", profile, "--",
		"aws", "ec2", "describe-regions",
		"--all-regions",
		"--filters", "Name=opt-in-status,Values=opt-in-not-required,opted-in",
		"--query", "Regions[].RegionName",
		"--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch regions: %w", err)
	}

	var regions []string
	if err := json.Unmarshal(out, &regions); err != nil {
		return nil, fmt.Errorf("failed to parse regions: %w", err)
	}

	return regions, nil
}

type scanDoneMsg struct {
	clusters []Cluster
}

type scanProgressMsg struct {
	completed int
	total     int
	found     []Cluster // newly found clusters in this tick
}

type scanModel struct {
	progress  progress.Model
	total     int
	completed int
	found     []Cluster
	done      bool
	err       error
}

func (m scanModel) Init() tea.Cmd {
	return nil
}

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 8
		if m.progress.Width > 80 {
			m.progress.Width = 80
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case scanProgressMsg:
		m.completed = msg.completed
		m.total = msg.total
		m.found = append(m.found, msg.found...)
		return m, nil

	case scanDoneMsg:
		m.found = msg.clusters
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m scanModel) View() string {
	if m.done {
		return ""
	}

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	sb.WriteString(titleStyle.Render("Scanning for EKS clusters..."))
	sb.WriteString("\n\n")

	pct := 0.0
	if m.total > 0 {
		pct = float64(m.completed) / float64(m.total)
	}
	sb.WriteString("    ")
	sb.WriteString(m.progress.ViewAs(pct))
	sb.WriteString(fmt.Sprintf("  %d/%d", m.completed, m.total))
	sb.WriteString("\n\n")

	foundStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	if len(m.found) > 0 {
		sb.WriteString(foundStyle.Render(fmt.Sprintf("    Found %d cluster(s):", len(m.found))))
		sb.WriteString("\n")
		// Show only the last 5 found clusters
		start := len(m.found) - 5
		if start < 0 {
			start = 0
		}
		for _, c := range m.found[start:] {
			sb.WriteString(fmt.Sprintf("      %-40s | %-12s | %s\n", c.Cluster, c.Region, c.Account))
		}
	} else {
		sb.WriteString("    No clusters found yet...\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

func runScan(opts scanOptions) {
	profiles, err := parseAWSProfiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing AWS config: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Fprintf(os.Stderr, "No profiles found in ~/.aws/config\n")
		os.Exit(1)
	}

	// Filter profiles by account regex
	if opts.accountFilter != nil {
		var filtered []string
		for _, p := range profiles {
			if opts.accountFilter.MatchString(p) {
				filtered = append(filtered, p)
			}
		}
		profiles = filtered
		if len(profiles) == 0 {
			fmt.Fprintf(os.Stderr, "No profiles matched account filter\n")
			os.Exit(1)
		}
	}

	// Fetch available regions from AWS using the first profile
	fmt.Printf("Fetching available regions...\n")
	allRegions, err := fetchEKSRegions(profiles[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching regions: %v\n", err)
		os.Exit(1)
	}

	// Filter regions by region regex
	regions := allRegions
	if opts.regionFilter != nil {
		var filtered []string
		for _, r := range regions {
			if opts.regionFilter.MatchString(r) {
				filtered = append(filtered, r)
			}
		}
		regions = filtered
		if len(regions) == 0 {
			fmt.Fprintf(os.Stderr, "No regions matched region filter\n")
			os.Exit(1)
		}
	}

	total := len(profiles) * len(regions)

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(60),
	)

	m := scanModel{
		progress: prog,
		total:    total,
	}

	p := tea.NewProgram(m)

	// Run scan in background goroutine, sending progress to the TUI
	go func() {
		var (
			mu        sync.Mutex
			clusters  []Cluster
			wg        sync.WaitGroup
			completed int64
			sem       = make(chan struct{}, 10)
		)

		for _, profile := range profiles {
			for _, region := range regions {
				wg.Add(1)
				go func(profile, region string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					names, err := listEKSClusters(profile, region)

					done := int(atomic.AddInt64(&completed, 1))

					var found []Cluster
					if err == nil && len(names) > 0 {
						mu.Lock()
						for _, name := range names {
							// Apply cluster filter
							if opts.clusterFilter != nil && !opts.clusterFilter.MatchString(name) {
								continue
							}
							c := Cluster{
								Account: profile,
								Region:  region,
								Cluster: name,
							}
							clusters = append(clusters, c)
							found = append(found, c)
						}
						mu.Unlock()
					}

					p.Send(scanProgressMsg{
						completed: done,
						total:     total,
						found:     found,
					})
				}(profile, region)
			}
		}

		wg.Wait()
		p.Send(scanDoneMsg{clusters: clusters})
	}()

	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	final := result.(scanModel)
	scanned := final.found

	// In append mode, merge with existing config and deduplicate
	var clusters []Cluster
	if opts.append {
		existing, err := loadExistingConfig(opts.outputPath)
		if err == nil {
			clusters = append(clusters, existing...)
		}
	}
	clusters = append(clusters, scanned...)
	clusters = deduplicateClusters(clusters)

	if len(clusters) == 0 {
		fmt.Println("No EKS clusters found.")
		return
	}

	config := Config{Clusters: clusters}
	data, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling config: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(opts.outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", opts.outputPath, err)
		os.Exit(1)
	}

	mode := "Wrote"
	if opts.append {
		mode = "Merged"
	}
	fmt.Printf("%s %d clusters to %s\n", mode, len(clusters), opts.outputPath)
}

// parseAWSProfiles reads ~/.aws/config and returns profile names.
func parseAWSProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(filepath.Join(home, ".aws", "config"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseAWSProfilesFromReader(f)
}

// parseAWSProfilesFromReader extracts profile names from an AWS config file.
func parseAWSProfilesFromReader(r io.Reader) ([]string, error) {
	var profiles []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
			name := strings.TrimPrefix(line, "[profile ")
			name = strings.TrimSuffix(name, "]")
			profiles = append(profiles, name)
		} else if line == "[default]" {
			profiles = append(profiles, "default")
		}
	}

	return profiles, scanner.Err()
}

type eksListOutput struct {
	Clusters []string `json:"clusters"`
}

func loadExistingConfig(path string) ([]Cluster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config.Clusters, nil
}

func deduplicateClusters(clusters []Cluster) []Cluster {
	seen := make(map[string]struct{})
	var result []Cluster
	for _, c := range clusters {
		key := c.Account + "|" + c.Region + "|" + c.Cluster
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, c)
		}
	}
	return result
}

func listEKSClusters(profile, region string) ([]string, error) {
	cmd := exec.Command("aws-vault", "exec", profile, "--region", region, "--",
		"aws", "eks", "list-clusters", "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result eksListOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	return result.Clusters, nil
}
