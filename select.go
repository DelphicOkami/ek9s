package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// clusterItem implements list.Item for display in the selector.
type clusterItem struct {
	cluster Cluster
}

func (c clusterItem) FilterValue() string {
	return c.cluster.Cluster + " " + c.cluster.Region + " " + c.cluster.Account
}

// clusterDelegate renders each item in the list.
type clusterDelegate struct {
	readonly *bool
}

func (d clusterDelegate) Height() int                            { return 1 }
func (d clusterDelegate) Spacing() int                           { return 0 }
func (d clusterDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d clusterDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(clusterItem)
	if !ok {
		return
	}

	mode := "read-write"
	if *d.readonly {
		mode = "readonly"
	}

	line := fmt.Sprintf("%-40s | %-12s | %s", ci.cluster.Cluster, ci.cluster.Region, mode)

	if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			Render("> "+line))
	} else {
		fmt.Fprint(w, "  "+line)
	}
}

// selectModel is the bubbletea model for the cluster selector.
// readonly is a *bool shared with the delegate so toggling is visible in renders.
type selectModel struct {
	list     list.Model
	choice   *Cluster
	readonly *bool
	quitting bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		// Don't intercept keys while filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("w"))):
			*m.readonly = !*m.readonly
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if item, ok := m.list.SelectedItem().(clusterItem); ok {
				m.choice = &item.cluster
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func runSelect(configPath string) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	if len(config.Clusters) == 0 {
		fmt.Fprintf(os.Stderr, "No clusters found in %s\n", configPath)
		os.Exit(1)
	}

	items := make([]list.Item, len(config.Clusters))
	for i, c := range config.Clusters {
		items[i] = clusterItem{cluster: c}
	}

	readonly := true
	l := list.New(items, clusterDelegate{readonly: &readonly}, 80, 20)
	l.Title = "Select Cluster"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		PaddingLeft(1)

	helpStyle := list.DefaultStyles().HelpStyle.PaddingLeft(1)
	l.Styles.HelpStyle = helpStyle

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "toggle readonly")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
		}
	}

	m := selectModel{list: l, readonly: &readonly}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	final := result.(selectModel)
	if final.choice == nil {
		os.Exit(0)
	}

	account := final.choice.Account
	region := final.choice.Region
	cluster := final.choice.Cluster

	modeStr := "readonly"
	if !*final.readonly {
		modeStr = "read-write"
	}
	fmt.Printf("Connecting to %s in %s (%s)...\n", cluster, region, modeStr)

	// Update kubeconfig
	kubeconfig := exec.Command("aws-vault", "exec", account,
		"--region", region, "--",
		"aws", "eks", "update-kubeconfig", "--name", cluster)
	kubeconfig.Stdout = os.Stdout
	kubeconfig.Stderr = os.Stderr
	if err := kubeconfig.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Launch k9s
	k9sArgs := []string{"exec", account, "--region", region, "--", "k9s"}
	if *final.readonly {
		k9sArgs = append(k9sArgs, "--readonly")
	}
	k9s := exec.Command("aws-vault", k9sArgs...)
	k9s.Stdin = os.Stdin
	k9s.Stdout = os.Stdout
	k9s.Stderr = os.Stderr
	if err := k9s.Run(); err != nil {
		// k9s returns non-zero on normal exit sometimes, don't treat as fatal
		if strings.Contains(err.Error(), "exit status") {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error running k9s: %v\n", err)
		os.Exit(1)
	}
}
