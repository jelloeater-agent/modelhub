package fetch

import (
	"github.com/user/modelhub/internal/model"
)

// AAResponse wraps the Artificial Analysis API response.
type AAResponse struct {
	Status int       `json:"status"`
	Data   []AAModel `json:"data"`
}

// AAModel represents a model from Artificial Analysis.
type AAModel struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	ReleaseDate string        `json:"release_date"`
	Creator     AACreator     `json:"model_creator"`
	Evaluations AAEvaluations `json:"evaluations"`
	Pricing     AAPricing     `json:"pricing"`
	Speed       float64       `json:"median_output_tokens_per_second"`
	TTFT        float64       `json:"median_time_to_first_token_seconds"`
}

// AACreator is the model creator/provider.
type AACreator struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// AAEvaluations contains benchmark scores.
type AAEvaluations struct {
	IntelligenceIndex float64 `json:"artificial_analysis_intelligence_index"`
	CodingIndex       float64 `json:"artificial_analysis_coding_index"`
	MathIndex         float64 `json:"artificial_analysis_math_index"`
	MMLUPro           float64 `json:"mmlu_pro"`
	GPQA              float64 `json:"gpqa"`
	LiveCodeBench     float64 `json:"livecodebench"`
	AIME25            float64 `json:"aime_25"`
}

// AAPricing contains pricing per 1M tokens.
type AAPricing struct {
	Input   float64 `json:"price_1m_input_tokens"`
	Output  float64 `json:"price_1m_output_tokens"`
	Blended float64 `json:"price_1m_blended_3_to_1"`
}

// FetchAA fetches and parses the Artificial Analysis API.
// If apiKey is empty, returns nil with no error (graceful degradation).
func FetchAA(url string, apiKey string) ([]model.Model, error) {
	if apiKey == "" {
		return nil, nil
	}

	var resp AAResponse
	if err := FetchJSON(url, apiKey, &resp); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []model.Model

	for _, m := range resp.Data {
		provider := m.Creator.Name
		if provider == "" {
			provider = "unknown"
		}
		// Build a normalized ID from provider + name
		id := provider + "/" + m.Slug
		if seen[id] {
			continue
		}
		seen[id] = true

		entry := model.Model{
			ID:                    id,
			Name:                  m.Name,
			Provider:              provider,
			Mode:                  "chat",
			IntelligenceIndex:     m.Evaluations.IntelligenceIndex,
			CodingIndex:           m.Evaluations.CodingIndex,
			MathIndex:             m.Evaluations.MathIndex,
			MMLUPro:               m.Evaluations.MMLUPro,
			GPQA:                  m.Evaluations.GPQA,
			LiveCodeBench:         m.Evaluations.LiveCodeBench,
			AIME25:                m.Evaluations.AIME25,
			InputPricePer1M:       m.Pricing.Input,
			OutputPricePer1M:      m.Pricing.Output,
			MedianTokensPerSecond: m.Speed,
			MedianTTFTSeconds:     m.TTFT,
			ReleaseDate:           m.ReleaseDate,
			Sources:               []string{"aa"},
		}
		result = append(result, entry)
	}
	return result, nil
}
