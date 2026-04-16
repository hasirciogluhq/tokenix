package input

import (
	"strings"
	"testing"
)

func TestResolveInput_PrioritizesDataFlag(t *testing.T) {
	got, source, err := ResolveInput(strings.NewReader("stdin text"), "flag text", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "flag text" {
		t.Fatalf("unexpected content: %q", got)
	}
	if source != "flag:data" {
		t.Fatalf("unexpected source: %q", source)
	}
}

func TestResolveInput_UsesStdinForNonFileReader(t *testing.T) {
	got, source, err := ResolveInput(strings.NewReader("stdin value"), "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "stdin value" {
		t.Fatalf("unexpected content: %q", got)
	}
	if source != "stdin" {
		t.Fatalf("unexpected source: %q", source)
	}
}

func TestResolveInput_NoInputReturnsError(t *testing.T) {
	_, _, err := ResolveInput(strings.NewReader("   "), "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
