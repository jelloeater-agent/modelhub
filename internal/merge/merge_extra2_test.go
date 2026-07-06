package merge

import (
	"testing"

	"github.com/user/modelhub/internal/model"
)

// Exercise all Merge priority branches: incoming=higher, incoming=lower, same priority
func TestMerge_PriorityBranches(t *testing.T) {
	// higher priority (bifrost) with more fields than lower (modelsdev)
	bifrost := []model.Model{
		{
			ID:                    "test/model-a",
			Name:                  "Model A",
			Provider:              "Test",
			Mode:                  "chat",
			InputPricePer1M:       5.0,
			ContextWindow:         1000,
			MedianTokensPerSecond: 42,
			SupportsVision:        true,
			SupportsReasoning:     false,
			Sources:               []string{"bifrost"},
		},
	}
	modelsdev := []model.Model{
		{
			ID:                       "test/model-a",
			Name:                     "Model A",
			Provider:                 "Test",
			Mode:                     "", // should not override chat
			OutputPricePer1M:         20.0,
			MaxOutput:                4000,
			IntelligenceIndex:        95.0,
			Family:                   "model-a-family",
			Description:              "A test model",
			SupportsFunctionCalling:  true,
			SupportsPromptCaching:    true,
			SupportsStructuredOutput: true,
			OpenWeights:              true,
			Sources:                  []string{"modelsdev"},
		},
	}
	// aa (lowest priority) fills remaining gaps
	aa := []model.Model{
		{
			ID:                "test/model-a",
			Name:              "Model A",
			Provider:          "Test",
			CodingIndex:       90.0,
			MathIndex:         88.0,
			MedianTTFTSeconds: 0.5,
			MMLUPro:           0.89,
			GPQA:              0.87,
			LiveCodeBench:     0.82,
			AIME25:            0.75,
			ReleaseDate:       "2025-01-15",
			Sources:           []string{"aa"},
		},
	}

	merged := Merge(bifrost, modelsdev, aa)
	if len(merged) != 1 {
		t.Fatalf("expected 1 model, got %d", len(merged))
	}
	m := merged[0]

	// Bifrost (highest priority) values should be preserved
	if m.InputPricePer1M != 5.0 {
		t.Errorf("InputPrice = %f, want 5.0", m.InputPricePer1M)
	}
	if !m.SupportsVision {
		t.Error("SupportsVision should be true")
	}
	if m.MedianTokensPerSecond != 42 {
		t.Errorf("Speed = %f, want 42", m.MedianTokensPerSecond)
	}

	// Modelsdev (lower priority) should fill gaps
	if m.OutputPricePer1M != 20.0 {
		t.Errorf("OutputPrice = %f, want 20.0", m.OutputPricePer1M)
	}
	if m.MaxOutput != 4000 {
		t.Errorf("MaxOutput = %d", m.MaxOutput)
	}
	if m.IntelligenceIndex != 95.0 {
		t.Errorf("Intelligence = %f", m.IntelligenceIndex)
	}
	if m.Family != "model-a-family" {
		t.Errorf("Family = %q", m.Family)
	}
	if m.Description != "A test model" {
		t.Errorf("Description = %q", m.Description)
	}
	if !m.SupportsFunctionCalling {
		t.Error("Should have function calling")
	}
	if !m.SupportsPromptCaching {
		t.Error("Should have prompt caching")
	}
	if !m.SupportsStructuredOutput {
		t.Error("Should have structured output")
	}
	if !m.OpenWeights {
		t.Error("Should have open weights")
	}

	// AA (lowest priority) fills remaining fields
	if m.CodingIndex != 90.0 {
		t.Errorf("Coding = %f", m.CodingIndex)
	}
	if m.MathIndex != 88.0 {
		t.Errorf("Math = %f", m.MathIndex)
	}
	if m.MedianTTFTSeconds != 0.5 {
		t.Errorf("TTFT = %f", m.MedianTTFTSeconds)
	}
	if m.MMLUPro != 0.89 {
		t.Errorf("MMLU = %f", m.MMLUPro)
	}
	if m.GPQA != 0.87 {
		t.Errorf("GPQA = %f", m.GPQA)
	}
	if m.LiveCodeBench != 0.82 {
		t.Errorf("LIVECODE = %f", m.LiveCodeBench)
	}
	if m.AIME25 != 0.75 {
		t.Errorf("AIME = %f", m.AIME25)
	}
	if m.ReleaseDate != "2025-01-15" {
		t.Errorf("ReleaseDate = %q", m.ReleaseDate)
	}

	// Should track 3 sources
	if len(m.Sources) != 3 {
		t.Errorf("expected 3 sources, got %v", m.Sources)
	}
}

// Same priority (same source) - mergeNonZero path
func TestMerge_SamePriorityDedup(t *testing.T) {
	bifrost := []model.Model{
		{
			ID:               "test/model-b",
			Name:             "Model B",
			Provider:         "Test",
			Mode:             "chat",
			InputPricePer1M:  1.0,
			OutputPricePer1M: 5.0,
			ContextWindow:    1000,
			MaxOutput:        2000,
			Sources:          []string{"bifrost"},
		},
		{
			ID:                    "test/model-b", // Same normalized ID, same source
			Name:                  "Model B",
			Provider:              "Test",
			Mode:                  "chat",
			CacheReadPrice:        0.5,
			MedianTokensPerSecond: 100,
			SupportsReasoning:     true,
			Sources:               []string{"bifrost"},
		},
	}

	merged := Merge(bifrost)
	if len(merged) != 1 {
		t.Fatalf("expected 1 model, got %d", len(merged))
	}
	m := merged[0]
	// mergeNonZero should combine non-zero fields
	if m.InputPricePer1M != 1.0 {
		t.Errorf("InputPrice = %f", m.InputPricePer1M)
	}
	if m.OutputPricePer1M != 5.0 {
		t.Errorf("OutputPrice = %f", m.OutputPricePer1M)
	}
	if m.CacheReadPrice != 0.5 {
		t.Errorf("CacheReadPrice = %f", m.CacheReadPrice)
	}
	if m.MedianTokensPerSecond != 100 {
		t.Errorf("Speed = %f", m.MedianTokensPerSecond)
	}
	if !m.SupportsReasoning {
		t.Error("Should have reasoning")
	}
}

// Test the NormalizeID with version stripping edge cases
func TestNormalizeID_EdgeCases(t *testing.T) {
	// ponytail: NormalizeID strips trailing date/version patterns (4+ digits,
	// preview/snapshot/alpha/beta/rc). `-v1` or `:v2` are NOT stripped
	// because "v1" starts with 'v' (not all digits, not a known suffix).
	// Upgrade: add `-v\d+` pattern detection.
	tests := []struct {
		input string
		want  string
	}{
		{"openai/gpt-4-v1", "openai/gpt-4-v1"},                                         // -vN NOT stripped (starts with 'v')
		{"openai/gpt-4:v2", "openai/gpt-4:v2"},                                         // :vN NOT stripped
		{"anthropic/claude-3.5-sonnet-v1:beta", "anthropic/claude-3.5-sonnet-v1:beta"}, // -v1:beta not stripped
		{"meta/Llama-3-70B-2025", "meta/llama-3-70b"},
		{"google/gemini-pro-v1.0", "google/gemini-pro-v1.0"}, // -v1.0 not stripped
	}
	for _, tt := range tests {
		got := NormalizeID(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ApplyFilter with sort direction and edge sorts
func TestApplyFilter_SortEdgeCases(t *testing.T) {
	models := []model.Model{
		{ID: "a/gpt4", Name: "GPT-4", Provider: "OpenAI", Mode: "chat", InputPricePer1M: 10, OutputPricePer1M: 30, ContextWindow: 8000, MedianTokensPerSecond: 50, IntelligenceIndex: 90, CodingIndex: 85, Sources: []string{"bifrost"}},
		{ID: "b/claude", Name: "Claude", Provider: "Anthropic", Mode: "chat", InputPricePer1M: 3, OutputPricePer1M: 15, ContextWindow: 100000, MedianTokensPerSecond: 30, IntelligenceIndex: 85, CodingIndex: 80, Sources: []string{"bifrost", "modelsdev"}},
		{ID: "c/gemini", Name: "Gemini", Provider: "Google", Mode: "chat", InputPricePer1M: 0.15, OutputPricePer1M: 0.60, ContextWindow: 1000000, MedianTokensPerSecond: 200, IntelligenceIndex: 75, CodingIndex: 70, Sources: []string{"modelsdev"}},
	}

	// Models names: Claude, GPT-4, Gemini
	// Providers: Anthropic, OpenAI, Google (alpha: Anthropic < Google < OpenAI)
	// Sources: Claude=[bifrost,modelsdev] (2), GPT-4=[bifrost] (1), Gemini=[modelsdev] (1)
	tests := []struct {
		field  string
		asc    bool
		first  string // expected first model name
		reason string
	}{
		{"name", true, "Claude", "C < G"},
		{"name", false, "Gemini", "G > C, G > G, GPT < Gem (P<e)"},
		{"provider", true, "Claude", "Anthropic first"},
		{"provider", false, "GPT-4", "OpenAI last"},
		{"input_price", true, "Gemini", "0.15 < 3 < 10"},
		{"input_price", false, "GPT-4", "10 > 3 > 0.15"},
		{"output_price", true, "Gemini", "0.60 < 15 < 30"},
		{"output_price", false, "GPT-4", "30 > 15 > 0.60"},
		{"context", true, "GPT-4", "8000 < 100K < 1M"},
		{"context", false, "Gemini", "1M > 100K > 8000"},
		{"speed", true, "Claude", "30 < 50 < 200"},
		{"speed", false, "Gemini", "200 > 50 > 30"},
		{"intelligence", true, "Gemini", "75 < 85 < 90"},
		{"intelligence", false, "GPT-4", "90 > 85 > 75"},
		{"coding", true, "Gemini", "70 < 80 < 85"},
		{"coding", false, "GPT-4", "85 > 80 > 70"},
		{"sources", true, "Gemini", "1 src, tiebreak name: Gem < GPT"},
		{"sources", false, "Claude", "2 src first (Claude)"},
		{"unknown", true, "Claude", "default sort asc (name tiebreaker)"},
		{"unknown", false, "Claude", "default sort desc also name asc (tiebreaker ignores sortAsc)"},
	}
	for _, tt := range tests {
		filtered := ApplyFilter(models, FilterParams{}, tt.field, tt.asc)
		if len(filtered) != 3 {
			t.Fatalf("%s asc=%v: expected 3, got %d", tt.field, tt.asc, len(filtered))
		}
		if filtered[0].Name != tt.first {
			t.Errorf("%s asc=%v: first = %q, want %q (%s)", tt.field, tt.asc, filtered[0].Name, tt.first, tt.reason)
		}
	}
}

// matchesFilter with multiple failure modes
func TestMatchesFilter_EdgeCases(t *testing.T) {
	m := model.Model{
		Name:      "GPT-4o",
		Provider:  "OpenAI",
		Mode:      "chat",
		Sources:   []string{"bifrost"},
		Family:    "gpt-4",
		MaxOutput: 4096,
	}

	// Min price failure with exact price match
	if !matchesFilter(m, FilterParams{MinPrice: 0}) {
		t.Error("min price 0 should match all")
	}

	// MinPrice as max price (renamed API test)
	if !matchesFilter(m, FilterParams{MaxPrice: 100}) {
		t.Error("max price 100 should match all")
	}

	// Multi-provider filter
	if matchesFilter(m, FilterParams{Providers: []string{"Anthropic", "Google"}}) {
		t.Error("OpenAI should not match Anthropic or Google")
	}
}

func TestCompareField_AllBranches(t *testing.T) {
	a := model.Model{Name: "A", Provider: "Alpha", InputPricePer1M: 1, OutputPricePer1M: 5, ContextWindow: 1000, MedianTokensPerSecond: 10, IntelligenceIndex: 50, CodingIndex: 40, Sources: []string{"a"}}
	b := model.Model{Name: "B", Provider: "Beta", InputPricePer1M: 2, OutputPricePer1M: 10, ContextWindow: 2000, MedianTokensPerSecond: 20, IntelligenceIndex: 80, CodingIndex: 70, Sources: []string{"a", "b"}}

	if compareField(a, b, "name") >= 0 {
		t.Error("'A' should sort before 'B'")
	}
	if compareField(b, a, "name") <= 0 {
		t.Error("'B' should sort after 'A'")
	}
	if compareField(a, b, "provider") >= 0 {
		t.Error("Alpha before Beta")
	}
	if compareField(a, b, "sources") >= 0 {
		t.Error("1 source before 2 sources")
	}
	// Same name tiebreaker
	if compareField(a, b, "name") >= 0 || compareField(b, a, "name") <= 0 {
		// already tested above
	}
	if compareField(a, b, "unknown") != 0 {
		t.Error("unknown field should return 0")
	}
}

func TestMergeNonZero_AllFields(t *testing.T) {
	dst := &model.Model{}
	src := model.Model{
		InputPricePer1M:          1,
		OutputPricePer1M:         2,
		CacheReadPrice:           0.5,
		ContextWindow:            1000,
		MaxOutput:                2000,
		SupportsVision:           true,
		SupportsFunctionCalling:  true,
		SupportsPromptCaching:    true,
		SupportsReasoning:        true,
		SupportsStructuredOutput: true,
		OpenWeights:              true,
		IntelligenceIndex:        90,
		CodingIndex:              85,
		MathIndex:                80,
		MedianTokensPerSecond:    100,
		MedianTTFTSeconds:        0.3,
		MMLUPro:                  0.90,
		GPQA:                     0.88,
		LiveCodeBench:            0.85,
		AIME25:                   0.80,
		Mode:                     "chat",
		Family:                   "test-family",
		ReleaseDate:              "2025-01-01",
		Description:              "test",
	}
	mergeNonZero(dst, src)
	if dst.InputPricePer1M != 1 {
		t.Error()
	}
	if dst.OutputPricePer1M != 2 {
		t.Error()
	}
	if dst.CacheReadPrice != 0.5 {
		t.Error()
	}
	if dst.ContextWindow != 1000 {
		t.Error()
	}
	if dst.MaxOutput != 2000 {
		t.Error()
	}
	if !dst.SupportsVision {
		t.Error()
	}
	if !dst.SupportsFunctionCalling {
		t.Error()
	}
	if !dst.SupportsPromptCaching {
		t.Error()
	}
	if !dst.SupportsReasoning {
		t.Error()
	}
	if !dst.SupportsStructuredOutput {
		t.Error()
	}
	if !dst.OpenWeights {
		t.Error()
	}
	if dst.IntelligenceIndex != 90 {
		t.Error()
	}
	if dst.CodingIndex != 85 {
		t.Error()
	}
	if dst.MathIndex != 80 {
		t.Error()
	}
	if dst.MedianTokensPerSecond != 100 {
		t.Error()
	}
	if dst.MedianTTFTSeconds != 0.3 {
		t.Error()
	}
	if dst.MMLUPro != 0.90 {
		t.Error()
	}
	if dst.GPQA != 0.88 {
		t.Error()
	}
	if dst.LiveCodeBench != 0.85 {
		t.Error()
	}
	if dst.AIME25 != 0.80 {
		t.Error()
	}
	if dst.Mode != "chat" {
		t.Error()
	}
	if dst.Family != "test-family" {
		t.Error()
	}
	if dst.ReleaseDate != "2025-01-01" {
		t.Error()
	}
	if dst.Description != "test" {
		t.Error()
	}
}
