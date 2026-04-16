package token

import "testing"

func TestParseModelSelectionAll(t *testing.T) {
	models, err := ParseModelSelection("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != len(SupportedModels()) {
		t.Fatalf("expected %d models, got %d", len(SupportedModels()), len(models))
	}
}

func TestParseModelSelectionList(t *testing.T) {
	models, err := ParseModelSelection("gpt-4o, gemini-1.5-pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0] != ModelGPT4o || models[1] != ModelGemini15Pro {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestEstimateSummarySpread(t *testing.T) {
	result, err := Estimate("Merhaba bu bir token tahmin test içeriğidir.", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least one model result")
	}
	if result.Summary.Max < result.Summary.Min {
		t.Fatalf("invalid summary range: min=%d max=%d", result.Summary.Min, result.Summary.Max)
	}
	if result.Summary.Spread != result.Summary.Max-result.Summary.Min {
		t.Fatalf("invalid spread: %d", result.Summary.Spread)
	}
}

func TestEstimateSingleModel(t *testing.T) {
	result, err := Estimate("token sayısı için kısa test", ModelGPT4oMini)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(result.Results))
	}
	if result.Results[0].Model != ModelGPT4oMini {
		t.Fatalf("unexpected model: %s", result.Results[0].Model)
	}
}

func TestEstimateUnsupportedModel(t *testing.T) {
	_, err := Estimate("hello", "unknown-model")
	if err == nil {
		t.Fatal("expected error for unsupported model")
	}
}
