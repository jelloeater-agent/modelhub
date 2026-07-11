// Package model defines the unified data model for all AI model sources.
package model

import (
	"os"
	"path/filepath"
	"time"
)

// xdgPath returns $XDG_<env>/modelhub/<file> if the env var is set,
// otherwise ~/.modelhub/<file>. Always respects XDG when configured.
func xdgPath(env, file string) string {
	if d := os.Getenv("XDG_" + env + "_HOME"); d != "" {
		return filepath.Join(d, "modelhub", file)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".modelhub", file)
}

// ConfigPath returns the default config path following XDG.
func ConfigPath() string { return xdgPath("CONFIG", "config.json") }

// CachePath returns the default cache path following XDG.
func CachePath() string { return xdgPath("CACHE", "cache.json") }

// Model is the unified representation merging data from all sources.
type Model struct {
	// Identity
	ID       string `json:"id"`       // Normalized unique ID e.g. "openai/gpt-4o"
	Name     string `json:"name"`     // Human-readable name
	Provider string `json:"provider"` // Provider name
	Mode     string `json:"mode"`     // chat, image_generation, embedding, audio, etc.

	// Pricing (per 1M tokens where applicable)
	InputPricePer1M  float64 `json:"input_price_per_1m"`
	OutputPricePer1M float64 `json:"output_price_per_1m"`
	CacheReadPrice   float64 `json:"cache_read_price"`

	// Limits
	ContextWindow int `json:"context_window"`
	MaxOutput     int `json:"max_output"`

	// Capabilities
	SupportsVision           bool `json:"supports_vision"`
	SupportsFunctionCalling  bool `json:"supports_function_calling"`
	SupportsPromptCaching    bool `json:"supports_prompt_caching"`
	SupportsReasoning        bool `json:"supports_reasoning"`
	SupportsStructuredOutput bool `json:"supports_structured_output"`
	OpenWeights              bool `json:"open_weights"`

	// Performance (Artificial Analysis)
	IntelligenceIndex     float64 `json:"intelligence_index"`
	CodingIndex           float64 `json:"coding_index"`
	MathIndex             float64 `json:"math_index"`
	MedianTokensPerSecond float64 `json:"median_tokens_per_second"`
	MedianTTFTSeconds     float64 `json:"median_ttft_seconds"`

	// Benchmarks (Artificial Analysis)
	MMLUPro       float64 `json:"mmlu_pro"`
	GPQA          float64 `json:"gpqa"`
	LiveCodeBench float64 `json:"live_code_bench"`
	AIME25        float64 `json:"aime_25"`

	// Metadata
	Family      string `json:"family"`
	ReleaseDate string `json:"release_date"`
	Description string `json:"description"`

	// Source tracking
	Sources []string `json:"sources"` // Which sources contributed data
}

// Cache holds the full app state for persistence.
type Cache struct {
	Models    []Model           `json:"models"`
	FetchedAt map[string]string `json:"fetched_at"` // source -> ISO timestamp
	Version   int               `json:"version"`
}

// Config holds user configuration.
type Config struct {
	AAAPIKey           string `json:"aa_api_key"`
	RefreshIntervalMin int    `json:"refresh_interval_minutes"`
	CachePath          string `json:"cache_path"`
	BifrostURL         string `json:"bifrost_url"`
	ModelsDevURL       string `json:"models_dev_url"`
	AAURL              string `json:"aa_url"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		RefreshIntervalMin: 60,
		CachePath:          CachePath(),
		BifrostURL:         "https://getbifrost.ai/datasheet",
		ModelsDevURL:       "https://models.dev/api.json",
		AAURL:              "https://artificialanalysis.ai/api/v2/data/llms/models",
	}
}

// Stats returns quick statistics about a model collection.
type Stats struct {
	Total      int            `json:"total"`
	BySource   map[string]int `json:"by_source"`
	ByMode     map[string]int `json:"by_mode"`
	ByProvider map[string]int `json:"by_provider"`
	LastUpdate string         `json:"last_update"`
	Version    int            `json:"version"`
}

func ComputeStats(models []Model, fetchedAt map[string]string) Stats {
	s := Stats{
		Total:      len(models),
		BySource:   make(map[string]int),
		ByMode:     make(map[string]int),
		ByProvider: make(map[string]int),
		LastUpdate: fetchedAt["_merged"],
	}
	for _, m := range models {
		for _, src := range m.Sources {
			s.BySource[src]++
		}
		if m.Mode != "" {
			s.ByMode[m.Mode]++
		}
		if m.Provider != "" {
			s.ByProvider[m.Provider]++
		}
	}
	// Compute version as unix timestamp of last update for change detection
	if t, err := time.Parse(time.RFC3339, s.LastUpdate); err == nil {
		s.Version = int(t.Unix())
	}
	return s
}
