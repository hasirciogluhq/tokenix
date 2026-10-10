package input

import (
	"os"
	"path/filepath"
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

func TestResolveInput_ReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(path, []byte("from file"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, source, err := ResolveInput(strings.NewReader("ignored stdin"), "", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from file" || source != "file:"+path {
		t.Fatalf("unexpected result: %q %q", got, source)
	}
}

func TestResolveInput_FileErrors(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveInput(strings.NewReader(""), "", empty); err == nil {
		t.Fatal("expected error for empty file")
	}
	if _, _, err := ResolveInput(strings.NewReader(""), "", filepath.Join(dir, "missing.txt")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pat, rel string
		want     bool
	}{
		{"*.go", "a/b/c.go", true},
		{"*.go", "a/b/c.md", false},
		{"vendor", "x/vendor/y.go", true},
		{"src/**", "src/a/b.go", true},
		{"src/**", "srcx/a.go", false},
		{"./docs/*.md", "docs/a.md", true},
		{"", "a.go", false},
	}
	for _, c := range cases {
		if got := matchPattern(c.pat, c.rel); got != c.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", c.pat, c.rel, got, c.want)
		}
	}
}
