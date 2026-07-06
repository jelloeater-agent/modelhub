package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/user/modelhub/internal/merge"
	"github.com/user/modelhub/internal/model"
)

func (m *teaModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (work in any state)
	switch msg.String() {
	case "ctrl+c", "q":
		if m.state != viewDetail { // detail uses 'q' to go back
			return m, tea.Quit
		}
		return m, nil
	case "?":
		if m.state == viewHelp {
			m.state = viewList
		} else {
			m.state = viewHelp
		}
		return m, nil
	}

	switch m.state {
	case viewList:
		if m.filtering {
			// In filter mode: only enter/esc handled by app
			switch msg.String() {
			case "enter":
				m.filter.Search = m.filterInput.Value()
				m.filtering = false
				m.applyFilter()
				m.initTable()
				return m, nil
			case "esc":
				m.filtering = false
				m.filterInput.SetValue("")
				return m, nil
			}
			// Forward all other keys to the filter input
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			return m, cmd
		}

		// App-level list keys
		if handled, cmd := m.handleListKey(msg); handled {
			return m, cmd
		}

		// Unhandled — forward to table for native navigation (arrows, pgup/dn, home/end)
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

	case viewDetail:
		m.handleDetailKey(msg)
		return m, nil
	}

	return m, nil
}

func (m *teaModel) handleListKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	handled = true
	switch msg.String() {
	case "/":
		m.filtering = true
		m.filterInput.SetValue("")
		m.filterInput.Focus()

	case "s":
		sortFields := []string{"name", "provider", "input_price", "output_price", "context", "speed", "intelligence", "sources"}
		for i, f := range sortFields {
			if m.sortBy == f {
				m.sortBy = sortFields[(i+1)%len(sortFields)]
				break
			}
		}
		m.applyFilter()
		m.initTable()

	case "S":
		m.sortAsc = !m.sortAsc
		m.applyFilter()
		m.initTable()

	case "m":
		modes := []string{"", "chat", "image_generation", "embedding"}
		current := ""
		if len(m.filter.Modes) > 0 {
			current = m.filter.Modes[0]
		}
		for i, mode := range modes {
			if current == mode {
				next := modes[(i+1)%len(modes)]
				if next == "" {
					m.filter.Modes = nil
				} else {
					m.filter.Modes = []string{next}
				}
				break
			}
		}
		m.applyFilter()
		m.initTable()

	case "p":
		providers := m.uniqueProviders()
		providers = append([]string{""}, providers...)
		current := ""
		if len(m.filter.Providers) > 0 {
			current = m.filter.Providers[0]
		}
		for i, p := range providers {
			if current == p {
				next := providers[(i+1)%len(providers)]
				if next == "" {
					m.filter.Providers = nil
				} else {
					m.filter.Providers = []string{next}
				}
				break
			}
		}
		m.applyFilter()
		m.initTable()

	case "v":
		m.filter.CapFlags.Vision = !m.filter.CapFlags.Vision
		m.applyFilter()
		m.initTable()

	case "r":
		m.state = viewList
		cmd = refreshModels(m.cfg)

	case "enter":
		if len(m.models) > 0 {
			row := m.table.Cursor()
			if row >= 0 && row < len(m.models) {
				m.selectedModel = &m.models[row]
				m.state = viewDetail
			}
		}

	case "esc":
		m.filter = merge.FilterParams{}
		m.filterInput.SetValue("")
		m.models = m.allModels
		m.applyFilter()
		m.initTable()

	default:
		handled = false
	}

	return
}

func (m *teaModel) handleDetailKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "esc", "q", "left":
		m.state = viewList
		m.selectedModel = nil
		return true
	}
	return false
}

func (m *teaModel) uniqueProviders() []string {
	seen := make(map[string]bool)
	for _, mdl := range m.allModels {
		if mdl.Provider != "" && !seen[mdl.Provider] {
			seen[mdl.Provider] = true
		}
	}
	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	sort.Strings(result)
	return result
}

func (m *teaModel) applyFilter() {
	m.models = merge.ApplyFilter(m.allModels, m.filter, m.sortBy, m.sortAsc)
	m.stats = model.ComputeStats(m.models, m.fetchedAt)
}

func (m *teaModel) handleRefresh(msg refreshMsg) tea.Cmd {
	if msg.result == nil {
		return nil
	}
	r := msg.result
	if r.Models != nil {
		m.allModels = r.Models
		m.fetchedAt = r.FetchedAt
		m.errors = r.Errors
		m.applyFilter()
		m.initTable()

		// Cache to disk
		c := &model.Cache{
			Models:    r.Models,
			FetchedAt: r.FetchedAt,
			Version:   int(time.Now().Unix()),
		}
		if m.store != nil {
			if err := m.store.Save(c); err != nil {
				m.errors["cache"] = err.Error()
			}
		}
	}
	return nil
}

func (m *teaModel) helpView() string {
	var b strings.Builder
	b.WriteString(DetailStyle.Width(m.width - 4).Render(
		SectionTitle.Render("Keyboard Shortcuts") + "\n\n" +
			"  ↑/↓       Scroll through models\n" +
			"  PgUp/PgDn  Page up/down\n" +
			"  /          Focus search filter\n" +
			"  s          Cycle sort column\n" +
			"  S          Toggle sort direction\n" +
			"  m          Filter by mode (chat/img/embed)\n" +
			"  p          Filter by provider\n" +
			"  v          Toggle vision capability filter\n" +
			"  r          Force refresh all data\n" +
			"  Enter      Open model detail view\n" +
			"  Esc        Clear all filters\n" +
			"  ?          Toggle this help\n" +
			"  q/Ctrl+C   Quit\n\n" +
			SectionTitle.Render("Detail View") + "\n\n" +
			"  Esc/q      Back to list\n",
	))
	b.WriteString("\n" + HelpStyle.Render("Press ? to close help"))
	return AppStyle.Render(b.String())
}

func (m *teaModel) errorView() string {
	return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
}

// ponytail: Keyboard shortcuts are hardcoded. Upgrade path: add a config map for custom keybindings.
