package merge

import (
	"sort"
	"strings"
	"time"

	"github.com/user/modelhub/internal/fetch"
	"github.com/user/modelhub/internal/model"
)

const (
	SourceBifrost   = "bifrost"
	SourceModelsDev = "modelsdev"
	SourceAA        = "aa"
)

// Merge combines models from sources. Higher-priority sources override
// conflicting fields; lower-priority sources fill in missing fields.
// Models matched by normalized ID (lowercase, version suffixes stripped).
func Merge(allModels ...[]model.Model) []model.Model {
	// Priority order (lower index = higher priority)
	sourcePriority := []string{SourceBifrost, SourceModelsDev, SourceAA}
	priMap := map[string]int{SourceBifrost: 0, SourceModelsDev: 1, SourceAA: 2}

	// Group by source
	bySource := make(map[string][]model.Model)
	for _, models := range allModels {
		for _, m := range models {
			src := m.Sources[0]
			bySource[src] = append(bySource[src], m)
		}
	}

	index := make(map[string]model.Model) // normalized key -> model
	srcMap := make(map[string][]string)   // normalized key -> sources

	// Apply sources in priority order
	for _, src := range sourcePriority {
		for _, m := range bySource[src] {
			key := NormalizeID(m.ID)
			existing, exists := index[key]

			if !exists {
				// Store with source tracking
				m.Sources = []string{src}
				index[key] = m
				srcMap[key] = []string{src}
				continue
			}

			// Merge: higher priority (already in index) keeps its non-zero fields.
			// Lower priority fills gaps.
			merged := existing
			incomingPri := priMap[src]
			existingPri := 999
			for _, s := range srcMap[key] {
				if p, ok := priMap[s]; ok && p < existingPri {
					existingPri = p
				}
			}

			if incomingPri < existingPri {
				// Incoming has higher priority: override all non-zero fields
				if m.InputPricePer1M != 0 {
					merged.InputPricePer1M = m.InputPricePer1M
				}
				if m.OutputPricePer1M != 0 {
					merged.OutputPricePer1M = m.OutputPricePer1M
				}
				if m.CacheReadPrice != 0 {
					merged.CacheReadPrice = m.CacheReadPrice
				}
				if m.ContextWindow > 0 {
					merged.ContextWindow = m.ContextWindow
				}
				if m.MaxOutput > 0 {
					merged.MaxOutput = m.MaxOutput
				}
				// Bools: incoming=true overrides
				if m.SupportsVision {
					merged.SupportsVision = true
				}
				if m.SupportsFunctionCalling {
					merged.SupportsFunctionCalling = true
				}
				if m.SupportsPromptCaching {
					merged.SupportsPromptCaching = true
				}
				if m.SupportsReasoning {
					merged.SupportsReasoning = true
				}
				if m.SupportsStructuredOutput {
					merged.SupportsStructuredOutput = true
				}
				if m.OpenWeights {
					merged.OpenWeights = true
				}
				if m.IntelligenceIndex != 0 {
					merged.IntelligenceIndex = m.IntelligenceIndex
				}
				if m.CodingIndex != 0 {
					merged.CodingIndex = m.CodingIndex
				}
				if m.MathIndex != 0 {
					merged.MathIndex = m.MathIndex
				}
				if m.MedianTokensPerSecond != 0 {
					merged.MedianTokensPerSecond = m.MedianTokensPerSecond
				}
				if m.MedianTTFTSeconds != 0 {
					merged.MedianTTFTSeconds = m.MedianTTFTSeconds
				}
				if m.MMLUPro != 0 {
					merged.MMLUPro = m.MMLUPro
				}
				if m.GPQA != 0 {
					merged.GPQA = m.GPQA
				}
				if m.LiveCodeBench != 0 {
					merged.LiveCodeBench = m.LiveCodeBench
				}
				if m.AIME25 != 0 {
					merged.AIME25 = m.AIME25
				}
				if m.Mode != "" && merged.Mode == "" {
					merged.Mode = m.Mode
				}
				if m.Family != "" && merged.Family == "" {
					merged.Family = m.Family
				}
				if m.ReleaseDate != "" && merged.ReleaseDate == "" {
					merged.ReleaseDate = m.ReleaseDate
				}
				if m.Description != "" && merged.Description == "" {
					merged.Description = m.Description
				}
			} else if incomingPri > existingPri {
				// Incoming has lower priority: only fill zero/default fields
				if merged.InputPricePer1M == 0 && m.InputPricePer1M != 0 {
					merged.InputPricePer1M = m.InputPricePer1M
				}
				if merged.OutputPricePer1M == 0 && m.OutputPricePer1M != 0 {
					merged.OutputPricePer1M = m.OutputPricePer1M
				}
				if merged.CacheReadPrice == 0 && m.CacheReadPrice != 0 {
					merged.CacheReadPrice = m.CacheReadPrice
				}
				if merged.ContextWindow == 0 && m.ContextWindow > 0 {
					merged.ContextWindow = m.ContextWindow
				}
				if merged.MaxOutput == 0 && m.MaxOutput > 0 {
					merged.MaxOutput = m.MaxOutput
				}
				if !merged.SupportsVision && m.SupportsVision {
					merged.SupportsVision = true
				}
				if !merged.SupportsFunctionCalling && m.SupportsFunctionCalling {
					merged.SupportsFunctionCalling = true
				}
				if !merged.SupportsPromptCaching && m.SupportsPromptCaching {
					merged.SupportsPromptCaching = true
				}
				if !merged.SupportsReasoning && m.SupportsReasoning {
					merged.SupportsReasoning = true
				}
				if !merged.SupportsStructuredOutput && m.SupportsStructuredOutput {
					merged.SupportsStructuredOutput = true
				}
				if !merged.OpenWeights && m.OpenWeights {
					merged.OpenWeights = true
				}
				if merged.IntelligenceIndex == 0 && m.IntelligenceIndex != 0 {
					merged.IntelligenceIndex = m.IntelligenceIndex
				}
				if merged.CodingIndex == 0 && m.CodingIndex != 0 {
					merged.CodingIndex = m.CodingIndex
				}
				if merged.MathIndex == 0 && m.MathIndex != 0 {
					merged.MathIndex = m.MathIndex
				}
				if merged.MedianTokensPerSecond == 0 && m.MedianTokensPerSecond != 0 {
					merged.MedianTokensPerSecond = m.MedianTokensPerSecond
				}
				if merged.MedianTTFTSeconds == 0 && m.MedianTTFTSeconds != 0 {
					merged.MedianTTFTSeconds = m.MedianTTFTSeconds
				}
				if merged.MMLUPro == 0 && m.MMLUPro != 0 {
					merged.MMLUPro = m.MMLUPro
				}
				if merged.GPQA == 0 && m.GPQA != 0 {
					merged.GPQA = m.GPQA
				}
				if merged.LiveCodeBench == 0 && m.LiveCodeBench != 0 {
					merged.LiveCodeBench = m.LiveCodeBench
				}
				if merged.AIME25 == 0 && m.AIME25 != 0 {
					merged.AIME25 = m.AIME25
				}
				if merged.Mode == "" && m.Mode != "" {
					merged.Mode = m.Mode
				}
				if merged.Family == "" && m.Family != "" {
					merged.Family = m.Family
				}
				if merged.ReleaseDate == "" && m.ReleaseDate != "" {
					merged.ReleaseDate = m.ReleaseDate
				}
				if merged.Description == "" && m.Description != "" {
					merged.Description = m.Description
				}
			} else {
				// Same priority: merge all non-zero
				mergeNonZero(&merged, m)
			}

			// Track source
			addSource := true
			for _, s := range srcMap[key] {
				if s == src {
					addSource = false
					break
				}
			}
			if addSource {
				srcMap[key] = append(srcMap[key], src)
				merged.Sources = append(merged.Sources, src)
				sort.Strings(merged.Sources)
			}
			index[key] = merged
		}
	}

	result := make([]model.Model, 0, len(index))
	for _, m := range index {
		result = append(result, m)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Provider != result[j].Provider {
			return result[i].Provider < result[j].Provider
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func mergeNonZero(dst *model.Model, src model.Model) {
	if src.InputPricePer1M != 0 {
		dst.InputPricePer1M = src.InputPricePer1M
	}
	if src.OutputPricePer1M != 0 {
		dst.OutputPricePer1M = src.OutputPricePer1M
	}
	if src.CacheReadPrice != 0 {
		dst.CacheReadPrice = src.CacheReadPrice
	}
	if src.ContextWindow > 0 {
		dst.ContextWindow = src.ContextWindow
	}
	if src.MaxOutput > 0 {
		dst.MaxOutput = src.MaxOutput
	}
	if src.SupportsVision {
		dst.SupportsVision = true
	}
	if src.SupportsFunctionCalling {
		dst.SupportsFunctionCalling = true
	}
	if src.SupportsPromptCaching {
		dst.SupportsPromptCaching = true
	}
	if src.SupportsReasoning {
		dst.SupportsReasoning = true
	}
	if src.SupportsStructuredOutput {
		dst.SupportsStructuredOutput = true
	}
	if src.OpenWeights {
		dst.OpenWeights = true
	}
	if src.IntelligenceIndex != 0 {
		dst.IntelligenceIndex = src.IntelligenceIndex
	}
	if src.CodingIndex != 0 {
		dst.CodingIndex = src.CodingIndex
	}
	if src.MathIndex != 0 {
		dst.MathIndex = src.MathIndex
	}
	if src.MedianTokensPerSecond != 0 {
		dst.MedianTokensPerSecond = src.MedianTokensPerSecond
	}
	if src.MedianTTFTSeconds != 0 {
		dst.MedianTTFTSeconds = src.MedianTTFTSeconds
	}
	if src.MMLUPro != 0 {
		dst.MMLUPro = src.MMLUPro
	}
	if src.GPQA != 0 {
		dst.GPQA = src.GPQA
	}
	if src.LiveCodeBench != 0 {
		dst.LiveCodeBench = src.LiveCodeBench
	}
	if src.AIME25 != 0 {
		dst.AIME25 = src.AIME25
	}
	if src.Mode != "" {
		dst.Mode = src.Mode
	}
	if src.Family != "" {
		dst.Family = src.Family
	}
	if src.ReleaseDate != "" {
		dst.ReleaseDate = src.ReleaseDate
	}
	if src.Description != "" {
		dst.Description = src.Description
	}
}

// NormalizeID creates a stable matching key, stripping version/date info.
func NormalizeID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))

	// Strip :version suffixes (:0, :v1, :latest)
	if idx := strings.LastIndex(id, ":"); idx > 0 {
		suffix := id[idx+1:]
		if isShortVersion(suffix) {
			id = id[:idx]
		}
	}

	// Strip trailing /vN
	if len(id) > 3 && id[len(id)-2] == '/' {
		last := id[len(id)-1]
		if last >= '0' && last <= '9' {
			id = id[:len(id)-3]
		}
	}

	// Strip trailing date/version segments one by one
	parts := strings.Split(id, "-")
	for len(parts) > 1 && isTrailingVersion(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}

	return strings.Join(parts, "-")
}

func isShortVersion(s string) bool {
	// :0, :v1, :latest, :snapshot, etc.
	if len(s) <= 2 {
		allDigit := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			return true
		}
	}
	return s == "latest" || s == "snapshot"
}

func isTrailingVersion(s string) bool {
	// 4+ digits (year, YYYYMMDD)
	if len(s) >= 4 {
		allDigits := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	// 2 digits (month or day in a YYYY-MM-DD pattern)
	if len(s) == 2 {
		allDigits := true
		for _, c := range s {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	return s == "preview" || s == "latest" || s == "snapshot" || s == "alpha" || s == "beta" || s == "rc1" || s == "rc2"
}

// FilterParams holds filter criteria.
type FilterParams struct {
	Search    string
	Providers []string
	Modes     []string
	Sources   []string
	MinPrice  float64
	MaxPrice  float64
	MinCtx    int
	CapFlags  CapabilityFlags
}

type CapabilityFlags struct {
	Vision          bool
	FunctionCalling bool
	Reasoning       bool
}

func ApplyFilter(models []model.Model, fp FilterParams, sortBy string, sortAsc bool) []model.Model {
	var filtered []model.Model
	for _, m := range models {
		if !matchesFilter(m, fp) {
			continue
		}
		filtered = append(filtered, m)
	}
	sort.Slice(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		cmp := compareField(a, b, sortBy)
		if !sortAsc {
			cmp = -cmp
		}
		if cmp == 0 {
			if a.Provider != b.Provider {
				return a.Provider < b.Provider
			}
			return a.Name < b.Name
		}
		return cmp < 0
	})
	return filtered
}

func matchesFilter(m model.Model, fp FilterParams) bool {
	if fp.Search != "" {
		q := strings.ToLower(fp.Search)
		if !strings.Contains(strings.ToLower(m.Name), q) &&
			!strings.Contains(strings.ToLower(m.Provider), q) &&
			!strings.Contains(strings.ToLower(m.Description), q) {
			return false
		}
	}
	if len(fp.Providers) > 0 {
		ok := false
		for _, p := range fp.Providers {
			if strings.EqualFold(m.Provider, p) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(fp.Modes) > 0 {
		ok := false
		for _, mode := range fp.Modes {
			if strings.EqualFold(m.Mode, mode) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(fp.Sources) > 0 {
		ok := false
		for _, want := range fp.Sources {
			for _, have := range m.Sources {
				if have == want {
					ok = true
					break
				}
			}
			if ok {
				break
			}
		}
		if !ok {
			return false
		}
	}
	if fp.MinPrice > 0 && m.InputPricePer1M < fp.MinPrice {
		return false
	}
	if fp.MaxPrice > 0 && m.InputPricePer1M > fp.MaxPrice {
		return false
	}
	if fp.MinCtx > 0 && m.ContextWindow < fp.MinCtx {
		return false
	}
	if fp.CapFlags.Vision && !m.SupportsVision {
		return false
	}
	if fp.CapFlags.FunctionCalling && !m.SupportsFunctionCalling {
		return false
	}
	if fp.CapFlags.Reasoning && !m.SupportsReasoning {
		return false
	}
	return true
}

func compareField(a, b model.Model, field string) float64 {
	switch field {
	case "name":
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
	case "provider":
		if a.Provider < b.Provider {
			return -1
		} else if a.Provider > b.Provider {
			return 1
		}
	case "input_price":
		return a.InputPricePer1M - b.InputPricePer1M
	case "output_price":
		return a.OutputPricePer1M - b.OutputPricePer1M
	case "context":
		return float64(a.ContextWindow - b.ContextWindow)
	case "speed":
		return a.MedianTokensPerSecond - b.MedianTokensPerSecond
	case "intelligence":
		return a.IntelligenceIndex - b.IntelligenceIndex
	case "coding":
		return a.CodingIndex - b.CodingIndex
	case "sources":
		return float64(len(a.Sources) - len(b.Sources))
	}
	return 0
}

// RefreshResult wraps the result of a fetch cycle.
type RefreshResult struct {
	Models    []model.Model
	FetchedAt map[string]string
	Errors    map[string]string
}

// DoRefresh fetches all sources and merges them.
func DoRefresh(cfg model.Config) *RefreshResult {
	result := &RefreshResult{
		FetchedAt: make(map[string]string),
		Errors:    make(map[string]string),
	}
	now := time.Now().UTC().Format(time.RFC3339)

	type fetchRes struct {
		source string
		models []model.Model
		err    error
	}
	ch := make(chan fetchRes, 3)

	go func() {
		m, err := fetch.FetchBifrost(cfg.BifrostURL)
		ch <- fetchRes{SourceBifrost, m, err}
	}()
	go func() {
		m, err := fetch.FetchModelsDev(cfg.ModelsDevURL)
		ch <- fetchRes{SourceModelsDev, m, err}
	}()
	go func() {
		m, err := fetch.FetchAA(cfg.AAURL, cfg.AAAPIKey)
		ch <- fetchRes{SourceAA, m, err}
	}()

	var allModels [][]model.Model
	for range 3 {
		r := <-ch
		if r.err != nil {
			result.Errors[r.source] = r.err.Error()
			continue
		}
		if r.models != nil {
			allModels = append(allModels, r.models)
			result.FetchedAt[r.source] = now
		}
	}
	close(ch)

	result.Models = Merge(allModels...)
	result.FetchedAt["_merged"] = now
	return result
}
