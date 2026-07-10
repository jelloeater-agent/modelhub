package fetch

import (
	"github.com/jelloeater-agent/modelhub/internal/model"
)

// ModelsDevProvider represents a provider entry in models.dev/api.json.
type ModelsDevProvider struct {
	ID     string                    `json:"id"`
	Name   string                    `json:"name"`
	API    string                    `json:"api"`
	Doc    string                    `json:"doc"`
	Env    []string                  `json:"env"`
	NPM    string                    `json:"npm"`
	Models map[string]ModelsDevModel `json:"models"`
}

// ModelsDevModel represents a model within a provider.
type ModelsDevModel struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Family           string              `json:"family"`
	Reasoning        bool                `json:"reasoning"`
	ToolCall         bool                `json:"tool_call"`
	StructuredOutput bool                `json:"structured_output"`
	Temperature      any                 `json:"temperature"` // bool or absent
	OpenWeights      bool                `json:"open_weights"`
	ReleaseDate      string              `json:"release_date"`
	Limit            ModelsDevLimit      `json:"limit"`
	Cost             ModelsDevCost       `json:"cost"`
	Modalities       ModelsDevModalities `json:"modalities"`
}

// ModelsDevLimit represents context/output limits.
type ModelsDevLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

// ModelsDevCost represents pricing.
type ModelsDevCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

// ModelsDevModalities represents input/output modality lists.
type ModelsDevModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// FetchModelsDev fetches and parses models.dev/api.json.
func FetchModelsDev(url string) ([]model.Model, error) {
	raw := make(map[string]ModelsDevProvider)
	if err := FetchJSON(url, "", &raw); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []model.Model

	for providerKey, provider := range raw {
		for modelID, m := range provider.Models {
			// Normalize ID: if model already starts with provider/, use as-is
			id := m.ID
			if !stringsContainsRune(id, '/') {
				id = providerKey + "/" + modelID
			}
			if seen[id] {
				continue
			}
			seen[id] = true

			hasVision := false
			for _, input := range m.Modalities.Input {
				if input == "image" || input == "video" || input == "pdf" {
					hasVision = true
					break
				}
			}

			entry := model.Model{
				ID:                       id,
				Name:                     m.Name,
				Provider:                 provider.Name,
				Mode:                     "chat", // models.dev is primarily chat models
				InputPricePer1M:          m.Cost.Input,
				OutputPricePer1M:         m.Cost.Output,
				CacheReadPrice:           m.Cost.CacheRead,
				ContextWindow:            m.Limit.Context,
				MaxOutput:                m.Limit.Output,
				SupportsVision:           hasVision,
				SupportsFunctionCalling:  m.ToolCall,
				SupportsReasoning:        m.Reasoning,
				SupportsStructuredOutput: m.StructuredOutput,
				OpenWeights:              m.OpenWeights,
				Family:                   m.Family,
				ReleaseDate:              m.ReleaseDate,
				Description:              m.Description,
				Sources:                  []string{"modelsdev"},
			}
			result = append(result, entry)
		}
	}
	return result, nil
}

func stringsContainsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// ponytail: Mode is always "chat" for models.dev since it's a router registry.
// If models.dev adds non-chat modes, parse from the model ID or add a mode field.}
