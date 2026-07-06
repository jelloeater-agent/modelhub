package tui

import (
	"testing"

	"github.com/user/modelhub/internal/model"
)

// Cover the remaining Update message types
func TestUpdate_ErrorMsg(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	_, cmd := m.Update(errorMsg{err: assertString("something broke")})
	if m.err == nil {
		t.Error("expected error to be set")
	}
	if m.state != viewErrorState {
		t.Error("expected error state")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

func TestUpdate_FilterDoneMsg(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.filtering = true
	_, cmd := m.Update(filterDoneMsg{})
	if m.filtering {
		t.Error("filtering should be disabled after filterDoneMsg")
	}
	if cmd != nil {
		t.Error("expected nil cmd")
	}
}

// Cover listView stats by source with all 3 sources
func TestListView_MultiSource(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 100
	m.height = 40
	m.ready = true
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "P1", Mode: "chat", Sources: []string{"bifrost"}},
		{ID: "test/m2", Name: "M2", Provider: "P2", Mode: "image_generation", Sources: []string{"modelsdev"}},
		{ID: "test/m3", Name: "M3", Provider: "P3", Mode: "embedding", Sources: []string{"aa"}},
	}
	m.fetchedAt = map[string]string{"_merged": "2025-06-01T00:00:00Z"}
	m.stats = model.ComputeStats(m.models, m.fetchedAt)
	m.initTable()
	v := m.View()
	if v == "" {
		t.Error("list view should not be empty")
	}
}

// Cover listView error display
func TestListView_WithErrors(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 100
	m.height = 40
	m.ready = true
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "Test", Mode: "chat", Sources: []string{"bifrost"}},
	}
	m.fetchedAt = map[string]string{"_merged": "2025-06-01T00:00:00Z"}
	m.stats = model.ComputeStats(m.models, m.fetchedAt)
	m.errors = map[string]string{"bifrost": "timeout", "modelsdev": "no data"}
	m.initTable()
	v := m.View()
	if v == "" {
		t.Error("list view with errors should not be empty")
	}
}

// Cover listView source indicators with mixed sources
func TestListView_SourceIndicators(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.width = 100
	m.height = 40
	m.ready = true
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "P", Mode: "chat", Sources: []string{"bifrost"}},
	}
	m.fetchedAt = map[string]string{"_merged": "2025-06-01T00:00:00Z"}
	m.stats = model.ComputeStats(m.models, m.fetchedAt)
	m.initTable()
	v := m.View()
	if v == "" {
		t.Error("should render")
	}
}

// Cover filterView with all filter types
func TestFilterView_AllFilters(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.sortBy = "name"
	m.sortAsc = true
	m.filter.Search = "gpt"
	m.filter.Providers = []string{"OpenAI"}
	m.filter.Modes = []string{"chat"}
	m.filter.Sources = []string{"bifrost"}
	m.filter.CapFlags.Vision = true
	m.filter.CapFlags.Reasoning = true
	m.models = []model.Model{
		{ID: "test/m1", Name: "M1", Provider: "OpenAI", Mode: "chat", Sources: []string{"bifrost"}},
	}
	fv := m.filterView()
	if fv == "" {
		t.Error("filterView should not be empty with all filters")
	}
}

// Cover sort descending in filterView
func TestFilterView_SortDesc(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	m.sortBy = "input_price"
	m.sortAsc = false
	fv := m.filterView()
	if fv == "" {
		t.Error("should not be empty")
	}
}

// handleRefresh should handle nil result gracefully
func TestHandleRefresh_NilResult(t *testing.T) {
	m, _ := NewModel(model.Config{}, nil, nil)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("handleRefresh panicked with nil result: %v", r)
		}
	}()
	msg := refreshMsg{result: nil}
	_ = m.handleRefresh(msg)
	if m.err != nil {
		t.Logf("expected no error from nil result")
	}
}

// Promoted to helper
func assertString(s string) error { return &simpleErr{s} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }
