package token

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Model constants supported by the estimator.
const (
	ModelGPT4oMini      = "gpt-4o-mini"
	ModelGPT4o          = "gpt-4o"
	ModelClaude35Sonnet = "claude-3.5-sonnet"
	ModelClaude3Haiku   = "claude-3-haiku"
	ModelGemini15Flash  = "gemini-1.5-flash"
	ModelGemini15Pro    = "gemini-1.5-pro"
)

type modelProfile struct {
	MinCharsPerToken float64
	MaxCharsPerToken float64
	NetBias          float64
}

var modelProfiles = map[string]modelProfile{
	ModelGPT4oMini:      {MinCharsPerToken: 3.2, MaxCharsPerToken: 4.8, NetBias: 0.15},
	ModelGPT4o:          {MinCharsPerToken: 3.4, MaxCharsPerToken: 4.9, NetBias: 0.16},
	ModelClaude35Sonnet: {MinCharsPerToken: 3.0, MaxCharsPerToken: 4.6, NetBias: 0.18},
	ModelClaude3Haiku:   {MinCharsPerToken: 3.1, MaxCharsPerToken: 4.7, NetBias: 0.14},
	ModelGemini15Flash:  {MinCharsPerToken: 3.5, MaxCharsPerToken: 5.1, NetBias: 0.12},
	ModelGemini15Pro:    {MinCharsPerToken: 3.6, MaxCharsPerToken: 5.2, NetBias: 0.11},
}

var defaultModelOrder = []string{
	ModelGPT4oMini,
	ModelGPT4o,
	ModelClaude35Sonnet,
	ModelClaude3Haiku,
	ModelGemini15Flash,
	ModelGemini15Pro,
}

// ModelResult represents token estimations for a single model.
type ModelResult struct {
	Model       string `json:"model"`
	Min         int    `json:"min"`
	Max         int    `json:"max"`
	Average     int    `json:"average"`
	NetEstimate int    `json:"net_estimate"`
}

// SummaryResult represents aggregate estimations across all selected models.
type SummaryResult struct {
	Min         int `json:"min"`
	Max         int `json:"max"`
	Average     int `json:"average"`
	NetEstimate int `json:"net_estimate"`
	Spread      int `json:"spread"`
}

// ResultSet is the full estimation payload.
type ResultSet struct {
	Models    []string      `json:"models"`
	CharCount int           `json:"char_count"`
	WordCount int           `json:"word_count"`
	Results   []ModelResult `json:"results"`
	Summary   SummaryResult `json:"summary"`
}

// SupportedModels returns models in stable order.
func SupportedModels() []string {
	out := make([]string, len(defaultModelOrder))
	copy(out, defaultModelOrder)
	return out
}

// ParseModelSelection parses "--model" into selected model keys.
func ParseModelSelection(modelFlag string) ([]string, error) {
	normalized := strings.TrimSpace(strings.ToLower(modelFlag))
	if normalized == "" || normalized == "all" {
		return SupportedModels(), nil
	}

	parts := strings.Split(normalized, ",")
	seen := make(map[string]struct{}, len(parts))
	models := make([]string, 0, len(parts))

	for _, p := range parts {
		model := strings.TrimSpace(p)
		if model == "" {
			continue
		}
		if _, ok := modelProfiles[model]; !ok {
			return nil, fmt.Errorf("unsupported model: %s (supported: %s)", model, strings.Join(defaultModelOrder, ", "))
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		models = append(models, model)
	}

	if len(models) == 0 {
		return nil, errors.New("model selection cannot be empty")
	}

	return models, nil
}

// Estimate calculates per-model and aggregate token estimations.
func Estimate(content string, modelFlag string) (ResultSet, error) {
	input := strings.TrimSpace(content)
	if input == "" {
		return ResultSet{}, errors.New("input text is empty")
	}

	models, err := ParseModelSelection(modelFlag)
	if err != nil {
		return ResultSet{}, err
	}

	charCount := utf8.RuneCountInString(input)
	wordCount := len(strings.Fields(input))
	results := make([]ModelResult, 0, len(models))

	for _, model := range models {
		profile := modelProfiles[model]
		results = append(results, estimateForProfile(model, charCount, wordCount, profile))
	}

	return ResultSet{
		Models:    models,
		CharCount: charCount,
		WordCount: wordCount,
		Results:   results,
		Summary:   summarize(results),
	}, nil
}

func estimateForProfile(model string, charCount int, wordCount int, profile modelProfile) ModelResult {
	minByChars := int(math.Ceil(float64(charCount) / profile.MaxCharsPerToken))
	maxByChars := int(math.Ceil(float64(charCount) / profile.MinCharsPerToken))

	// Word multiplier keeps estimates reasonable when punctuation/whitespace density is high.
	minByWords := int(math.Ceil(float64(wordCount) * 0.70))
	maxByWords := int(math.Ceil(float64(wordCount) * 1.60))

	minTokens := max(1, max(minByChars, minByWords))
	maxTokens := max(minTokens, max(maxByChars, maxByWords))
	average := int(math.Round(float64(minTokens+maxTokens) / 2.0))
	net := int(math.Round(float64(average)*(1.0-profile.NetBias) + float64(maxTokens)*profile.NetBias))
	net = max(minTokens, min(net, maxTokens))

	return ModelResult{
		Model:       model,
		Min:         minTokens,
		Max:         maxTokens,
		Average:     average,
		NetEstimate: net,
	}
}

func summarize(results []ModelResult) SummaryResult {
	if len(results) == 0 {
		return SummaryResult{}
	}

	globalMin := results[0].Min
	globalMax := results[0].Max
	totalAverage := 0
	totalNet := 0

	for _, r := range results {
		globalMin = min(globalMin, r.Min)
		globalMax = max(globalMax, r.Max)
		totalAverage += r.Average
		totalNet += r.NetEstimate
	}

	avg := int(math.Round(float64(totalAverage) / float64(len(results))))
	net := int(math.Round(float64(totalNet) / float64(len(results))))

	return SummaryResult{
		Min:         globalMin,
		Max:         globalMax,
		Average:     avg,
		NetEstimate: net,
		Spread:      globalMax - globalMin,
	}
}
