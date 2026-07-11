// ModelHub CLI — browse AI model pricing, capabilities, and benchmarks.
// Default output is JSON (pipe to jq/fzf). Use --table for a quick visual scan.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jelloeater-agent/modelhub/internal/cache"
	"github.com/jelloeater-agent/modelhub/internal/merge"
	"github.com/jelloeater-agent/modelhub/internal/model"
)

// version is set at build time via -ldflags, falls back to the tag below.
// ponytail: no separate version file, no build script — just update this on tag.
var version = "v0.2.0" // bump on each release

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	if len(os.Args) < 2 {
		usage()
		return
	}

	// ponytail: completion and version need no config — short-circuit before initConfigAndStore
	if os.Args[1] == "completion" {
		cmdCompletion()
		return
	}
	if os.Args[1] == "version" || os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	cfg, store := initConfigAndStore()

	switch os.Args[1] {
	case "refresh":
		cmdRefresh(cfg, store)
	case "update":
		cmdUpdate()
	case "list":
		cmdList(cfg, store)
	case "show":
		cmdShow(cfg, store)
	case "stats":
		cmdStats(cfg, store)
	case "search":
		cmdSearch(cfg, store)
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
  modelhub search            Interactive fuzzy search (fzf) with clipboard copy
  modelhub stats             Aggregate statistics (JSON)
  modelhub version           Print version
  modelhub update            Self-update via go install
  modelhub completion <sh>   Generate shell completion (bash|zsh|fish)

Config: $XDG_CONFIG_HOME/modelhub/config.json or AA_API_KEY env var
Cache:  $XDG_CACHE_HOME/modelhub/cache.json
        (falls back to ~/.modelhub/ if XDG vars are unset)

Examples:
  eval "$(modelhub completion bash)"   # bash
  eval "$(modelhub completion zsh)"    # zsh
  modelhub completion fish | source    # fish
  modelhub search                      # interactive fzf + copy
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
		cfgPath = model.ConfigPath()
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

// cmdUpdate self-updates the binary via go install.
// ponytail: no downloader, no checksum verification — just delegates to go install.
// Ceiling: requires Go in PATH. Upgrade path: pull prebuilt binaries from GitHub releases.
func cmdUpdate() {
	if _, err := exec.LookPath("go"); err != nil {
		log.Fatal("update requires Go — install it from https://go.dev/dl")
	}
	fmt.Fprint(os.Stderr, "Updating modelhub...\n")
	cmd := exec.Command("go", "install", "github.com/jelloeater-agent/modelhub/cmd/modelhub@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("update failed: %v", err)
	}
	fmt.Fprint(os.Stderr, "✓ updated\n")
}

// ponytail: static search via fzf. No TUI lib, no fancy UI — just pipe TSV to fzf.
// Ceiling: fzf must be installed. Upgrade path: embed a basic TUI as an alternative.
func cmdSearch(cfg model.Config, store *cache.Store) {
	if _, err := exec.LookPath("fzf"); err != nil {
		log.Fatal("modelhub search requires fzf — install it from https://github.com/junegunn/fzf")
	}

	c := getCache(store)

	// Build TSV: slug first (hidden), then fixed-width display columns
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "PROVIDER           MODEL                                   MODE              INPUT/1M  OUTPUT/1M   CTX    SPEED")
	for _, m := range c.Models {
		speed := fmt.Sprintf("%.0f", m.MedianTokensPerSecond)
		if m.MedianTokensPerSecond == 0 {
			speed = "-"
		}
		fmt.Fprintf(&buf, "%s\t%-18s %-38s %-16s $%-7.2f $%-8.2f %-5d %s\n",
			m.ID, m.Provider, m.Name, m.Mode,
			m.InputPricePer1M, m.OutputPricePer1M,
			m.ContextWindow, speed)
	}

	cmd := exec.Command("fzf",
		"--header", "Enter → copy provider/slug to clipboard",
		"--header-lines=1",
		"--delimiter=\t",
		"--with-nth=2..",
	)
	cmd.Stdin = &buf
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 130 || exitErr.ExitCode() == 1) {
			return // user cancelled or no match
		}
		log.Fatalf("fzf: %v", err)
	}

	slug := strings.TrimSpace(string(out))
	// slug includes trailing tab+rest, take only first field
	if idx := strings.IndexByte(slug, '\t'); idx >= 0 {
		slug = slug[:idx]
	}
	if slug == "" {
		return
	}

	copyToClipboard(slug)
	fmt.Fprintf(os.Stderr, "✓ %s\n", slug)
}

func copyToClipboard(text string) {
	for _, args := range [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "-ib"},
		{"wl-copy"},
		{"pbcopy"},
	} {
		if _, err := exec.LookPath(args[0]); err == nil {
			c := exec.Command(args[0], args[1:]...)
			c.Stdin = strings.NewReader(text)
			c.Run()
			return
		}
	}
	// ponytail: no clipboard tool found, just print to stdout for manual piping
	fmt.Print(text)
}

func cmdCompletion() {
	if len(os.Args) < 3 {
		log.Fatal("usage: modelhub completion bash|zsh|fish")
	}
	// ponytail: static completion scripts — no cobra, no codegen.
	// If the CLI grows beyond 4 subcommands, switch to dynamic completion.
	switch os.Args[2] {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		log.Fatalf("unknown shell %q (use: bash, zsh, fish)", os.Args[2])
	}
}

var bashCompletion = `_modelhub() {
    local cur prev words cword
    _init_completion || return

    local subcmds="refresh list show stats search version update completion"
    local list_flags="--table"
    local global_flags="--config"

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=($(compgen -W "$subcmds $global_flags" -- "$cur"))
        return
    fi

    case ${words[1]} in
        list)
            COMPREPLY=($(compgen -W "$list_flags" -- "$cur"))
            ;;
        show)
            COMPREPLY=()
            ;;
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            ;;
    esac
} &&
complete -F _modelhub modelhub
`

var zshCompletion = `#compdef modelhub

_modelhub() {
  local -a subcmds
  subcmds=(
    'refresh:Fetch latest data from all sources'
    'list:List models (--table for human-readable)'
    'show:Show a single model by ID'
    'search:Interactive fuzzy search with clipboard copy'
    'stats:Aggregate statistics'
    'version:Print version'
    'update:Self-update via go install'
    'completion:Generate shell completion script'
  )

  local -a list_opts
  list_opts=('--table[Tabular output]')

  local -a global_opts
  global_opts=('--config[Path to config file]')

  _arguments \
    $global_opts \
    "1: :{_describe 'command' subcmds}" \
    "*::args:->args"

  case $state in
    args)
      case $line[1] in
        list)  _arguments $list_opts ;;
        show)  _arguments ':model-id:' ;;
        completion) _arguments ':shell:(bash zsh fish)' ;;
      esac
      ;;
  esac
}

compdef _modelhub modelhub
`

var fishCompletion = `function __fish_modelhub_needs_command
    set cmd (commandline -opc)
    if test (count $cmd) -eq 1
        return 0
    end
    return 1
end

function __fish_modelhub_using_command
    set cmd (commandline -opc)
    if test (count $cmd) -gt 1
        set -l subcmd $cmd[2]
        for arg in $argv
            if test "$subcmd" = "$arg"
                return 0
            end
        end
    end
    return 1
end

complete -c modelhub -f -n '__fish_modelhub_needs_command' -a refresh -d 'Fetch latest data from all sources'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a list -d 'List models (JSON default, --table for human)'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a show -d 'Show a single model by ID'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a search -d 'Interactive fuzzy search with clipboard copy'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a stats -d 'Aggregate statistics'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a version -d 'Print version'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a update -d 'Self-update via go install'
complete -c modelhub -f -n '__fish_modelhub_needs_command' -a completion -d 'Generate shell completion script'

complete -c modelhub -f -n '__fish_modelhub_needs_command' -l config -d 'Path to config file'

complete -c modelhub -f -n '__fish_modelhub_using_command list' -l table -d 'Tabular output'
complete -c modelhub -f -n '__fish_modelhub_using_command completion' -a 'bash zsh fish'
`
