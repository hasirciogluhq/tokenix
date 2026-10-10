package token

import (
	"math"
	"testing"
)

func TestCountRejectsBlank(t *testing.T) {
	if _, err := Count("  \n\t "); err == nil {
		t.Fatal("expected error for blank input")
	}
}

func TestCountUsesRunes(t *testing.T) {
	c, err := Count("héllo wörld")
	if err != nil {
		t.Fatal(err)
	}
	if c.Chars != 11 || c.Words != 2 {
		t.Fatalf("unexpected counts: %+v", c)
	}
}

func TestEstimateOrdering(t *testing.T) {
	c, _ := Count("Hello, this is a sample text for token estimation.")
	for _, p := range []Profile{Global, ProfileFor("openai"), ProfileFor("anthropic"), ProfileFor("gemini")} {
		e := p.Estimate(c)
		if e.Min < 1 || e.Min > e.Net || e.Net > e.Max || e.Average < e.Min || e.Average > e.Max {
			t.Fatalf("inconsistent estimate: %+v", e)
		}
	}
}

func TestProfileForUnknownProviderIsGlobal(t *testing.T) {
	if ProfileFor("some-new-lab") != Global {
		t.Fatal("expected global profile")
	}
}

func TestCost(t *testing.T) {
	c := Estimate{Min: 100, Max: 300, Average: 200, Net: 250}.Cost(2e-6)
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-12 }
	if !near(c.Min, 0.0002) || !near(c.Max, 0.0006) || !near(c.Net, 0.0005) {
		t.Fatalf("unexpected cost: %+v", c)
	}
}

func TestSummarize(t *testing.T) {
	s := Summarize([]Estimate{
		{Min: 10, Max: 20, Average: 15, Net: 16},
		{Min: 12, Max: 30, Average: 21, Net: 24},
	})
	if s.Min != 10 || s.Max != 30 || s.Spread != 20 || s.Average != 18 || s.Net != 20 {
		t.Fatalf("unexpected summary: %+v", s)
	}
	if Summarize(nil) != (Summary{}) {
		t.Fatal("empty summary should be zero")
	}
}
