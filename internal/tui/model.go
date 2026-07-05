package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/user/modelhub/internal/cache"
	"github.com/user/modelhub/internal/merge"
	"github.com/user/modelhub/internal/model"
)

type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewHelp
	viewErrorState
)

// TUI messages
type refreshMsg struct {
	result *merge.RefreshResult
}

type errorMsg struct {
	err error
}

type filterDoneMsg struct{}

// teaModel is the top-level Bubble Tea model.
type teaModel struct {
	state  viewState
	width  int
	height int

	// Data
	models    []model.Model
	allModels []model.Model
	stats     model.Stats
	fetchedAt map[string]string
	errors    map[string]string

	// Cache
	store *cache.Store

	// Config
	cfg model.Config

	// Table
	table   table.Model
	columns []table.Column
	sortBy  string
	sortAsc bool

	// Filter
	filterInput textinput.Model
	filtering   bool
	filter      merge.FilterParams

	// Detail
	selectedModel *model.Model

	// State
	ready bool
	err   error
}

func NewModel(cfg model.Config, store *cache.Store, cached *model.Cache) (*teaModel, error) {
	ti := textinput.New()
	ti.Placeholder = "Search models..."
	ti.CharLimit = 60
	ti.Width = 40

	m := &teaModel{
		state:   viewList,
		cfg:     cfg,
		store:   store,
		sortBy:  "name",
		sortAsc: true,

		filterInput: ti,

		fetchedAt: make(map[string]string),
		errors:    make(map[string]string),
	}

	if cached != nil {
		m.models = cached.Models
		m.allModels = cached.Models
		m.fetchedAt = cached.FetchedAt
		m.stats = model.ComputeStats(cached.Models, cached.FetchedAt)
	}

	return m, nil
}

func (m *teaModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		refreshModels(m.cfg),
	)
}

func (m *teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.initTable()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case refreshMsg:
		cmd := m.handleRefresh(msg)
		return m, cmd

	case errorMsg:
		m.err = msg.err
		m.state = viewErrorState
		return m, nil

	case filterDoneMsg:
		m.filtering = false
		m.applyFilter()
		m.initTable()
		return m, nil
	}

	if m.filtering {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}

	if m.state == viewList {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *teaModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	switch m.state {
	case viewDetail:
		return m.detailView()
	case viewHelp:
		return m.helpView()
	case viewErrorState:
		return m.errorView()
	default:
		return m.listView()
	}
}

func (m *teaModel) listView() string {
	var b strings.Builder

	headerText := fmt.Sprintf(" ModelHub  •  %d models  •  %d sources  •  %d providers",
		m.stats.Total, len(m.stats.BySource), len(m.stats.ByProvider))
	if t, ok := m.fetchedAt["_merged"]; ok && len(t) > 10 {
		headerText += fmt.Sprintf("  •  updated %s", t[:10])
	}
	b.WriteString(HeaderStyle.Width(m.width).Render(headerText))
	b.WriteString("\n")

	var srcIndicators []string
	for src, count := range m.stats.BySource {
		switch src {
		case "bifrost":
			srcIndicators = append(srcIndicators, fmt.Sprintf("Bifrost (%d)", count))
		case "modelsdev":
			srcIndicators = append(srcIndicators, fmt.Sprintf("models.dev (%d)", count))
		case "aa":
			srcIndicators = append(srcIndicators, fmt.Sprintf("AA (%d)", count))
		}
	}
	if len(srcIndicators) > 0 {
		b.WriteString(mutedStyle.Render(strings.Join(srcIndicators, "  ·  ")))
		b.WriteString("\n")
	}

	for src, err := range m.errors {
		b.WriteString(errorStyle.Render(fmt.Sprintf("! %s: %s", src, err)))
		b.WriteString("\n")
	}

	b.WriteString(m.filterView())
	b.WriteString("\n")

	tv := m.table.View()
	b.WriteString(tv)
	b.WriteString("\n")

	if m.filtering {
		b.WriteString("\n")
		b.WriteString(FilterStyle.Render(m.filterInput.View()))
	}

	status := "  arrows scroll  |  / search  |  s sort  |  m mode  |  p provider  |  enter detail  |  r refresh  |  ? help  |  q quit"
	b.WriteString(StatusBarStyle.Width(m.width).Render(status))

	return AppStyle.Render(b.String())
}

func (m *teaModel) filterView() string {
	var parts []string
	if m.filter.Search != "" {
		parts = append(parts, fmt.Sprintf("search:%s", m.filter.Search))
	}
	if len(m.filter.Providers) > 0 {
		parts = append(parts, fmt.Sprintf("provider:%s", strings.Join(m.filter.Providers, ",")))
	}
	if len(m.filter.Modes) > 0 {
		parts = append(parts, fmt.Sprintf("mode:%s", strings.Join(m.filter.Modes, ",")))
	}
	if len(m.filter.Sources) > 0 {
		parts = append(parts, fmt.Sprintf("source:%s", strings.Join(m.filter.Sources, ",")))
	}
	if m.filter.CapFlags.Vision {
		parts = append(parts, "vision")
	}
	if m.filter.CapFlags.Reasoning {
		parts = append(parts, "reasoning")
	}

	sortDir := "▲"
	if !m.sortAsc {
		sortDir = "▼"
	}

	label := fmt.Sprintf(" Sort: %s %s  |  %d models shown", m.sortBy, sortDir, len(m.models))
	if len(parts) > 0 {
		label = fmt.Sprintf(" Filters: %s  |  %s", strings.Join(parts, ", "), label)
	}
	return mutedStyle.Render(label)
}

func (m *teaModel) initTable() {
	m.columns = []table.Column{
		{Title: "Provider", Width: clamp(14, m.width/8, 20)},
		{Title: "Model", Width: clamp(24, m.width/4, 40)},
		{Title: "Mode", Width: 12},
		{Title: "Input $/1M", Width: 14},
		{Title: "Output $/1M", Width: 14},
		{Title: "Context", Width: 10},
		{Title: "Speed t/s", Width: 12},
		{Title: "Intel", Width: 10},
		{Title: "Src", Width: 6},
	}

	rows := make([]table.Row, len(m.models))
	for i, mdl := range m.models {
		rows[i] = m.modelToRow(mdl)
	}

	t := table.New(
		table.WithColumns(m.columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-8),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(border).
		BorderBottom(true).
		Bold(false).
		Foreground(primary)
	s.Selected = s.Selected.
		Foreground(highlight).
		Background(lipgloss.Color("#374151")).
		Bold(false)
	t.SetStyles(s)

	m.table = t
}

func (m *teaModel) modelToRow(mdl model.Model) table.Row {
	speed := "-"
	if mdl.MedianTokensPerSecond > 0 {
		speed = fmt.Sprintf("%.0f", mdl.MedianTokensPerSecond)
	}
	intel := "-"
	if mdl.IntelligenceIndex > 0 {
		intel = fmt.Sprintf("%.1f", mdl.IntelligenceIndex)
	}

	return table.Row{
		mdl.Provider,
		mdl.Name,
		mdl.Mode,
		formatPrice(mdl.InputPricePer1M),
		formatPrice(mdl.OutputPricePer1M),
		formatInt(mdl.ContextWindow),
		speed,
		intel,
		fmt.Sprintf("%d", len(mdl.Sources)),
	}
}

func formatPrice(p float64) string {
	if p == 0 {
		return "-"
	}
	if p < 0.01 {
		return fmt.Sprintf("$%.4f", p)
	}
	if p < 1 {
		return fmt.Sprintf("$%.3f", p)
	}
	return fmt.Sprintf("$%.2f", p)
}

func formatInt(n int) string {
	if n == 0 {
		return "-"
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%dM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%dK", n/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func clamp(min, val, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func refreshModels(cfg model.Config) tea.Cmd {
	return func() tea.Msg {
		result := merge.DoRefresh(cfg)
		return refreshMsg{result: result}
	}
}
