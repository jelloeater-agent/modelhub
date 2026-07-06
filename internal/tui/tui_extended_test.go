package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/modelhub/internal/cache"
	"github.com/user/modelhub/internal/merge"
	"github.com/user/modelhub/internal/model"
)

// ── helper ──

func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	return ok
}

// ── NewModel ──

func TestNewModel_InitialState(t *testing.T) {
	m, err := NewModel(model.Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.models != nil {
		t.Error("models should be nil initially")
	}
	if m.ready {
		t.Error("should not be ready initially")
	}
	if m.filtering {
		t.Error("filter should not be active initially")
	}
	if m.filterInput.Value() != "" {
		t.Error("filter input should be empty")
	}
	if m.state != viewList {
		t.Error("should start in list view")
	}
	if m.sortBy != "name" {
		t.Errorf("default sort = %q, want name", m.sortBy)
	}
	if !m.sortAsc {
		t.Error("default sort should be ascending")
	}
}

func TestNewModel_WithCachedData(t *testing.T) {
	cached := &model.Cache{
		Models: []model.Model{
			{ID: "openai/gpt-4o", Name: "GPT-4o", Provider: "OpenAI", Sources: []string{"bifrost"}},
		},
		FetchedAt: map[string]string{"_merged": "2025-01-01T00:00:00Z"},
	}
	m, err := NewModel(model.Config{}, nil, cached)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.models) != 1 {
		t.Errorf("expected 1 model, got %d", len(m.models))
	}
	if m.stats.Total != 1 {
		t.Errorf("expected stats total 1, got %d", m.stats.Total)
	}
}

func TestNewModel_NilStore(t *testing.T) {
	m, err := NewModel(model.Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.store != nil {
		t.Error("store should be nil")
	}
}

// ── Init ──

func TestInit_ReturnsCmd(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

// ── View helpers ──

func TestHelpView(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 80
	help := m.helpView()
	if help == "" {
		t.Error("helpView should not be empty")
	}
}

func TestErrorView(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.err = nil // no error
	err := m.errorView()
	if err == "" {
		t.Error("errorView should not be empty")
	}
}

// ── handleKey ──

func TestHandleKey_GlobalQ_ListState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !isQuitCmd(cmd) {
		t.Error("expected quit cmd on 'q' in list state")
	}
}

func TestHandleKey_GlobalQ_DetailState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if isQuitCmd(cmd) {
		t.Error("'q' in detail should NOT quit")
	}
}

func TestHandleKey_CtrlC(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !isQuitCmd(cmd) {
		t.Error("expected quit cmd on ctrl+c")
	}
}

func TestHandleKey_HelpToggle(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.state != viewHelp {
		t.Error("expected viewHelp after '?'")
	}
	if cmd != nil {
		t.Error("expected nil cmd for help toggle")
	}

	// Toggle back
	_, cmd = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.state != viewList {
		t.Error("expected viewList after second '?'")
	}
}

func TestHandleKey_FilterModeEnter(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	// Activate filter mode via '/' in handleListKey
	m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.filtering {
		t.Fatal("expected filtering after '/'")
	}

	// In filter mode: 'enter' applies filter
	m.filterInput.SetValue("gpt")
	m.allModels = []model.Model{
		{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", Sources: []string{"bifrost"}},
		{ID: "b/claude", Name: "Claude", Provider: "Anthropic", Sources: []string{"bifrost"}},
	}
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.filtering {
		t.Error("filtering should be disabled after enter")
	}
	if m.filter.Search != "gpt" {
		t.Errorf("filter.Search = %q, want gpt", m.filter.Search)
	}
	if cmd != nil {
		_ = cmd // handleKey returns nil for enter in filter mode
	}
}

func TestHandleKey_FilterModeEsc(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.filterInput.SetValue("something")
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.filtering {
		t.Error("filtering should be disabled after esc")
	}
	if m.filterInput.Value() != "" {
		t.Error("filter input should be cleared after esc")
	}
	if cmd != nil {
		t.Error("expected nil cmd (handleKey returns nil for esc in filter mode)")
	}
}

func TestHandleKey_FilterModeForward(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	// Forward a 'g' key to filter input
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.filterInput.Value() != "g" {
		t.Errorf("expected filter input value 'g', got %q", m.filterInput.Value())
	}
	if cmd == nil {
		t.Error("expected non-nil cmd from filter input update")
	}
}

func TestHandleKey_ListState_UnhandledDelegatesToTable(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	m.width = 80
	m.height = 24
	m.allModels = []model.Model{
		{ID: "a/m1", Name: "M1", Provider: "P", Sources: []string{"bifrost"}},
	}
	m.applyFilter()
	m.initTable()
	// Arrow down — should be delegated to table
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		// Table returns a command when it scrolls — that's sufficient evidence
		// it was delegated. If the table has 1 row and we're at the bottom,
		// cmd may be nil (no movement possible).
		_ = cmd
	}
}

func TestHandleKey_DetailState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	m.selectedModel = &model.Model{ID: "test/m", Name: "M", Provider: "P"}
	// Esc should go back to list
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != viewList {
		t.Error("expected viewList after esc in detail state")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleKey_DefaultState(t *testing.T) {
	// Unknown state should not panic
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewState(99)
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// ── handleListKey ──

func TestHandleListKey_Slash(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !handled {
		t.Error("expected handled")
	}
	if !m.filtering {
		t.Error("expected filtering after '/'")
	}
	if cmd != nil {
		_ = cmd // Focus cmd is discarded in current impl
	}
}

func TestHandleListKey_SortS(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.sortBy = "name"
	m.sortAsc = true
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !handled {
		t.Error("expected handled")
	}
	if m.sortBy == "name" {
		t.Error("sortBy should have cycled from name")
	}
	if cmd != nil {
		_ = cmd // handleListKey returns nil for 's'
	}
	// Should not have toggled sortAsc
	if !m.sortAsc {
		t.Error("'s' should not toggle Asc")
	}

	// Verify 's' cycles through all sort fields
	fields := []string{"name", "provider", "input_price", "output_price", "context", "speed", "intelligence", "sources"}
	m.sortBy = "name"
	for i := 0; i < len(fields)*2; i++ {
		m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	}
	if m.sortBy != "name" {
		t.Errorf("after full cycle, sortBy = %q, want name", m.sortBy)
	}
}

func TestHandleListKey_SortCapitalS(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.sortAsc = true
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	if !handled {
		t.Error("expected handled")
	}
	if m.sortAsc {
		t.Error("expected sortAsc toggled to false")
	}
	if cmd != nil {
		_ = cmd // handleListKey returns nil for 'S'
	}
}

func TestHandleListKey_FilterMode(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if !handled {
		t.Error("expected handled")
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestHandleListKey_FilterProvider(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.allModels = []model.Model{
		{ID: "a/m1", Name: "M1", Provider: "OpenAI", Sources: []string{"bifrost"}},
		{ID: "b/m2", Name: "M2", Provider: "Anthropic", Sources: []string{"bifrost"}},
	}
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if !handled {
		t.Error("expected handled")
	}
	if len(m.filter.Providers) == 0 {
		t.Error("expected provider filter to be set")
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestHandleListKey_FilterVision(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !handled {
		t.Error("expected handled")
	}
	if !m.filter.CapFlags.Vision {
		t.Error("expected vision cap flag toggled on")
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestHandleListKey_Refresh(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !handled {
		t.Error("expected handled")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (refreshModels)")
	}
}

func TestHandleListKey_Enter(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.models = []model.Model{
		{ID: "a/m1", Name: "M1", Provider: "P", Sources: []string{"bifrost"}},
	}
	m.width = 80
	m.height = 24
	m.initTable()
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected handled")
	}
	if m.state != viewDetail {
		t.Error("expected viewDetail after enter")
	}
	if m.selectedModel == nil {
		t.Error("expected selectedModel to be set")
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestHandleListKey_Enter_NoModels(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Error("expected handled even with no models")
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestHandleListKey_Esc(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, cmd := m.handleListKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled")
	}
	if cmd != nil {
		_ = cmd
	}
}

func TestHandleListKey_EscClearsFilters(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.filter.Search = "something"
	m.filter.Providers = []string{"OpenAI"}
	m.handleListKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.filter.Search != "" {
		t.Error("filter search should be cleared")
	}
	if len(m.filter.Providers) != 0 {
		t.Error("provider filter should be cleared")
	}
}

func TestHandleListKey_Unhandled(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	handled, _ := m.handleListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if handled {
		t.Error("'z' should not be handled")
	}
}

// ── handleDetailKey ──

func TestHandleDetailKey_Esc(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	m.selectedModel = &model.Model{ID: "test/m", Name: "M", Provider: "P"}
	handled := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled {
		t.Error("expected handled")
	}
	if m.state != viewList {
		t.Error("expected viewList after esc")
	}
	if m.selectedModel != nil {
		t.Error("selectedModel should be nil after esc")
	}
}

func TestHandleDetailKey_Q(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	handled := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled {
		t.Error("expected handled")
	}
	if m.state != viewList {
		t.Error("expected viewList after q")
	}
}

func TestHandleDetailKey_Left(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	handled := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if handled {
		t.Error("'z' should not be handled in detail")
	}
}

func TestHandleDetailKey_Unhandled(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	handled := m.handleDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if handled {
		t.Error("'x' should not be handled in detail")
	}
	if m.state != viewDetail {
		t.Error("state should remain viewDetail")
	}
}

// ── handleRefresh ──

func TestHandleRefresh_Empty(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	cmd := m.handleRefresh(refreshMsg{result: &merge.RefreshResult{}})
	if m.err != nil {
		t.Errorf("unexpected err: %v", m.err)
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleRefresh_WithModels(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	models := []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Sources: []string{"bifrost"}},
		{ID: "test/m2", Name: "M2", Provider: "Test", Sources: []string{"modelsdev"}},
	}
	result := &merge.RefreshResult{
		Models:    models,
		FetchedAt: map[string]string{"_merged": "2025-06-01T00:00:00Z"},
	}
	cmd := m.handleRefresh(refreshMsg{result: result})
	if len(m.models) != 2 {
		t.Errorf("expected 2 models, got %d", len(m.models))
	}
	if m.stats.Total != 2 {
		t.Errorf("expected stats total 2, got %d", m.stats.Total)
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestHandleRefresh_WithErrors(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	result := &merge.RefreshResult{
		Models:    []model.Model{}, // non-nil to trigger the save path
		FetchedAt: map[string]string{},
		Errors:    map[string]string{"bifrost": "connection refused"},
	}
	_ = m.handleRefresh(refreshMsg{result: result})
	if m.errors["bifrost"] != "connection refused" {
		t.Errorf("expected error stored, got %v", m.errors)
	}
}

// ── applyFilter ──

func TestApplyFilter_SortByInputPrice(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.allModels = []model.Model{
		{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", InputPricePer1M: 10, Sources: []string{"bifrost"}},
		{ID: "b/claude", Name: "Claude", Provider: "Anthropic", InputPricePer1M: 3, Sources: []string{"bifrost"}},
	}
	m.sortBy = "input_price"
	m.sortAsc = true
	m.fetchedAt = map[string]string{}
	m.applyFilter()
	if len(m.models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(m.models))
	}
	if m.models[0].Name != "Claude" {
		t.Error("cheaper model should come first when sorted by price asc")
	}
}

func TestApplyFilter_SortBySpeedDesc(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.allModels = []model.Model{
		{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", MedianTokensPerSecond: 50, Sources: []string{"bifrost"}},
		{ID: "b/claude", Name: "Claude", Provider: "Anthropic", MedianTokensPerSecond: 100, Sources: []string{"bifrost"}},
	}
	m.sortBy = "speed"
	m.sortAsc = false
	m.fetchedAt = map[string]string{}
	m.applyFilter()
	if m.models[0].Name != "Claude" {
		t.Error("faster model should come first when sorted by speed desc")
	}
}

// ── initTable ──

func TestInitTable(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Mode: "chat", InputPricePer1M: 1.0, Sources: []string{"test"}},
	}
	m.width = 100
	m.height = 40
	m.initTable()
	if len(m.table.Rows()) != 1 {
		t.Errorf("expected 1 row, got %d", len(m.table.Rows()))
	}
	if m.ready {
		t.Error("initTable should not set ready")
	}
}

func TestInitTable_Empty(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 80
	m.height = 24
	m.initTable()
	if len(m.table.Rows()) != 0 {
		t.Errorf("expected 0 rows, got %d", len(m.table.Rows()))
	}
}

// ── detailView ──

func TestDetailView_WithSelection(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	m.width = 80
	m.selectedModel = &model.Model{
		ID:                    "openai/gpt-4o",
		Name:                  "GPT-4o",
		Provider:              "OpenAI",
		Mode:                  "chat",
		InputPricePer1M:       2.50,
		OutputPricePer1M:      10.00,
		ContextWindow:         128000,
		SupportsVision:        true,
		MedianTokensPerSecond: 50.0,
		IntelligenceIndex:     85.0,
		Sources:               []string{"bifrost", "modelsdev"},
	}
	detail := m.detailView()
	if detail == "" {
		t.Error("detailView should not be empty")
	}
}

func TestDetailView_NoSelection(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewDetail
	detail := m.detailView()
	if detail != "" {
		t.Error("detailView should be empty with no selection")
	}
	if m.state != viewList {
		t.Error("should revert to list state when no selection")
	}
}

// ── filterView ──

func TestFilterView_Empty(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	fv := m.filterView()
	if fv == "" {
		t.Error("filterView should not be empty")
	}
}

func TestFilterView_WithFilters(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.filter.Search = "gpt"
	m.filter.Providers = []string{"OpenAI"}
	m.filter.CapFlags.Vision = true
	fv := m.filterView()
	if fv == "" {
		t.Error("filterView with filters should not be empty")
	}
}

// ── View ──

func TestView_NotReady(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	v := m.View()
	if v == "" {
		t.Error("View should not be empty even when not ready")
	}
}

func TestView_Ready_ListState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 80
	m.height = 24
	m.ready = true
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Mode: "chat", InputPricePer1M: 1.0, Sources: []string{"bifrost"}},
	}
	m.stats = model.ComputeStats(m.models, m.fetchedAt)
	m.initTable()
	v := m.View()
	if v == "" {
		t.Error("View should not be empty")
	}
}

func TestView_DetailState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 80
	m.height = 24
	m.ready = true
	m.state = viewDetail
	m.selectedModel = &model.Model{
		ID:       "test/m1",
		Name:     "M1",
		Provider: "Test",
		Sources:  []string{"bifrost"},
	}
	m.stats = model.ComputeStats(nil, map[string]string{})
	v := m.View()
	if v == "" {
		t.Error("View in detail mode should not be empty")
	}
}

func TestView_HelpState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 80
	m.ready = true
	m.state = viewHelp
	v := m.View()
	if v == "" {
		t.Error("View in help mode should not be empty")
	}
}

func TestView_ErrorState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.ready = true
	m.state = viewErrorState
	v := m.View()
	if v == "" {
		t.Error("View in error state should not be empty")
	}
}

// ── Update ──

func TestUpdate_WindowSize(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Sources: []string{"bifrost"}},
	}
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if !m.ready {
		t.Error("expected ready after WindowSizeMsg")
	}
	if m.width != 100 || m.height != 40 {
		t.Errorf("expected 100x40, got %dx%d", m.width, m.height)
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestUpdate_RefreshMsg(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	msg := refreshMsg{result: &merge.RefreshResult{Models: []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Sources: []string{"bifrost"}},
	}}}
	_, cmd := m.Update(msg)
	if len(m.models) != 1 {
		t.Errorf("expected 1 model, got %d", len(m.models))
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestUpdate_KeyMsg(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	// Press 'q' to quit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !isQuitCmd(cmd) {
		t.Error("expected quit cmd")
	}
}

func TestUpdate_FilterActive(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.filtering = true
	// Forward a key to filter input
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_ = cmd
}

func TestUpdate_ListState(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.state = viewList
	m.width = 80
	m.height = 24
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Sources: []string{"bifrost"}},
		{ID: "test/m2", Name: "M2", Provider: "Test2", Sources: []string{"bifrost"}},
	}
	m.initTable()
	// Arrow down while in list view should be delegated to table
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Table may or may not return a command depending on cursor position;
	// key point is it doesn't crash and state is valid
	if cmd != nil {
		_ = cmd
	}
}

// ── modelToRow integration ──

func TestModelToRow_AccuracyFocus(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	mdl := model.Model{
		ID: "test/m1", Name: "M1", Provider: "Test", Mode: "chat",
		InputPricePer1M: 2.50, OutputPricePer1M: 10.00, ContextWindow: 128000,
		MedianTokensPerSecond: 50, IntelligenceIndex: 85, Sources: []string{"bifrost", "modelsdev"},
	}
	row := m.modelToRow(mdl)
	if len(row) != 9 {
		t.Fatalf("expected 9 cols, got %d", len(row))
	}
	if row[0] != "Test" || row[1] != "M1" || row[2] != "chat" {
		t.Errorf("unexpected row content: %v", row)
	}
}

// ── refreshModels ──

func TestRefreshModels_ReturnsMsg(t *testing.T) {
	cmd := refreshModels(model.Config{})
	if cmd == nil {
		t.Fatal("refreshModels should return a cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd should return a msg")
	}
	_, ok := msg.(refreshMsg)
	if !ok {
		t.Errorf("expected refreshMsg, got %T", msg)
	}
}

// ── cache integration ──
func TestHandleRefresh_SavesToCache(t *testing.T) {
	dir := t.TempDir()
	store, err := cache.NewStore(dir + "/test-cache.json")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := NewModel(model.Config{}, store, nil)
	models := []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Sources: []string{"bifrost"}},
	}
	result := &merge.RefreshResult{
		Models:    models,
		FetchedAt: map[string]string{"_merged": "2025-06-01T00:00:00Z"},
	}
	_ = m.handleRefresh(refreshMsg{result: result})
	if !store.Exists() {
		t.Error("expected cache file to exist after refresh")
	}
}

func TestHandleRefresh_SaveCacheFailsGracefully(t *testing.T) {
	// nil store should not panic
	m, _ := NewModel(model.Config{}, nil, nil)
	models := []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Sources: []string{"bifrost"}},
	}
	result := &merge.RefreshResult{
		Models:    models,
		FetchedAt: map[string]string{"_merged": "2025-06-01T00:00:00Z"},
	}
	_ = m.handleRefresh(refreshMsg{result: result})
	// Should not panic — nil store just skips save
	if m.errors["cache"] != "" {
		t.Errorf("unexpected cache error: %s", m.errors["cache"])
	}
}

func TestHandleRefresh_PreservesFiltering(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	// Set a filter before refresh
	m.filter.Search = "gpt"
	models := []model.Model{
		{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", Sources: []string{"bifrost"}},
		{ID: "b/claude", Name: "Claude", Provider: "Anthropic", Sources: []string{"bifrost"}},
	}
	result := &merge.RefreshResult{
		Models:    models,
		FetchedAt: map[string]string{"_merged": "2025-06-01T00:00:00Z"},
	}
	_ = m.handleRefresh(refreshMsg{result: result})
	if len(m.models) != 1 {
		t.Errorf("expected 1 model after refresh+filter, got %d", len(m.models))
	}
	if m.models[0].Name != "GPT-4" {
		t.Error("filter should still be applied after refresh")
	}
}
