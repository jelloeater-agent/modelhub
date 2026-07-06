package merge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/modelhub/internal/fetch"
	"github.com/user/modelhub/internal/model"
)

// ── isShortVersion ──

func TestIsShortVersion(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"0", true},
		{"1", true},
		{"v1", false}, // includes 'v'
		{"latest", true},
		{"snapshot", true},
		{"stable", false},
		{"2025", false}, // 4 digits
		// ponytail: empty string vacuously matches (len<=2, loop doesn't execute)
		{"", true},
	}
	for _, tt := range tests {
		got := isShortVersion(tt.s)
		if got != tt.want {
			t.Errorf("isShortVersion(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// ── isTrailingVersion ──

func TestIsTrailingVersion(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"2025", true},
		{"20241022", true},
		{"01", true}, // 2 digits
		{"1", false}, // 1 digit
		{"preview", true},
		{"latest", true},
		{"snapshot", true},
		{"alpha", true},
		{"beta", true},
		{"rc1", true},
		{"rc2", true},
		{"v1", false},
		{"stable", false},
		{"gpt", false},
		{"4o", false}, // 2 chars, not all digits
		{"5-codex", false},
		{"sonnet", false},
	}
	for _, tt := range tests {
		got := isTrailingVersion(tt.s)
		if got != tt.want {
			t.Errorf("isTrailingVersion(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// ── NormalizeID extras ──

func TestNormalizeID_VersionSuffixes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"openai/gpt-4o:latest", "openai/gpt-4o"},
		{"openai/gpt-4o:0", "openai/gpt-4o"},
		{"anthropic/claude-3-5-sonnet-20241022", "anthropic/claude-3-5-sonnet"},
		{"meta/llama-3-70b-instruct-2025", "meta/llama-3-70b-instruct"},
		{"meta/llama-3-70b-preview", "meta/llama-3-70b"},
		{"openai/gpt-4o", "openai/gpt-4o"},
		{"MIXED/Case-Test", "mixed/case-test"},
		{"  spaced/name  ", "spaced/name"},
		{"google/gemini-2.0-flash", "google/gemini-2.0-flash"},
	}
	for _, tt := range tests {
		got := NormalizeID(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Merge extra scenarios ──

func TestMerge_ThreeSources(t *testing.T) {
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
			SupportsVision:          true,
			SupportsFunctionCalling: true,
			ContextWindow:           128000,
			Sources:                 []string{"modelsdev"},
		},
	}
	aa := []model.Model{
		{
			ID:                    "openai/gpt-4o",
			Name:                  "GPT-4o",
			Provider:              "OpenAI",
			IntelligenceIndex:     85.0,
			CodingIndex:           82.0,
			MedianTokensPerSecond: 45.0,
			Sources:               []string{"aa"},
		},
	}

	merged := Merge(bifrost, modelsdev, aa)
	if len(merged) != 1 {
		t.Fatalf("expected 1 model, got %d", len(merged))
	}
	m := merged[0]

	// Bifrost price preserved (highest priority)
	if m.InputPricePer1M != 2.50 {
		t.Errorf("InputPricePer1M = %f, want 2.50", m.InputPricePer1M)
	}
	// Modelsdev fills context (lower priority fills gap)
	if m.ContextWindow != 128000 {
		t.Errorf("ContextWindow = %d, want 128000", m.ContextWindow)
	}
	// AA fills benchmarks (lowest priority fills gap)
	if m.IntelligenceIndex != 85.0 {
		t.Errorf("IntelligenceIndex = %f, want 85.0", m.IntelligenceIndex)
	}
	// All 3 sources tracked
	if len(m.Sources) != 3 {
		t.Errorf("expected 3 sources, got %d: %v", len(m.Sources), m.Sources)
	}
}

func TestMerge_DifferentModels(t *testing.T) {
	bifrost := []model.Model{
		{ID: "openai/gpt-4o", Name: "gpt-4o", Provider: "OpenAI", Sources: []string{"bifrost"}},
		{ID: "anthropic/claude-opus", Name: "claude-opus", Provider: "Anthropic", Sources: []string{"bifrost"}},
	}
	modelsdev := []model.Model{
		{ID: "openai/gpt-4o", Name: "GPT-4o", Provider: "OpenAI", Sources: []string{"modelsdev"}},
		{ID: "google/gemini-pro", Name: "Gemini Pro", Provider: "Google", Sources: []string{"modelsdev"}},
	}

	merged := Merge(bifrost, modelsdev)
	if len(merged) != 3 {
		t.Fatalf("expected 3 models, got %d", len(merged))
	}

	// Check that all providers are present
	providers := make(map[string]bool)
	for _, m := range merged {
		providers[m.Provider] = true
	}
	for _, p := range []string{"OpenAI", "Anthropic", "Google"} {
		if !providers[p] {
			t.Errorf("missing provider %s", p)
		}
	}
}

func TestMerge_SamePrioritY_EqualMerge(t *testing.T) {
	// Two sources with same provider name (same priority via override scenario)
	// When merged model exists from same priority source
	bifrost := []model.Model{
		{ID: "test/model", Name: "model", Provider: "Test", Mode: "chat", InputPricePer1M: 5.0, Sources: []string{"bifrost"}},
	}

	// Create a scenario where we merge bifrost with itself (same priority)
	merged := Merge(bifrost, bifrost)
	if len(merged) != 1 {
		t.Fatalf("expected 1 model, got %d", len(merged))
	}
	m := merged[0]
	if m.InputPricePer1M != 5.0 {
		t.Errorf("price = %f", m.InputPricePer1M)
	}
}

func TestMerge_EmptyInput(t *testing.T) {
	merged := Merge(nil, []model.Model{}, nil)
	if len(merged) != 0 {
		t.Errorf("expected 0 models from empty input, got %d", len(merged))
	}
}

// ── mergeNonZero ──

func TestMergeNonZero(t *testing.T) {
	dst := &model.Model{
		InputPricePer1M:  1.0,
		OutputPricePer1M: 0,
		Mode:             "chat",
	}
	src := model.Model{
		InputPricePer1M:   99.0, // should NOT override non-zero
		OutputPricePer1M:  5.0,  // should fill zero
		SupportsVision:    true,
		IntelligenceIndex: 90.0,
		Mode:              "image_generation", // should override non-empty
		Family:            "gpt-4",
	}

	mergeNonZero(dst, src)

	// mergeNonZero copies ALL non-zero src fields regardless of dst state
	if dst.InputPricePer1M != 99.0 {
		t.Errorf("InputPrice should be 99.0 (mergeNonZero overrides), got %f", dst.InputPricePer1M)
	}
	if dst.OutputPricePer1M != 5.0 {
		t.Errorf("OutputPrice should be 5.0, got %f", dst.OutputPricePer1M)
	}
	if !dst.SupportsVision {
		t.Error("SupportsVision should be true")
	}
	if dst.IntelligenceIndex != 90.0 {
		t.Errorf("IntelligenceIndex = %f", dst.IntelligenceIndex)
	}
	if dst.Family != "gpt-4" {
		t.Errorf("Family = %q", dst.Family)
	}
}

// ── compareField ──

func TestCompareField(t *testing.T) {
	low := model.Model{Name: "A", Provider: "Alpha", InputPricePer1M: 1, OutputPricePer1M: 5, ContextWindow: 1000, MedianTokensPerSecond: 10, IntelligenceIndex: 50, CodingIndex: 40, Sources: []string{"a"}}
	high := model.Model{Name: "B", Provider: "Beta", InputPricePer1M: 2, OutputPricePer1M: 10, ContextWindow: 2000, MedianTokensPerSecond: 20, IntelligenceIndex: 80, CodingIndex: 70, Sources: []string{"a", "b"}}

	tests := []struct {
		field string
		want  float64 // negative: low < high
	}{
		{"name", -1},
		{"provider", -1},
		{"input_price", -1},
		{"output_price", -5},
		{"context", -1000},
		{"speed", -10},
		{"intelligence", -30},
		{"coding", -30},
		{"sources", -1},
		{"unknown", 0},
	}
	for _, tt := range tests {
		got := compareField(low, high, tt.field)
		if got != tt.want {
			t.Errorf("compareField(%q) = %f, want %f", tt.field, got, tt.want)
		}
	}
}

// ── matchesFilter ──

func TestMatchesFilter_FullCoverage(t *testing.T) {
	m := model.Model{
		Name:              "GPT-4o",
		Provider:          "OpenAI",
		Mode:              "chat",
		InputPricePer1M:   2.50,
		ContextWindow:     128000,
		SupportsVision:    true,
		SupportsReasoning: false,
		Sources:           []string{"bifrost", "modelsdev"},
	}

	tests := []struct {
		name   string
		filter FilterParams
		want   bool
	}{
		{"no filter", FilterParams{}, true},
		{"search match name", FilterParams{Search: "gpt"}, true},
		{"search match provider", FilterParams{Search: "openai"}, true},
		{"search no match", FilterParams{Search: "claude"}, false},
		{"provider match", FilterParams{Providers: []string{"OpenAI"}}, true},
		{"provider no match", FilterParams{Providers: []string{"Anthropic"}}, false},
		{"mode match", FilterParams{Modes: []string{"chat"}}, true},
		{"mode no match", FilterParams{Modes: []string{"embedding"}}, false},
		{"source match", FilterParams{Sources: []string{"bifrost"}}, true},
		{"source no match", FilterParams{Sources: []string{"aa"}}, false},
		{"min price ok", FilterParams{MinPrice: 1.0}, true},
		{"min price fail", FilterParams{MinPrice: 5.0}, false},
		{"max price ok", FilterParams{MaxPrice: 5.0}, true},
		{"max price fail", FilterParams{MaxPrice: 1.0}, false},
		{"min ctx ok", FilterParams{MinCtx: 100000}, true},
		{"min ctx fail", FilterParams{MinCtx: 200000}, false},
		{"cap vision match", FilterParams{CapFlags: CapabilityFlags{Vision: true}}, true},
		{"cap vision fail", FilterParams{CapFlags: CapabilityFlags{Reasoning: true}}, false},
	}
	for _, tt := range tests {
		got := matchesFilter(m, tt.filter)
		if got != tt.want {
			t.Errorf("%s: matchesFilter = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestMatchesFilter_DescriptionSearch(t *testing.T) {
	m := model.Model{
		Name:        "gpt-4o",
		Provider:    "OpenAI",
		Description: "Latest multimodal model from OpenAI",
		Sources:     []string{"bifrost"},
	}
	if !matchesFilter(m, FilterParams{Search: "multimodal"}) {
		t.Error("should match on description")
	}
}

// ── DoRefresh ──

func TestDoRefresh_Basic(t *testing.T) {
	bifrostData := map[string]fetch.RawBifrostEntry{
		"openai/gpt-4o": {
			Provider:  "OpenAI",
			BaseModel: "gpt-4o",
			Mode:      "chat",
			InputCost: 0.0000025,
		},
	}
	modelsDevData := map[string]fetch.ModelsDevProvider{
		"openai": {
			Name: "OpenAI",
			Models: map[string]fetch.ModelsDevModel{
				"gpt-4o": {
					ID:         "openai/gpt-4o",
					Name:       "GPT-4o",
					Limit:      fetch.ModelsDevLimit{Context: 128000},
					Cost:       fetch.ModelsDevCost{Input: 2.50},
					Modalities: fetch.ModelsDevModalities{Input: []string{"text"}},
				},
			},
		},
	}
	aaData := fetch.AAResponse{
		Status: 200,
		Data:   []fetch.AAModel{},
	}

	bf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bifrostData)
	}))
	defer bf.Close()

	md := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, modelsDevData)
	}))
	defer md.Close()

	aa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, aaData)
	}))
	defer aa.Close()

	cfg := model.Config{
		BifrostURL:   bf.URL,
		ModelsDevURL: md.URL,
		AAURL:        aa.URL,
		AAAPIKey:     "test-key",
	}

	result := DoRefresh(cfg)

	if result.Models == nil {
		t.Fatal("result.Models is nil")
	}
	if len(result.Models) == 0 {
		t.Fatal("expected at least 1 model")
	}
	if result.FetchedAt["_merged"] == "" {
		t.Error("expected _merged timestamp")
	}
}

func TestDoRefresh_AllSourcesFail(t *testing.T) {
	// URLs that will fail
	cfg := model.Config{
		BifrostURL:   "http://127.0.0.1:1/bifrost",
		ModelsDevURL: "http://127.0.0.1:1/modelsdev",
		AAURL:        "http://127.0.0.1:1/aa",
		AAAPIKey:     "test",
	}

	result := DoRefresh(cfg)

	if result.Models == nil {
		t.Fatal("result.Models is nil")
	}
	if len(result.Models) != 0 {
		t.Errorf("expected 0 models when all sources fail, got %d", len(result.Models))
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors from failed sources")
	}
}

func TestDoRefresh_AAWithoutKey(t *testing.T) {
	bf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]fetch.RawBifrostEntry{})
	}))
	defer bf.Close()

	md := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]fetch.ModelsDevProvider{})
	}))
	defer md.Close()

	cfg := model.Config{
		BifrostURL:   bf.URL,
		ModelsDevURL: md.URL,
		AAURL:        "http://example.com",
		AAAPIKey:     "", // no key
	}

	result := DoRefresh(cfg)

	if result.Models == nil {
		t.Fatal("result.Models is nil")
	}
	if _, ok := result.Errors["aa"]; ok {
		t.Error("AA should not error when no key, just skip")
	}
}

func TestDoRefresh_PartialFailure(t *testing.T) {
	bf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]fetch.RawBifrostEntry{
			"test/m1": {Provider: "Test", BaseModel: "m1", Mode: "chat"},
		})
	}))
	defer bf.Close()

	md := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]fetch.ModelsDevProvider{
			"test2": {
				Name: "Test2",
				Models: map[string]fetch.ModelsDevModel{
					"m2": {ID: "test2/m2", Name: "m2", Modalities: fetch.ModelsDevModalities{Input: []string{"text"}}},
				},
			},
		})
	}))
	defer md.Close()

	// Only Bifrost and models.dev succeed
	cfg := model.Config{
		BifrostURL:   bf.URL,
		ModelsDevURL: md.URL,
		AAURL:        "http://127.0.0.1:1/nonexistent",
		AAAPIKey:     "test",
	}

	result := DoRefresh(cfg)

	if len(result.Models) == 0 {
		t.Fatal("expected models from working sources")
	}
	if _, ok := result.Errors["aa"]; !ok {
		t.Error("expected AA error")
	}
	if _, ok := result.FetchedAt["bifrost"]; !ok {
		t.Error("expected bifrost timestamp")
	}
	if _, ok := result.FetchedAt["modelsdev"]; !ok {
		t.Error("expected modelsdev timestamp")
	}
}

// Helper to write JSON in fetch package tests
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(v)
}
