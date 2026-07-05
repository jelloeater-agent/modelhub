package tui

import (
	"fmt"
	"strings"
)

func (m *teaModel) detailView() string {
	if m.selectedModel == nil {
		m.state = viewList
		return ""
	}

	mdl := *m.selectedModel
	var b strings.Builder

	srcStr := fmt.Sprintf("%d sources", len(mdl.Sources))
	header := fmt.Sprintf("  Back    %s    %s", mdl.Name, srcStr)
	b.WriteString(HeaderStyle.Width(m.width).Render(header))
	b.WriteString("\n\n")

	// Source badges
	var badges []string
	for _, src := range mdl.Sources {
		switch src {
		case "bifrost":
			badges = append(badges, fmt.Sprintf("Bifrost"))
		case "modelsdev":
			badges = append(badges, "models.dev")
		case "aa":
			badges = append(badges, "Artificial Analysis")
		}
	}
	b.WriteString(strings.Join(badges, "  "))
	b.WriteString("\n\n")

	// Capabilities
	var caps []string
	caps = addCap(caps, "Vision", mdl.SupportsVision)
	caps = addCap(caps, "Function Calling", mdl.SupportsFunctionCalling)
	caps = addCap(caps, "Prompt Caching", mdl.SupportsPromptCaching)
	caps = addCap(caps, "Reasoning", mdl.SupportsReasoning)
	caps = addCap(caps, "Structured Output", mdl.SupportsStructuredOutput)
	caps = addCap(caps, "Open Weights", mdl.OpenWeights)

	content := DetailStyle.Width(m.width - 4).Render(
		fmt.Sprintf("Provider:  %s\n", mdl.Provider) +
			fmt.Sprintf("Model:     %s\n", mdl.Name) +
			fmt.Sprintf("Family:    %s\n", mdl.Family) +
			fmt.Sprintf("Mode:      %s\n", mdl.Mode) +
			fmt.Sprintf("Released:  %s\n\n", mdl.ReleaseDate) +

			"── Pricing ($ per 1M tokens) ──\n" +
			fmt.Sprintf("  Input:      %s\n", formatPrice(mdl.InputPricePer1M)) +
			fmt.Sprintf("  Output:     %s\n", formatPrice(mdl.OutputPricePer1M)) +
			fmt.Sprintf("  Cache Read: %s\n\n", formatPrice(mdl.CacheReadPrice)) +

			"── Limits ──\n" +
			fmt.Sprintf("  Context Window: %s\n", formatInt(mdl.ContextWindow)) +
			fmt.Sprintf("  Max Output:     %s\n\n", formatInt(mdl.MaxOutput)) +

			"── Capabilities ──\n" +
			fmt.Sprintf("  %s\n\n", strings.Join(caps, "\n  ")) +

			"── Performance (Artificial Analysis) ──\n" +
			fmt.Sprintf("  Intelligence Index:  %.1f\n", mdl.IntelligenceIndex) +
			fmt.Sprintf("  Coding Index:        %.1f\n", mdl.CodingIndex) +
			fmt.Sprintf("  Math Index:          %.1f\n", mdl.MathIndex) +
			fmt.Sprintf("  Speed:               %.0f tok/s\n", mdl.MedianTokensPerSecond) +
			fmt.Sprintf("  TTFT:                %.1fs\n\n", mdl.MedianTTFTSeconds) +

			"── Benchmarks ──\n" +
			fmt.Sprintf("  MMLU-Pro:      %.3f\n", mdl.MMLUPro) +
			fmt.Sprintf("  GPQA:          %.3f\n", mdl.GPQA) +
			fmt.Sprintf("  LiveCodeBench: %.3f\n", mdl.LiveCodeBench) +
			fmt.Sprintf("  AIME 25:       %.3f\n", mdl.AIME25),
	)
	b.WriteString(content)
	b.WriteString("\n")

	if mdl.Description != "" {
		b.WriteString(ItalicStyle.Render(mdl.Description))
	}

	status := "  arrows scroll  |  Esc/q back  |  ? help"
	b.WriteString("\n")
	b.WriteString(StatusBarStyle.Width(m.width).Render(status))

	return AppStyle.Render(b.String())
}

func addCap(caps []string, name string, enabled bool) []string {
	mark := "x"
	if enabled {
		mark = "v"
	}
	return append(caps, fmt.Sprintf("%s  %s", mark, name))
}
