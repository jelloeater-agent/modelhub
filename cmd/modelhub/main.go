// ModelHub CLI — browse AI model pricing, capabilities, and benchmarks.
// Default output is JSON (pipe to jq/fzf). Use --table for a quick visual scan.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jelloeater-agent/modelhub/internal/cache"
	"github.com/jelloeater-agent/modelhub/internal/merge"
	"github.com/jelloeater-agent/modelhub/internal/model"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	if len(os.Args) < 2 {
		usage()
		return
	}

	cfg, store := initConfigAndStore()

	switch os.Args[1] {
	case "refresh":
		cmdRefresh(cfg, store)
	case "list":
		cmdList(cfg, store)
	case "show":
		cmdShow(cfg, store)
	case "stats":
		cmdStats(cfg, store)
	default:
		usage()
	}
}

func usage() {
	// ponytail: no fancy help generator — stdlib flag + simple text is enough
	fmt.Fprint(os.Stderr, `ModelHub — AI model pricing & capability browser

Usage:
  modelhub refresh           Fetch latest data from all sources
  modelhub list [--table]    List models (JSON default, --table for human)
  modelhub show <id>         Show a single model (JSON)
  modelhub stats [--json]    Aggregate statistics (JSON by default)

Config: ~/.modelhub/config.json or AA_API_KEY env var
Cache:  ~/.modelhub/cache.json

Examples:
  modelhub list | jq '.[] | select(.provider=="openai") | .name'
  modelhub list --table | grep gpt-4
  modelhub show openai/gpt-4o | jq .context_window
  modelhub stats
`)
}

// initConfigAndStore parses the global flags, loads config, and opens cache.
// ponytail: --config is the only global flag; everything else is per-subcommand.
func initConfigAndStore() (model.Config, *cache.Store) {
	cfgPath := ""
	if len(os.Args) > 2 {
		// --config before subcommand: modelhub --config /path config
		for i, a := range os.Args[1:] {
			if a == "--config" && i+1 < len(os.Args)-1 {
				cfgPath = os.Args[i+2]
				break
			}
		}
	}
	cfg := resolveConfig(cfgPath)
	store, err := cache.NewStore(cfg.CachePath)
	if err != nil {
		log.Fatalf("cache init: %v", err)
	}
	return cfg, store
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

// getCache loads cached data; exits with a message if none found.
func getCache(store *cache.Store) *model.Cache {
	c, err := store.Load()
	if err != nil {
		log.Fatalf("cache load: %v", err)
	}
	if c == nil {
		log.Fatal("no cached data — run 'modelhub refresh' first")
	}
	return c
}

// --- Subcommands ---

func cmdRefresh(cfg model.Config, store *cache.Store) {
	fmt.Fprintf(os.Stderr, "Fetching data from all sources...\n")
	result := merge.DoRefresh(cfg)

	c := &model.Cache{
		Models:    result.Models,
		FetchedAt: result.FetchedAt,
	}
	if err := store.Save(c); err != nil {
		log.Fatalf("cache save: %v", err)
	}

	stats := model.ComputeStats(result.Models, result.FetchedAt)
	fmt.Fprintf(os.Stderr, "✓ %d models from %d sources\n", stats.Total, len(stats.BySource))
	for src, count := range stats.BySource {
		fmt.Fprintf(os.Stderr, "  • %s: %d\n", src, count)
	}
	if len(result.Errors) > 0 {
		for src, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "⚠ %s: %s\n", src, err)
		}
	}
}

func cmdList(cfg model.Config, store *cache.Store) {
	// ponytail: minimal flag parsing per subcommand. --table only.
	// Use jq for any real querying.
	tableFlag := false
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--table":
			tableFlag = true
		case "--config":
			i++ // skip, already handled
		}
	}

	c := getCache(store)

	if tableFlag {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROVIDER\tMODEL\tMODE\tINPUT/1M\tOUTPUT/1M\tCTX\tSPEED(t/s)")
		for _, m := range c.Models {
			speed := fmt.Sprintf("%.0f", m.MedianTokensPerSecond)
			if m.MedianTokensPerSecond == 0 {
				speed = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t$%.2f\t%d\t%s\n",
				m.Provider, m.Name, m.Mode,
				m.InputPricePer1M, m.OutputPricePer1M,
				m.ContextWindow, speed)
		}
		w.Flush()
		return
	}

	// Default: JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c.Models); err != nil {
		log.Fatalf("json encode: %v", err)
	}
}

func cmdShow(cfg model.Config, store *cache.Store) {
	if len(os.Args) < 3 {
		log.Fatal("usage: modelhub show <id>  (e.g. 'modelhub show openai/gpt-4o')")
	}
	query := strings.ToLower(os.Args[2])

	c := getCache(store)
	for _, m := range c.Models {
		if strings.EqualFold(m.ID, query) || strings.EqualFold(m.Name, query) {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(m); err != nil {
				log.Fatalf("json encode: %v", err)
			}
			return
		}
	}
	log.Fatalf("model %q not found", query)
}

func cmdStats(cfg model.Config, store *cache.Store) {
	c := getCache(store)
	stats := model.ComputeStats(c.Models, c.FetchedAt)

	// Always print JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(stats); err != nil {
		log.Fatalf("json encode: %v", err)
	}

	// Also print a human-friendly summary to stderr
	fmt.Fprintf(os.Stderr, "%d total models\n", stats.Total)
	fmt.Fprintf(os.Stderr, "\nBy mode:\n")
	for mode, count := range stats.ByMode {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", mode, count)
	}
	fmt.Fprintf(os.Stderr, "\nBy provider (top 10):\n")
	type pc struct {
		name  string
		count int
	}
	sorted := make([]pc, 0, len(stats.ByProvider))
	for name, count := range stats.ByProvider {
		sorted = append(sorted, pc{name, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].name < sorted[j].name
	})
	n := 10
	if len(sorted) < n {
		n = len(sorted)
	}
	for _, p := range sorted[:n] {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", p.name, p.count)
	}
	if len(sorted) > n {
		fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(sorted)-n)
	}
	if stats.LastUpdate != "" {
		t, _ := time.Parse(time.RFC3339, stats.LastUpdate)
		fmt.Fprintf(os.Stderr, "\nLast updated: %s\n", t.Format(time.RFC3339))
	}
}
