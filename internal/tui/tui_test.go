package tui

import (
	"testing"

	"github.com/user/modelhub/internal/model"
)

// ── Pure formatting functions ──

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "-"},
		{0.0001, "$0.0001"},
		{0.001, "$0.0010"},
		{0.009, "$0.0090"},
		{0.01, "$0.010"},
		{0.05, "$0.050"},
		{0.10, "$0.100"},
		{0.50, "$0.500"},
		{0.99, "$0.990"},
		{1.0, "$1.00"},
		{1.50, "$1.50"},
		{10.0, "$10.00"},
		{100.50, "$100.50"},
		{0.0055, "$0.0055"},
	}
	for _, tt := range tests {
		got := formatPrice(tt.input)
		if got != tt.want {
			t.Errorf("formatPrice(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "-"},
		{1, "1"},
		{100, "100"},
		{999, "999"},
		{1000, "1K"},
		{1500, "1K"},
		{128000, "128K"},
		{999999, "999K"},
		{1000000, "1M"},
		{2000000, "2M"},
	}
	for _, tt := range tests {
		got := formatInt(tt.input)
		if got != tt.want {
			t.Errorf("formatInt(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		min, val, max, want int
	}{
		{10, 5, 100, 10},    // below min
		{10, 50, 100, 50},   // within
		{10, 200, 100, 100}, // above max
		{0, 0, 100, 0},      // at min
		{0, 100, 100, 100},  // at max
		{5, 3, 10, 5},       // just below
		{5, 12, 10, 10},     // just above
	}
	for _, tt := range tests {
		got := clamp(tt.min, tt.val, tt.max)
		if got != tt.want {
			t.Errorf("clamp(%d, %d, %d) = %d, want %d", tt.min, tt.val, tt.max, got, tt.want)
		}
	}
}

// ── modelToRow ──

func TestModelToRow(t *testing.T) {
	m := &teaModel{}

	mdl := model.Model{
		Provider:              "OpenAI",
		Name:                  "GPT-4o",
		Mode:                  "chat",
		InputPricePer1M:       2.50,
		OutputPricePer1M:      10.00,
		ContextWindow:         128000,
		MedianTokensPerSecond: 45.0,
		IntelligenceIndex:     85.3,
		Sources:               []string{"bifrost", "modelsdev"},
	}

	row := m.modelToRow(mdl)

	if len(row) != 9 {
		t.Fatalf("expected 9 columns, got %d", len(row))
	}
	if row[0] != "OpenAI" {
		t.Errorf("col 0 = %q, want OpenAI", row[0])
	}
	if row[1] != "GPT-4o" {
		t.Errorf("col 1 = %q, want GPT-4o", row[1])
	}
	if row[2] != "chat" {
		t.Errorf("col 2 = %q, want chat", row[2])
	}
	if row[3] != "$2.50" {
		t.Errorf("col 3 = %q, want $2.50", row[3])
	}
	if row[4] != "$10.00" {
		t.Errorf("col 4 = %q, want $10.00", row[4])
	}
	if row[5] != "128K" {
		t.Errorf("col 5 = %q, want 128K", row[5])
	}
	if row[6] != "45" {
		t.Errorf("col 6 = %q, want 45", row[6])
	}
	if row[7] != "85.3" {
		t.Errorf("col 7 = %q, want 85.3", row[7])
	}
	if row[8] != "2" {
		t.Errorf("col 8 = %q, want 2", row[8])
	}
}

func TestModelToRow_ZeroValues(t *testing.T) {
	m := &teaModel{}

	mdl := model.Model{
		Provider: "Test",
		Name:     "Empty",
		Sources:  []string{"bifrost"},
	}

	row := m.modelToRow(mdl)

	if row[3] != "-" { // price
		t.Errorf("zero price = %q, want -", row[3])
	}
	if row[5] != "-" { // context
		t.Errorf("zero context = %q, want -", row[5])
	}
	if row[6] != "-" { // speed
		t.Errorf("zero speed = %q, want -", row[6])
	}
	if row[7] != "-" { // intelligence
		t.Errorf("zero intel = %q, want -", row[7])
	}
	if row[8] != "1" { // sources count
		t.Errorf("1 source = %q, want 1", row[8])
	}
}

// ── uniqueProviders ──

func TestUniqueProviders(t *testing.T) {
	m := &teaModel{
		allModels: []model.Model{
			{ID: "a/m1", Provider: "OpenAI"},
			{ID: "b/m2", Provider: "Anthropic"},
			{ID: "c/m3", Provider: "OpenAI"},
			{ID: "d/m4", Provider: "Google"},
			{ID: "e/m5", Provider: ""},
		},
	}

	providers := m.uniqueProviders()
	expected := []string{"Anthropic", "Google", "OpenAI"}

	if len(providers) != len(expected) {
		t.Fatalf("got %d providers: %v, want %v", len(providers), providers, expected)
	}
	for i, p := range providers {
		if p != expected[i] {
			t.Errorf("providers[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestUniqueProviders_Empty(t *testing.T) {
	m := &teaModel{}
	providers := m.uniqueProviders()
	if len(providers) != 0 {
		t.Errorf("expected empty, got %v", providers)
	}
}

// ── applyFilter ──

func TestApplyFilter(t *testing.T) {
	m := &teaModel{
		allModels: []model.Model{
			{ID: "a/1", Name: "A", Provider: "OpenAI", Mode: "chat", Sources: []string{"bifrost"}},
			{ID: "b/2", Name: "B", Provider: "Anthropic", Mode: "chat", Sources: []string{"modelsdev"}},
		},
		fetchedAt: map[string]string{},
		sortBy:    "name",
		sortAsc:   true,
	}

	m.applyFilter()

	if len(m.models) != 2 {
		t.Errorf("expected 2 models, got %d", len(m.models))
	}
}

func TestApplyFilter_WithSearch(t *testing.T) {
	m := &teaModel{
		allModels: []model.Model{
			{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", Sources: []string{"bifrost"}},
			{ID: "b/claude", Name: "Claude", Provider: "Anthropic", Sources: []string{"bifrost"}},
		},
		fetchedAt: map[string]string{},
		sortBy:    "name",
		sortAsc:   true,
	}
	m.filter.Search = "claude"
	m.applyFilter()

	if len(m.models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(m.models))
	}
	if m.models[0].Name != "Claude" {
		t.Errorf("Name = %q, want Claude", m.models[0].Name)
	}
}

func TestApplyFilter_WithProviderFilter(t *testing.T) {
	m := &teaModel{
		allModels: []model.Model{
			{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", Sources: []string{"bifrost"}},
			{ID: "b/claude", Name: "Claude", Provider: "Anthropic", Sources: []string{"bifrost"}},
		},
		fetchedAt: map[string]string{},
		sortBy:    "name",
		sortAsc:   true,
	}
	m.filter.Providers = []string{"Anthropic"}
	m.applyFilter()

	if len(m.models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(m.models))
	}
	if m.models[0].Provider != "Anthropic" {
		t.Errorf("Provider = %q, want Anthropic", m.models[0].Provider)
	}
}

func TestApplyFilter_WithEmptyModels(t *testing.T) {
	m := &teaModel{
		allModels: []model.Model{},
		fetchedAt: map[string]string{},
		sortBy:    "name",
		sortAsc:   true,
	}
	m.applyFilter()
	if len(m.models) != 0 {
		t.Errorf("expected 0 models, got %d", len(m.models))
	}
}

// ── addCap ──

func TestAddCap(t *testing.T) {
	caps := addCap(nil, "Vision", true)
	if len(caps) != 1 {
		t.Fatalf("expected 1 cap, got %d", len(caps))
	}
	if caps[0] != "v  Vision" {
		t.Errorf("enabled cap = %q, want 'v  Vision'", caps[0])
	}

	caps = addCap(caps, "Reasoning", false)
	if caps[1] != "x  Reasoning" {
		t.Errorf("disabled cap = %q, want 'x  Reasoning'", caps[1])
	}
}
