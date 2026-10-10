// Package token estimates token counts from text using character and word
// heuristics. It performs no tokenization and needs no network access.
package token

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

// Counts holds the raw text measurements an estimate is derived from.
type Counts struct {
	Chars int `json:"char_count"`
	Words int `json:"word_count"`
}

// Profile describes how many characters make up a token for a model family.
type Profile struct {
	MinCharsPerToken float64
	MaxCharsPerToken float64
	// NetBias pulls the net estimate from the average toward the maximum.
	NetBias float64
}

// Global is the model-agnostic profile used when no model is selected.
var Global = Profile{MinCharsPerToken: 3.2, MaxCharsPerToken: 4.8, NetBias: 0.15}

var providerProfiles = map[string]Profile{
	"openai":    {MinCharsPerToken: 3.4, MaxCharsPerToken: 4.9, NetBias: 0.16},
	"anthropic": {MinCharsPerToken: 3.0, MaxCharsPerToken: 4.6, NetBias: 0.18},
	"gemini":    {MinCharsPerToken: 3.5, MaxCharsPerToken: 5.1, NetBias: 0.12},
}

// ProfileFor returns the profile for a provider, or Global when unknown.
func ProfileFor(provider string) Profile {
	if p, ok := providerProfiles[provider]; ok {
		return p
	}
	return Global
}

// Estimate is a token range for one profile.
type Estimate struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Average int `json:"average"`
	Net     int `json:"net_estimate"`
}

// Cost is a price range in USD.
type Cost struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Net float64 `json:"net"`
}

// Summary aggregates several estimates.
type Summary struct {
	Estimate
	Spread int `json:"spread"`
}

// Count measures text. It errors on blank input.
func Count(text string) (Counts, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return Counts{}, errors.New("input text is empty")
	}
	return Counts{Chars: utf8.RuneCountInString(trimmed), Words: len(strings.Fields(trimmed))}, nil
}

// Estimate derives a token range from counts.
func (p Profile) Estimate(c Counts) Estimate {
	minByChars := int(math.Ceil(float64(c.Chars) / p.MaxCharsPerToken))
	maxByChars := int(math.Ceil(float64(c.Chars) / p.MinCharsPerToken))

	// Word multipliers keep estimates sane when punctuation density is high.
	minByWords := int(math.Ceil(float64(c.Words) * 0.70))
	maxByWords := int(math.Ceil(float64(c.Words) * 1.60))

	lo := max(1, minByChars, minByWords)
	hi := max(lo, maxByChars, maxByWords)
	avg := int(math.Round(float64(lo+hi) / 2))
	net := int(math.Round(float64(avg)*(1-p.NetBias) + float64(hi)*p.NetBias))
	net = max(lo, min(net, hi))

	return Estimate{Min: lo, Max: hi, Average: avg, Net: net}
}

// Cost prices an estimate given the USD cost of one input token.
func (e Estimate) Cost(perToken float64) Cost {
	// Round to 1e-12 USD to hide float noise such as 0.000007500000000000001.
	price := func(tokens int) float64 {
		return math.Round(float64(tokens)*perToken*1e12) / 1e12
	}
	return Cost{Min: price(e.Min), Max: price(e.Max), Net: price(e.Net)}
}

// Summarize aggregates estimates: overall min/max and mean average/net.
func Summarize(list []Estimate) Summary {
	if len(list) == 0 {
		return Summary{}
	}
	lo, hi := list[0].Min, list[0].Max
	var avgSum, netSum int
	for _, e := range list {
		lo = min(lo, e.Min)
		hi = max(hi, e.Max)
		avgSum += e.Average
		netSum += e.Net
	}
	n := float64(len(list))
	return Summary{
		Estimate: Estimate{
			Min:     lo,
			Max:     hi,
			Average: int(math.Round(float64(avgSum) / n)),
			Net:     int(math.Round(float64(netSum) / n)),
		},
		Spread: hi - lo,
	}
}
