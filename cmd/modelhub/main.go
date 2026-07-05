package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/user/modelhub/internal/cache"
	"github.com/user/modelhub/internal/merge"
	"github.com/user/modelhub/internal/model"
	"github.com/user/modelhub/internal/tui"
)

func main() {
	cfgPath := flag.String("config", "", "Path to config file")
	refresh := flag.Bool("refresh", false, "Force refresh all data and exit")
	flag.Parse()

	cfg := resolveConfig(*cfgPath)

	store, err := cache.NewStore(cfg.CachePath)
	if err != nil {
		log.Fatalf("Failed to init cache: %v", err)
	}

	if *refresh {
		doRefreshAndExit(cfg, store)
		return
	}

	cached, _ := store.Load()

	tm, err := tui.NewModel(cfg, store, cached)
	if err != nil {
		log.Fatalf("Failed to create TUI: %v", err)
	}

	p := tea.NewProgram(tm, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func resolveConfig(cfgPath string) model.Config {
	cfg := model.DefaultConfig()

	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".modelhub", "config.json")
	}

	data, err := os.ReadFile(cfgPath)
	if err == nil {
		var fileCfg model.Config
		if json.Unmarshal(data, &fileCfg) == nil {
			if fileCfg.AAAPIKey != "" {
				cfg.AAAPIKey = fileCfg.AAAPIKey
			}
			if fileCfg.CachePath != "" {
				cfg.CachePath = fileCfg.CachePath
			}
			if fileCfg.RefreshIntervalMin > 0 {
				cfg.RefreshIntervalMin = fileCfg.RefreshIntervalMin
			}
			if fileCfg.BifrostURL != "" {
				cfg.BifrostURL = fileCfg.BifrostURL
			}
			if fileCfg.ModelsDevURL != "" {
				cfg.ModelsDevURL = fileCfg.ModelsDevURL
			}
			if fileCfg.AAURL != "" {
				cfg.AAURL = fileCfg.AAURL
			}
		}
	}

	if v := os.Getenv("AA_API_KEY"); v != "" {
		cfg.AAAPIKey = v
	}

	return cfg
}

func doRefreshAndExit(cfg model.Config, store *cache.Store) {
	fmt.Println("Fetching data from all sources...")
	result := merge.DoRefresh(cfg)

	c := &model.Cache{
		Models:    result.Models,
		FetchedAt: result.FetchedAt,
	}
	if err := store.Save(c); err != nil {
		log.Fatalf("Failed to save cache: %v", err)
	}

	stats := model.ComputeStats(result.Models, result.FetchedAt)
	fmt.Printf("✓ Merged %d models from %d sources\n", stats.Total, len(stats.BySource))
	for src, count := range stats.BySource {
		fmt.Printf("  • %s: %d models\n", src, count)
	}
	if len(result.Errors) > 0 {
		for src, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "⚠ %s: %s\n", src, err)
		}
	}
}
