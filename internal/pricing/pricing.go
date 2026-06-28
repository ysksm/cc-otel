// Package pricing estimates LLM call cost from token counts when a provider does
// not report cost directly. Rates are approximate and the result is flagged as
// estimated. The table is intentionally small and easy to extend.
package pricing

import "strings"

// microcentsPerUSD: latitude-style fixed-point money (1 USD = 100,000,000 microcents).
const microcentsPerUSD = 100_000_000.0

// rate holds USD price per 1,000,000 tokens for each token class.
type rate struct {
	input     float64
	output    float64
	cacheRead float64
	cacheWrite float64
	reasoning float64
}

// table is keyed by a lowercase substring matched against the model name.
// Order matters: more specific keys should come first.
var table = []struct {
	match string
	rate  rate
}{
	// Anthropic Claude
	{"claude-opus", rate{15, 75, 1.5, 18.75, 75}},
	{"opus", rate{15, 75, 1.5, 18.75, 75}},
	{"claude-sonnet", rate{3, 15, 0.3, 3.75, 15}},
	{"sonnet", rate{3, 15, 0.3, 3.75, 15}},
	{"claude-haiku", rate{0.8, 4, 0.08, 1, 4}},
	{"haiku", rate{0.8, 4, 0.08, 1, 4}},
	// OpenAI
	{"gpt-4o-mini", rate{0.15, 0.6, 0.075, 0, 0.6}},
	{"gpt-4o", rate{2.5, 10, 1.25, 0, 10}},
	{"gpt-4.1-mini", rate{0.4, 1.6, 0.1, 0, 1.6}},
	{"gpt-4.1", rate{2, 8, 0.5, 0, 8}},
	{"gpt-4-turbo", rate{10, 30, 0, 0, 30}},
	{"o1", rate{15, 60, 7.5, 0, 60}},
	{"o3", rate{2, 8, 0.5, 0, 8}},
	// Google Gemini
	{"gemini-1.5-pro", rate{1.25, 5, 0.3125, 0, 5}},
	{"gemini-1.5-flash", rate{0.075, 0.3, 0.01875, 0, 0.3}},
	{"gemini-2", rate{1.25, 5, 0.3125, 0, 5}},
}

// Estimate returns the input/output/total cost in microcents for the given model
// and token counts. ok is false when no rate is known for the model.
func Estimate(model string, tokensInput, tokensOutput, cacheRead, cacheWrite, reasoning int64) (in, out, total int64, ok bool) {
	m := strings.ToLower(model)
	var r rate
	found := false
	for _, e := range table {
		if strings.Contains(m, e.match) {
			r = e.rate
			found = true
			break
		}
	}
	if !found {
		return 0, 0, 0, false
	}
	usd := func(tokens int64, perM float64) float64 { return float64(tokens) / 1_000_000.0 * perM }
	inUSD := usd(tokensInput, r.input) + usd(cacheRead, r.cacheRead) + usd(cacheWrite, r.cacheWrite)
	outUSD := usd(tokensOutput, r.output) + usd(reasoning, r.reasoning)
	in = int64(inUSD * microcentsPerUSD)
	out = int64(outUSD * microcentsPerUSD)
	total = in + out
	return in, out, total, true
}

// USDToMicrocents converts a USD float to microcents.
func USDToMicrocents(usd float64) int64 { return int64(usd * microcentsPerUSD) }
