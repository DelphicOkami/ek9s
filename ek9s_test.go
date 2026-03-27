package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// --- Config YAML ---

func TestConfigRoundTrips(t *testing.T) {
	original := Config{
		Clusters: []Cluster{
			{Account: "acme-dev.Admin", Region: "eu-west-1", Cluster: "platform-dev-1"},
			{Account: "acme-prod.Admin", Region: "us-east-1", Cluster: "api-prod-2"},
		},
	}

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.Clusters) != len(original.Clusters) {
		t.Fatalf("got %d clusters, want %d", len(loaded.Clusters), len(original.Clusters))
	}
	for i, c := range loaded.Clusters {
		want := original.Clusters[i]
		if c != want {
			t.Errorf("cluster[%d] = %+v, want %+v", i, c, want)
		}
	}
}

func TestConfigYAMLMatchesExpectedFormat(t *testing.T) {
	// The config file format uses a top-level "clusters" key with account/region/cluster fields
	raw := `clusters:
  - account: myprofile
    region: us-east-1
    cluster: my-cluster
`
	var config Config
	if err := yaml.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(config.Clusters) != 1 {
		t.Fatalf("got %d clusters, want 1", len(config.Clusters))
	}
	c := config.Clusters[0]
	if c.Account != "myprofile" || c.Region != "us-east-1" || c.Cluster != "my-cluster" {
		t.Errorf("got %+v", c)
	}
}

func TestEmptyConfigParsesToZeroClusters(t *testing.T) {
	var config Config
	if err := yaml.Unmarshal([]byte("clusters: []\n"), &config); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(config.Clusters) != 0 {
		t.Errorf("got %d clusters, want 0", len(config.Clusters))
	}
}

// --- CLI flag parsing ---

func TestParseScanFlagsDefaults(t *testing.T) {
	opts := parseScanFlags([]string{})
	if opts.outputPath != "clusters.yaml" {
		t.Errorf("output = %q, want %q", opts.outputPath, "clusters.yaml")
	}
	if opts.accountFilter != nil {
		t.Error("account filter should be nil by default")
	}
	if opts.regionFilter != nil {
		t.Error("region filter should be nil by default")
	}
	if opts.clusterFilter != nil {
		t.Error("cluster filter should be nil by default")
	}
	if opts.append {
		t.Error("append should be false by default")
	}
}

func TestParseScanFlagsOutput(t *testing.T) {
	for _, flag := range []string{"-o", "--output"} {
		opts := parseScanFlags([]string{flag, "custom.yaml"})
		if opts.outputPath != "custom.yaml" {
			t.Errorf("%s: output = %q, want %q", flag, opts.outputPath, "custom.yaml")
		}
	}
}

func TestParseScanFlagsFilters(t *testing.T) {
	opts := parseScanFlags([]string{
		"-a", "acme",
		"-r", "eu-",
		"-c", "prod",
	})

	if opts.accountFilter == nil || !opts.accountFilter.MatchString("acme-dev") {
		t.Error("account filter should match 'acme-dev'")
	}
	if opts.regionFilter == nil || !opts.regionFilter.MatchString("eu-west-1") {
		t.Error("region filter should match 'eu-west-1'")
	}
	if opts.clusterFilter == nil || !opts.clusterFilter.MatchString("api-prod-1") {
		t.Error("cluster filter should match 'api-prod-1'")
	}
}

func TestParseScanFlagsFiltersArePartialMatch(t *testing.T) {
	// Filters use partial matching (regex), not exact matching
	opts := parseScanFlags([]string{"-a", "(api|web)"})
	for _, should := range []string{"api-dev-1", "data-web-prod-1"} {
		if !opts.accountFilter.MatchString(should) {
			t.Errorf("filter should match %q", should)
		}
	}
	if opts.accountFilter.MatchString("data-only") {
		t.Error("filter should not match 'data-only'")
	}
}

func TestParseScanFlagsLongForms(t *testing.T) {
	opts := parseScanFlags([]string{
		"--account", "dev",
		"--region", "west",
		"--cluster", "api",
		"--output", "out.yaml",
		"--append",
	})
	if opts.accountFilter == nil || !opts.accountFilter.MatchString("dev") {
		t.Error("--account not parsed")
	}
	if opts.regionFilter == nil || !opts.regionFilter.MatchString("west") {
		t.Error("--region not parsed")
	}
	if opts.clusterFilter == nil || !opts.clusterFilter.MatchString("api") {
		t.Error("--cluster not parsed")
	}
	if opts.outputPath != "out.yaml" {
		t.Errorf("output = %q", opts.outputPath)
	}
	if !opts.append {
		t.Error("append should be true")
	}
}

func TestParseScanFlagsAppend(t *testing.T) {
	opts := parseScanFlags([]string{"--append"})
	if !opts.append {
		t.Error("append should be true")
	}
}

// --- AWS profile parsing ---

func TestParseAWSProfilesExtractsNamedProfiles(t *testing.T) {
	input := `[profile acme-dev]
region = eu-west-1

[profile acme-prod]
region = us-east-1
`
	profiles, err := parseAWSProfilesFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}
	if profiles[0] != "acme-dev" || profiles[1] != "acme-prod" {
		t.Errorf("got %v", profiles)
	}
}

func TestParseAWSProfilesIncludesDefault(t *testing.T) {
	input := `[default]
region = us-east-1

[profile other]
region = eu-west-1
`
	profiles, err := parseAWSProfilesFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}
	if profiles[0] != "default" {
		t.Errorf("first profile = %q, want 'default'", profiles[0])
	}
}

func TestParseAWSProfilesIgnoresNonProfileSections(t *testing.T) {
	input := `[sso-session my-sso]
sso_start_url = https://example.com

[profile real-one]
region = us-east-1

; a comment
some_key = some_value
`
	profiles, err := parseAWSProfilesFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 1 || profiles[0] != "real-one" {
		t.Errorf("got %v, want [real-one]", profiles)
	}
}

func TestParseAWSProfilesEmptyConfig(t *testing.T) {
	profiles, err := parseAWSProfilesFromReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("got %d profiles, want 0", len(profiles))
	}
}

// --- Cluster deduplication ---

func TestDeduplicateRemovesDuplicates(t *testing.T) {
	clusters := []Cluster{
		{Account: "a", Region: "r", Cluster: "c"},
		{Account: "a", Region: "r", Cluster: "c"},
		{Account: "a", Region: "r2", Cluster: "c"},
	}
	result := deduplicateClusters(clusters)
	if len(result) != 2 {
		t.Fatalf("got %d clusters, want 2", len(result))
	}
}

func TestDeduplicateIdentityIsAccountRegionCluster(t *testing.T) {
	// Same cluster name in different accounts or regions are distinct
	clusters := []Cluster{
		{Account: "dev", Region: "eu-west-1", Cluster: "api"},
		{Account: "prod", Region: "eu-west-1", Cluster: "api"},
		{Account: "dev", Region: "us-east-1", Cluster: "api"},
	}
	result := deduplicateClusters(clusters)
	if len(result) != 3 {
		t.Errorf("got %d clusters, want 3 — all three are distinct", len(result))
	}
}

func TestDeduplicatePreservesOrder(t *testing.T) {
	clusters := []Cluster{
		{Account: "b", Region: "r", Cluster: "c"},
		{Account: "a", Region: "r", Cluster: "c"},
		{Account: "b", Region: "r", Cluster: "c"},
	}
	result := deduplicateClusters(clusters)
	if len(result) != 2 {
		t.Fatalf("got %d, want 2", len(result))
	}
	// First occurrence wins
	if result[0].Account != "b" || result[1].Account != "a" {
		t.Errorf("order not preserved: %v", result)
	}
}

func TestDeduplicateEmpty(t *testing.T) {
	result := deduplicateClusters(nil)
	if len(result) != 0 {
		t.Errorf("got %d, want 0", len(result))
	}
}

// --- Config file loading ---

func TestLoadExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clusters.yaml")

	config := Config{
		Clusters: []Cluster{
			{Account: "dev", Region: "eu-west-1", Cluster: "app-1"},
			{Account: "prod", Region: "us-east-1", Cluster: "app-2"},
		},
	}
	data, _ := yaml.Marshal(&config)
	os.WriteFile(path, data, 0644)

	loaded, err := loadExistingConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("got %d clusters, want 2", len(loaded))
	}
	if loaded[0].Cluster != "app-1" || loaded[1].Cluster != "app-2" {
		t.Errorf("got %v", loaded)
	}
}

func TestLoadExistingConfigMissingFileReturnsError(t *testing.T) {
	_, err := loadExistingConfig("/nonexistent/path/clusters.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- Fuzzy search filter ---

func TestClusterItemFilterValueIncludesAllFields(t *testing.T) {
	item := clusterItem{cluster: Cluster{
		Account: "acme-dev.Admin",
		Region:  "eu-west-1",
		Cluster: "platform-dev-1",
	}}

	fv := item.FilterValue()

	// Filter value should contain all three fields so fuzzy search works on any of them
	if !strings.Contains(fv, "platform-dev-1") {
		t.Error("filter value should contain cluster name")
	}
	if !strings.Contains(fv, "eu-west-1") {
		t.Error("filter value should contain region")
	}
	if !strings.Contains(fv, "acme-dev.Admin") {
		t.Error("filter value should contain account")
	}
}

// --- Selector TUI behavior ---

func TestSelectorStartsInReadonlyMode(t *testing.T) {
	readonly := true
	m := selectModel{
		list:     list.New(nil, clusterDelegate{readonly: &readonly}, 80, 20),
		readonly: &readonly,
	}
	if !*m.readonly {
		t.Error("selector should start in readonly mode")
	}
}

func TestSelectorWKeyTogglesReadonly(t *testing.T) {
	readonly := true
	items := []list.Item{
		clusterItem{cluster: Cluster{Account: "a", Region: "r", Cluster: "c"}},
	}
	m := selectModel{
		list:     list.New(items, clusterDelegate{readonly: &readonly}, 80, 20),
		readonly: &readonly,
	}

	// Press 'w' once → read-write
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}
	updated, _ := m.Update(msg)
	m = updated.(selectModel)
	if *m.readonly {
		t.Error("after first 'w' press, should be read-write")
	}

	// Press 'w' again → readonly
	updated, _ = m.Update(msg)
	m = updated.(selectModel)
	if !*m.readonly {
		t.Error("after second 'w' press, should be readonly again")
	}
}

func TestSelectorEnterSelectsCluster(t *testing.T) {
	readonly := true
	cluster := Cluster{Account: "dev", Region: "eu-west-1", Cluster: "my-cluster"}
	items := []list.Item{clusterItem{cluster: cluster}}

	l := list.New(items, clusterDelegate{readonly: &readonly}, 80, 20)
	m := selectModel{
		list:     l,
		readonly: &readonly,
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := m.Update(msg)
	final := updated.(selectModel)

	if final.choice == nil {
		t.Fatal("enter should select the current cluster")
	}
	if *final.choice != cluster {
		t.Errorf("selected %+v, want %+v", *final.choice, cluster)
	}
}

func TestSelectorNoChoiceOnQuit(t *testing.T) {
	readonly := true
	items := []list.Item{
		clusterItem{cluster: Cluster{Account: "a", Region: "r", Cluster: "c"}},
	}
	m := selectModel{
		list:     list.New(items, clusterDelegate{readonly: &readonly}, 80, 20),
		readonly: &readonly,
	}

	// ctrl+c should not set a choice
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, _ := m.Update(msg)
	final := updated.(selectModel)

	if final.choice != nil {
		t.Error("ctrl+c should not select a cluster")
	}
}

// --- Delegate rendering ---

func TestDelegateShowsReadonlyMode(t *testing.T) {
	readonly := true
	d := clusterDelegate{readonly: &readonly}
	if d.Height() != 1 {
		t.Errorf("height = %d, want 1", d.Height())
	}
	if d.Spacing() != 0 {
		t.Errorf("spacing = %d, want 0", d.Spacing())
	}
}

// --- Scan model behavior ---

func TestScanModelTracksProgress(t *testing.T) {
	m := scanModel{total: 10}

	updated, _ := m.Update(scanProgressMsg{completed: 3, total: 10, found: []Cluster{
		{Account: "a", Region: "r", Cluster: "c1"},
	}})
	sm := updated.(scanModel)

	if sm.completed != 3 {
		t.Errorf("completed = %d, want 3", sm.completed)
	}
	if len(sm.found) != 1 {
		t.Errorf("found = %d, want 1", len(sm.found))
	}
	if sm.done {
		t.Error("should not be done yet")
	}
}

func TestScanModelDoneQuitsWithClusters(t *testing.T) {
	m := scanModel{total: 10}

	clusters := []Cluster{
		{Account: "a", Region: "r", Cluster: "c1"},
		{Account: "b", Region: "r", Cluster: "c2"},
	}
	updated, cmd := m.Update(scanDoneMsg{clusters: clusters})
	sm := updated.(scanModel)

	if !sm.done {
		t.Error("should be done after scanDoneMsg")
	}
	if len(sm.found) != 2 {
		t.Errorf("found = %d, want 2", len(sm.found))
	}
	// Should signal quit
	if cmd == nil {
		t.Error("should return a quit command")
	}
}

func TestScanModelProgressAccumulates(t *testing.T) {
	m := scanModel{total: 10}

	updated, _ := m.Update(scanProgressMsg{completed: 1, total: 10, found: []Cluster{
		{Account: "a", Region: "r", Cluster: "c1"},
	}})
	m = updated.(scanModel)

	updated, _ = m.Update(scanProgressMsg{completed: 2, total: 10, found: []Cluster{
		{Account: "b", Region: "r", Cluster: "c2"},
	}})
	m = updated.(scanModel)

	if len(m.found) != 2 {
		t.Errorf("accumulated found = %d, want 2", len(m.found))
	}
}
