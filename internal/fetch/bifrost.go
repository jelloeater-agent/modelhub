package fetch

import (
	"strings"

	"github.com/jelloeater-agent/modelhub/internal/model"
)

// RawBifrostEntry maps a single Bifrost datasheet entry.
type RawBifrostEntry struct {
	Provider        string  `json:"provider"`
	BaseModel       string  `json:"base_model"`
	Mode            string  `json:"mode"`
	InputCost       float64 `json:"input_cost_per_token"`
	OutputCost      float64 `json:"output_cost_per_token"`
	MaxInputTokens  int     `json:"max_input_tokens"`
	MaxOutputTokens int     `json:"max_output_tokens"`
	MaxTokens       int     `json:"max_tokens"`

	SupportsVision          bool `json:"supports_vision"`
	SupportsFunctionCalling bool `json:"supports_function_calling"`
	SupportsPromptCaching   bool `json:"supports_prompt_caching"`
	SupportsReasoning       bool `json:"supports_reasoning"`
	SupportsResponseSchema  bool `json:"supports_response_schema"`

	CacheReadInputCost float64 `json:"cache_read_input_token_cost"`
}

// FetchBifrost fetches and parses the Bifrost datasheet.
func FetchBifrost(url string) ([]model.Model, error) {
	raw := make(map[string]RawBifrostEntry)
	if err := FetchJSON(url, "", &raw); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []model.Model

	for key, entry := range raw {
		// Skip image generation with resolution prefixes like "1024-x-1024/..."
		// We only want models from the base key (without resolution prefix)
		if strings.Contains(key, "/") {
			parts := strings.SplitN(key, "/", 2)
			// If first part looks like a resolution, skip dimension-specific entries
			if isResolution(parts[0]) {
				continue
			}
		}

		// Determine provider and model name
		provider := entry.Provider
		if provider == "" {
			provider = inferProviderFromKey(key)
		}

		modelName := entry.BaseModel
		if modelName == "" {
			modelName = key
		}

		id := provider + "/" + modelName
		if seen[id] {
			continue
		}
		seen[id] = true

		m := model.Model{
			ID:                       id,
			Name:                     modelName,
			Provider:                 provider,
			Mode:                     entry.Mode,
			InputPricePer1M:          entry.InputCost * 1_000_000,
			OutputPricePer1M:         entry.OutputCost * 1_000_000,
			CacheReadPrice:           entry.CacheReadInputCost * 1_000_000,
			ContextWindow:            entry.MaxInputTokens,
			SupportsVision:           entry.SupportsVision,
			SupportsFunctionCalling:  entry.SupportsFunctionCalling,
			SupportsPromptCaching:    entry.SupportsPromptCaching,
			SupportsReasoning:        entry.SupportsReasoning,
			SupportsStructuredOutput: entry.SupportsResponseSchema,
			Sources:                  []string{"bifrost"},
		}
		if entry.MaxOutputTokens > 0 {
			m.MaxOutput = entry.MaxOutputTokens
		} else if entry.MaxTokens > 0 && entry.Mode == "chat" {
			m.MaxOutput = entry.MaxTokens
		}
		result = append(result, m)
	}
	return result, nil
}

func isResolution(s string) bool {
	return strings.Count(s, "-x-") > 0 || strings.Count(s, "x") == 2 && len(s) < 20
}

func inferProviderFromKey(key string) string {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	// Try to guess from the key prefix
	if strings.HasPrefix(key, "amazon.") || strings.HasPrefix(key, "us.") || strings.HasPrefix(key, "apac.") {
		return "bedrock"
	}
	if strings.HasPrefix(key, "ai21.") {
		return "bedrock"
	}
	return "unknown"
}

// ponytail: Resolution detection is heuristic (counts "x" patterns).
// If Bifrost adds more dimension formats, switch to a regex or known resolution list.
// Upgrade path: maintain a small set of known resolution prefixes.
