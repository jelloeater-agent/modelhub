package model

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RefreshIntervalMin != 60 {
		t.Errorf("RefreshIntervalMin = %d, want 60", cfg.RefreshIntervalMin)
	}
	if cfg.CachePath == "" {
		t.Error("CachePath should not be empty")
	}
	if !strings.HasSuffix(cfg.CachePath, "modelhub/cache.json") {
		t.Errorf("CachePath = %q, want .../modelhub/cache.json", cfg.CachePath)
	}
	if cfg.BifrostURL == "" {
		t.Error("BifrostURL should not be empty")
	}
	if cfg.ModelsDevURL == "" {
		t.Error("ModelsDevURL should not be empty")
	}
	if cfg.AAURL == "" {
		t.Error("AAURL should not be empty")
	}
}

func TestComputeStats(t *testing.T) {
	models := []Model{
		{ID: "a/m1", Provider: "OpenAI", Mode: "chat", Sources: []string{"bifrost"}},
		{ID: "b/m2", Provider: "Anthropic", Mode: "chat", Sources: []string{"bifrost", "modelsdev"}},
		{ID: "c/m3", Provider: "OpenAI", Mode: "image_generation", Sources: []string{"aa"}},
		{ID: "d/m4", Provider: "Google", Mode: "embedding", Sources: []string{"modelsdev"}},
	}
	fetchedAt := map[string]string{
		"_merged": "2025-01-15T10:00:00Z",
		"bifrost": "2025-01-15T10:00:00Z",
	}

	stats := ComputeStats(models, fetchedAt)

	if stats.Total != 4 {
		t.Errorf("Total = %d, want 4", stats.Total)
	}
	if stats.BySource["bifrost"] != 2 {
		t.Errorf("BySource[bifrost] = %d, want 2", stats.BySource["bifrost"])
	}
	if stats.BySource["modelsdev"] != 2 {
		t.Errorf("BySource[modelsdev] = %d, want 2", stats.BySource["modelsdev"])
	}
	if stats.BySource["aa"] != 1 {
		t.Errorf("BySource[aa] = %d, want 1", stats.BySource["aa"])
	}
	if stats.ByProvider["OpenAI"] != 2 {
		t.Errorf("ByProvider[OpenAI] = %d, want 2", stats.ByProvider["OpenAI"])
	}
	if stats.ByMode["chat"] != 2 {
		t.Errorf("ByMode[chat] = %d, want 2", stats.ByMode["chat"])
	}
	if stats.LastUpdate != "2025-01-15T10:00:00Z" {
		t.Errorf("LastUpdate = %q, want 2025-01-15T10:00:00Z", stats.LastUpdate)
	}
	if stats.Version != 1736935200 {
		t.Errorf("Version = %d, want 1736935200", stats.Version)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	stats := ComputeStats(nil, nil)
	if stats.Total != 0 {
		t.Errorf("Total = %d, want 0", stats.Total)
	}
	if stats.LastUpdate != "" {
		t.Errorf("LastUpdate should be empty")
	}
}

func TestComputeStatsNoSource(t *testing.T) {
	models := []Model{
		{ID: "a/m1", Provider: "", Mode: ""},
	}
	stats := ComputeStats(models, map[string]string{})
	if stats.Total != 1 {
		t.Errorf("Total = %d, want 1", stats.Total)
	}
	// Empty provider and mode should not create map entries
	if len(stats.ByProvider) != 0 {
		t.Errorf("ByProvider should be empty, got %v", stats.ByProvider)
	}
	if len(stats.ByMode) != 0 {
		t.Errorf("ByMode should be empty, got %v", stats.ByMode)
	}
}

func TestCacheVersion(t *testing.T) {
	// Version should be 0 for invalid timestamps
	stats := ComputeStats(nil, map[string]string{"_merged": "not-a-timestamp"})
	if stats.Version != 0 {
		t.Errorf("Version = %d, want 0 for invalid timestamp", stats.Version)
	}
}
