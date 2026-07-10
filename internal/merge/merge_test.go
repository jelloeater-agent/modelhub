package merge

import (
	"testing"

	"github.com/jelloeater-agent/modelhub/internal/model"
)

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"openai/gpt-4o", "openai/gpt-4o"},
		{"openai/gpt-4o-2024-08-06", "openai/gpt-4o"},
		{"anthropic/claude-3-5-sonnet-20241022", "anthropic/claude-3-5-sonnet"},
		{"openai/gpt-5.1-codex", "openai/gpt-5.1-codex"},
		{"xai/grok-4", "xai/grok-4"},
		{"xai/grok-4-fast", "xai/grok-4-fast"},
		{"amazon.nova-canvas-v1:0", "amazon.nova-canvas-v1"},
		{"writer.palmyra-x4-v1:0", "writer.palmyra-x4-v1"},
	}
	for _, tt := range tests {
		got := NormalizeID(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMerge(t *testing.T) {
	bifrost := []model.Model{
		{
			ID:               "openai/gpt-4o",
			Name:             "gpt-4o",
			Provider:         "OpenAI",
			Mode:             "chat",
			InputPricePer1M:  2.50,
			OutputPricePer1M: 10.00,
			Sources:          []string{"bifrost"},
		},
	}
	modelsdev := []model.Model{
		{
			ID:                      "openai/gpt-4o",
			Name:                    "GPT-4o",
			Provider:                "OpenAI",
			InputPricePer1M:         2.50,
			OutputPricePer1M:        10.00,
			SupportsVision:          true,
			SupportsFunctionCalling: true,
			ContextWindow:           128000,
			Sources:                 []string{"modelsdev"},
		},
	}

	merged := Merge(bifrost, modelsdev)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged model, got %d", len(merged))
	}
	m := merged[0]

	// Bifrost pricing should be preserved (higher priority overrides)
	if m.InputPricePer1M != 2.50 {
		t.Errorf("input price = %f, want 2.50", m.InputPricePer1M)
	}
	// Lower priority fills in missing bools
	if !m.SupportsVision {
		t.Error("SupportsVision should be true after merge")
	}
	if !m.SupportsFunctionCalling {
		t.Error("SupportsFunctionCalling should be true after merge")
	}
	// Lower priority fills in missing context window
	if m.ContextWindow != 128000 {
		t.Errorf("ContextWindow = %d, want 128000", m.ContextWindow)
	}
	// Both sources tracked
	if len(m.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(m.Sources))
	}
}

func TestMergePriority(t *testing.T) {
	// Bifrost has correct pricing, modelsdev has different (wrong) pricing
	bifrost := []model.Model{
		{
			ID:              "openai/gpt-4o",
			Name:            "gpt-4o",
			Provider:        "OpenAI",
			InputPricePer1M: 2.50,
			Sources:         []string{"bifrost"},
		},
	}
	modelsdev := []model.Model{
		{
			ID:              "openai/gpt-4o",
			Name:            "GPT-4o",
			Provider:        "OpenAI",
			InputPricePer1M: 10.00, // Different from bifrost
			Sources:         []string{"modelsdev"},
		},
	}

	merged := Merge(bifrost, modelsdev)
	m := merged[0]
	// Bifrost pricing should win
	if m.InputPricePer1M != 2.50 {
		t.Errorf("expected bifrost price 2.50, got %f", m.InputPricePer1M)
	}
}

func TestApplyFilter(t *testing.T) {
	models := []model.Model{
		{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", Mode: "chat", InputPricePer1M: 10, ContextWindow: 8000},
		{ID: "b/gpt4mini", Name: "GPT-4 Mini", Provider: "OpenAI", Mode: "chat", InputPricePer1M: 1, ContextWindow: 128000},
		{ID: "c/claude-opus", Name: "Claude Opus", Provider: "Anthropic", Mode: "chat", InputPricePer1M: 15, ContextWindow: 200000, SupportsVision: true},
	}

	// Filter by provider
	filtered := ApplyFilter(models, FilterParams{Providers: []string{"OpenAI"}}, "name", true)
	if len(filtered) != 2 {
		t.Errorf("expected 2 OpenAI models, got %d", len(filtered))
	}

	// Filter by price
	filtered = ApplyFilter(models, FilterParams{MaxPrice: 5}, "name", true)
	if len(filtered) != 1 {
		t.Errorf("expected 1 model under $5, got %d", len(filtered))
	}

	// Sort by price ascending
	filtered = ApplyFilter(models, FilterParams{}, "input_price", true)
	if len(filtered) != 3 || filtered[0].Name != "GPT-4 Mini" {
		t.Errorf("expected GPT-4 Mini first (cheapest), got %s", filtered[0].Name)
	}

	// Capability filter
	filtered = ApplyFilter(models, FilterParams{CapFlags: CapabilityFlags{Vision: true}}, "name", true)
	if len(filtered) != 1 {
		t.Errorf("expected 1 vision model, got %d", len(filtered))
	}
}
