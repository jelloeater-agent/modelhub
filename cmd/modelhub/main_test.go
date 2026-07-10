package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jelloeater-agent/modelhub/internal/model"
)

func TestResolveConfig_Defaults(t *testing.T) {
	cfg := resolveConfig("")
	// Should use defaults when no config file exists
	def := model.DefaultConfig()
	if cfg.AAAPIKey != def.AAAPIKey {
		t.Errorf("AAAPIKey = %q, want %q", cfg.AAAPIKey, def.AAAPIKey)
	}
	if cfg.RefreshIntervalMin != def.RefreshIntervalMin {
		t.Errorf("RefreshIntervalMin = %d", cfg.RefreshIntervalMin)
	}
	if cfg.CachePath != def.CachePath {
		t.Errorf("CachePath = %q", cfg.CachePath)
	}
}

func TestResolveConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	fileCfg := model.Config{
		AAAPIKey:           "from-file-key",
		RefreshIntervalMin: 30,
		CachePath:          filepath.Join(dir, "custom-cache.json"),
		BifrostURL:         "https://custom-bifrost.example.com",
	}
	data, _ := json.MarshalIndent(fileCfg, "", "  ")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Override home to our temp dir so the default path doesn't interfere
	cfg := resolveConfig(cfgPath)

	if cfg.AAAPIKey != "from-file-key" {
		t.Errorf("AAAPIKey = %q", cfg.AAAPIKey)
	}
	if cfg.RefreshIntervalMin != 30 {
		t.Errorf("RefreshIntervalMin = %d", cfg.RefreshIntervalMin)
	}
	if cfg.BifrostURL != "https://custom-bifrost.example.com" {
		t.Errorf("BifrostURL = %q", cfg.BifrostURL)
	}
}

func TestResolveConfig_EnvVarOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	fileCfg := model.Config{
		AAAPIKey: "from-file",
	}
	data, _ := json.MarshalIndent(fileCfg, "", "  ")
	os.WriteFile(cfgPath, data, 0644)

	t.Setenv("AA_API_KEY", "from-env")

	cfg := resolveConfig(cfgPath)
	if cfg.AAAPIKey != "from-env" {
		t.Errorf("AAAPIKey = %q, want 'from-env' (env should override file)", cfg.AAAPIKey)
	}
}

func TestResolveConfig_PartialFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Only set AAAPIKey, everything else should be default
	partial := map[string]string{"aa_api_key": "partial-key"}
	data, _ := json.MarshalIndent(partial, "", "  ")
	os.WriteFile(cfgPath, data, 0644)

	cfg := resolveConfig(cfgPath)
	if cfg.AAAPIKey != "partial-key" {
		t.Errorf("AAAPIKey = %q", cfg.AAAPIKey)
	}
	if cfg.RefreshIntervalMin != 60 {
		t.Errorf("RefreshIntervalMin = %d, want default 60", cfg.RefreshIntervalMin)
	}
}

func TestResolveConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	os.WriteFile(cfgPath, []byte("not json"), 0644)

	// Should fall back to defaults without crashing
	cfg := resolveConfig(cfgPath)
	def := model.DefaultConfig()
	if cfg.AAAPIKey != def.AAAPIKey {
		t.Errorf("AAAPIKey = %q", cfg.AAAPIKey)
	}
}
