package fetch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/modelhub/internal/model"
)

// ── FetchJSON ──

func TestFetchJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept header = %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
	}))
	defer ts.Close()

	var result map[string]string
	if err := FetchJSON(ts.URL, "", &result); err != nil {
		t.Fatalf("FetchJSON: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("result = %v", result)
	}
}

func TestFetchJSONWithAPIKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "secret123" {
			t.Errorf("x-api-key header = %q", r.Header.Get("x-api-key"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
	}))
	defer ts.Close()

	var result map[string]string
	if err := FetchJSON(ts.URL, "secret123", &result); err != nil {
		t.Fatalf("FetchJSON: %v", err)
	}
	if result["ok"] != "yes" {
		t.Errorf("result = %v", result)
	}
}

func TestFetchJSONNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer ts.Close()

	var result any
	err := FetchJSON(ts.URL, "", &result)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFetchJSONBadURL(t *testing.T) {
	var result any
	err := FetchJSON("http://127.0.0.1:1/nonexistent", "", &result)
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

// ── Bifrost ──

func TestFetchBifrost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]RawBifrostEntry{
			"openai/gpt-4o": {
				Provider:                "OpenAI",
				BaseModel:               "gpt-4o",
				Mode:                    "chat",
				InputCost:               0.0000025,
				OutputCost:              0.00001,
				MaxInputTokens:          128000,
				MaxOutputTokens:         4096,
				SupportsVision:          true,
				SupportsFunctionCalling: true,
				SupportsPromptCaching:   true,
				CacheReadInputCost:      0.00000125,
			},
			"anthropic/claude-3-5-sonnet-20241022": {
				Provider:   "Anthropic",
				BaseModel:  "claude-3-5-sonnet-20241022",
				Mode:       "chat",
				InputCost:  0.000003,
				OutputCost: 0.000015,
			},
			// Resolution-prefixed entry should be skipped
			"1024-x-1024/openai/dall-e-3": {
				Provider:  "OpenAI",
				BaseModel: "dall-e-3",
				Mode:      "image_generation",
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer ts.Close()

	models, err := FetchBifrost(ts.URL)
	if err != nil {
		t.Fatalf("FetchBifrost: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("got %d models, want 2 (resolution-prefixed should be skipped)", len(models))
	}

	// Check gpt-4o fields
	var gpt4o *model.Model
	for _, m := range models {
		if m.ID == "OpenAI/gpt-4o" {
			gpt4o = &m
			break
		}
	}
	if gpt4o == nil {
		t.Fatal("OpenAI/gpt-4o not found")
	}
	if gpt4o.InputPricePer1M != 2.5 {
		t.Errorf("InputPricePer1M = %f, want 2.5", gpt4o.InputPricePer1M)
	}
	if gpt4o.OutputPricePer1M != 10.0 {
		t.Errorf("OutputPricePer1M = %f, want 10.0", gpt4o.OutputPricePer1M)
	}
	if !gpt4o.SupportsVision {
		t.Error("SupportsVision should be true")
	}
	if !gpt4o.SupportsFunctionCalling {
		t.Error("SupportsFunctionCalling should be true")
	}
}

func TestFetchBifrost_ResolutionFilter(t *testing.T) {
	// Test the isResolution helper directly
	if !isResolution("1024-x-1024") {
		t.Error("expected '1024-x-1024' to be resolution")
	}
	if isResolution("gpt-4o") {
		t.Error("expected 'gpt-4o' to NOT be resolution")
	}
	// ponytail: "512x512" has only 1 "x" so the heuristic doesn't catch it.
	// Known limitation of the Count approach — upgrade to regex for full resolution detection.
	if isResolution("512x512") {
		t.Error("expected '512x512' to NOT match the heuristic isResolution")
	}
}

func TestInferProviderFromKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"openai/gpt-4o", "openai"},
		{"anthropic/claude", "anthropic"},
		{"amazon.nova-pro", "bedrock"},
		{"us.anthropic.claude", "bedrock"},
		{"apac.xyz", "bedrock"},
		{"ai21.jamba", "bedrock"},
		{"noprefix", "unknown"},
	}
	for _, tt := range tests {
		got := inferProviderFromKey(tt.key)
		if got != tt.want {
			t.Errorf("inferProviderFromKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestFetchBifrost_InferredProvider(t *testing.T) {
	// Entry with empty provider should infer from key
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]RawBifrostEntry{
			"openai/gpt-4o-mini": {
				Provider:  "",
				BaseModel: "gpt-4o-mini",
				Mode:      "chat",
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer ts.Close()

	models, err := FetchBifrost(ts.URL)
	if err != nil {
		t.Fatalf("FetchBifrost: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models", len(models))
	}
	if models[0].Provider != "openai" {
		t.Errorf("Provider = %q, want openai", models[0].Provider)
	}
}

// ── Models.dev ──

func TestFetchModelsDev(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]ModelsDevProvider{
			"openai": {
				Name: "OpenAI",
				Models: map[string]ModelsDevModel{
					"gpt-4o": {
						ID:               "openai/gpt-4o",
						Name:             "GPT-4o",
						Description:      "Latest multimodal model",
						Family:           "gpt-4",
						Reasoning:        false,
						ToolCall:         true,
						StructuredOutput: true,
						OpenWeights:      false,
						ReleaseDate:      "2024-05-13",
						Limit:            ModelsDevLimit{Context: 128000, Output: 4096},
						Cost:             ModelsDevCost{Input: 2.50, Output: 10.00, CacheRead: 1.25},
						Modalities:       ModelsDevModalities{Input: []string{"text", "image"}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer ts.Close()

	models, err := FetchModelsDev(ts.URL)
	if err != nil {
		t.Fatalf("FetchModelsDev: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	m := models[0]
	if m.ID != "openai/gpt-4o" {
		t.Errorf("ID = %q", m.ID)
	}
	if m.Provider != "OpenAI" {
		t.Errorf("Provider = %q", m.Provider)
	}
	if m.InputPricePer1M != 2.50 {
		t.Errorf("InputPricePer1M = %f", m.InputPricePer1M)
	}
	if !m.SupportsVision {
		t.Error("SupportsVision should be true for image modality")
	}
	if !m.SupportsFunctionCalling {
		t.Error("SupportsFunctionCalling should be true")
	}
	if !m.SupportsStructuredOutput {
		t.Error("SupportsStructuredOutput should be true")
	}
	if m.Description != "Latest multimodal model" {
		t.Errorf("Description = %q", m.Description)
	}
}

func TestStringsContainsRune(t *testing.T) {
	if !stringsContainsRune("hello/world", '/') {
		t.Error("expected true for '/' in 'hello/world'")
	}
	if stringsContainsRune("hello", '/') {
		t.Error("expected false for '/' in 'hello'")
	}
	if stringsContainsRune("", '/') {
		t.Error("expected false for empty string")
	}
}

func TestFetchModelsDev_IDNormalization(t *testing.T) {
	// Model without '/' in its ID should be prefixed with provider key
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]ModelsDevProvider{
			"google": {
				Name: "Google",
				Models: map[string]ModelsDevModel{
					"gemini-2.0-flash": {
						ID:         "gemini-2.0-flash", // no provider prefix
						Name:       "Gemini 2.0 Flash",
						Limit:      ModelsDevLimit{Context: 1000000},
						Modalities: ModelsDevModalities{Input: []string{"text"}},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(data)
	}))
	defer ts.Close()

	models, err := FetchModelsDev(ts.URL)
	if err != nil {
		t.Fatalf("FetchModelsDev: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models", len(models))
	}
	if models[0].ID != "google/gemini-2.0-flash" {
		t.Errorf("ID = %q, want google/gemini-2.0-flash", models[0].ID)
	}
}

// ── Artificial Analysis ──

func TestFetchAA(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		resp := AAResponse{
			Status: 200,
			Data: []AAModel{
				{
					ID:   "1",
					Name: "GPT-4o",
					Slug: "gpt-4o",
					Creator: AACreator{
						ID:   "openai",
						Name: "OpenAI",
					},
					Evaluations: AAEvaluations{
						IntelligenceIndex: 85.0,
						MMLUPro:           0.88,
					},
					Pricing: AAPricing{
						Input:  2.50,
						Output: 10.00,
					},
					Speed: 45.0,
					TTFT:  0.5,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	models, err := FetchAA(ts.URL, "test-key")
	if err != nil {
		t.Fatalf("FetchAA: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	m := models[0]
	if m.ID != "OpenAI/gpt-4o" {
		t.Errorf("ID = %q", m.ID)
	}
	if m.IntelligenceIndex != 85.0 {
		t.Errorf("IntelligenceIndex = %f", m.IntelligenceIndex)
	}
	if m.MedianTokensPerSecond != 45.0 {
		t.Errorf("Speed = %f", m.MedianTokensPerSecond)
	}
	if m.MedianTTFTSeconds != 0.5 {
		t.Errorf("TTFT = %f", m.MedianTTFTSeconds)
	}
}

func TestFetchAANoKey(t *testing.T) {
	models, err := FetchAA("http://example.com", "")
	if err != nil {
		t.Fatalf("FetchAA with empty key: %v", err)
	}
	if models != nil {
		t.Error("expected nil models when no API key")
	}
}

func TestFetchAAMissingCreator(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AAResponse{
			Status: 200,
			Data: []AAModel{
				{
					ID:   "1",
					Name: "Mystery Model",
					Slug: "mystery",
					Creator: AACreator{
						ID:   "",
						Name: "",
					},
					Evaluations: AAEvaluations{},
					Pricing:     AAPricing{},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	models, err := FetchAA(ts.URL, "test-key")
	if err != nil {
		t.Fatalf("FetchAA: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models", len(models))
	}
	if models[0].Provider != "unknown" {
		t.Errorf("Provider = %q, want unknown", models[0].Provider)
	}
}
