package input

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestReadDir_RecursiveAndFilters(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a.go":               "package a",
		"b.md":               "# b",
		"sub/c.go":           "package c",
		"sub/deep/d.go":      "package d",
		"sub/deep/d_test.go": "package d_test",
		"vendor/v.go":        "package v",
		".git/config":        "git",
		"node_modules/x.js":  "x",
		"bin.dat":            "ab\x00cd",
	})

	tests := []struct {
		name    string
		include []string
		exclude []string
		want    []string
	}{
		{"all text files", nil, nil, []string{"a.go", "b.md", "sub/c.go", "sub/deep/d.go", "sub/deep/d_test.go", "vendor/v.go"}},
		{"include ext any depth", []string{"*.go"}, nil, []string{"a.go", "sub/c.go", "sub/deep/d.go", "sub/deep/d_test.go", "vendor/v.go"}},
		{"include + exclude", []string{"*.go"}, []string{"*_test.go", "vendor"}, []string{"a.go", "sub/c.go", "sub/deep/d.go"}},
		{"dir prefix include", []string{"sub/**"}, nil, []string{"sub/c.go", "sub/deep/d.go", "sub/deep/d_test.go"}},
		{"exclude dir name", nil, []string{"deep", "vendor"}, []string{"a.go", "b.md", "sub/c.go"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ReadDir(root, tc.include, tc.exclude)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res.Files, tc.want) {
				t.Fatalf("files = %v, want %v", res.Files, tc.want)
			}
		})
	}
}

func TestReadDir_Errors(t *testing.T) {
	root := writeTree(t, map[string]string{"a.txt": "hi"})
	if _, err := ReadDir(filepath.Join(root, "missing"), nil, nil); err == nil {
		t.Fatal("expected error for missing dir")
	}
	if _, err := ReadDir(filepath.Join(root, "a.txt"), nil, nil); err == nil {
		t.Fatal("expected error for file path")
	}
	if _, err := ReadDir(root, []string{"*.go"}, nil); err == nil {
		t.Fatal("expected error when nothing matches")
	}
}
