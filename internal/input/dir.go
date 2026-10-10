package input

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// defaultSkipDirs are never descended into when scanning a directory.
var defaultSkipDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
}

// DirResult is the outcome of a recursive directory scan.
type DirResult struct {
	Content string
	Files   []string // slash-separated paths relative to the scanned root
}

// ReadDir walks root recursively and concatenates the text of every file that
// passes the include/exclude filters.
//
// A pattern matches a file when it matches the relative path, the base name,
// any parent directory name, or acts as a "dir/**" prefix. Patterns use
// path.Match syntax. An empty include list means "everything".
// Binary files and unreadable symlinks are skipped.
func ReadDir(root string, include, exclude []string) (DirResult, error) {
	info, err := os.Stat(root)
	if err != nil {
		return DirResult{}, fmt.Errorf("directory could not be read: %w", err)
	}
	if !info.IsDir() {
		return DirResult{}, fmt.Errorf("not a directory: %s", root)
	}

	var files []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if _, skip := defaultSkipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			if matchesAny(exclude, rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if matchesAny(exclude, rel) {
			return nil
		}
		if len(include) > 0 && !matchesAny(include, rel) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return DirResult{}, fmt.Errorf("directory walk failed: %w", err)
	}
	sort.Strings(files)

	var sb strings.Builder
	used := make([]string, 0, len(files))
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return DirResult{}, fmt.Errorf("file could not be read: %w", err)
		}
		if isBinary(b) {
			continue
		}
		sb.Write(b)
		sb.WriteByte('\n')
		used = append(used, rel)
	}

	if len(used) == 0 {
		return DirResult{}, errors.New("no matching text files found in directory")
	}
	return DirResult{Content: sb.String(), Files: used}, nil
}

func isBinary(b []byte) bool {
	n := min(len(b), 8000)
	for _, c := range b[:n] {
		if c == 0 {
			return true
		}
	}
	return false
}

func matchesAny(patterns []string, rel string) bool {
	for _, pat := range patterns {
		if matchPattern(strings.TrimSpace(pat), rel) {
			return true
		}
	}
	return false
}

func matchPattern(pat, rel string) bool {
	if pat == "" {
		return false
	}
	pat = strings.TrimPrefix(pat, "./")

	if prefix, ok := strings.CutSuffix(pat, "/**"); ok {
		return rel == prefix || strings.HasPrefix(rel, prefix+"/")
	}
	if ok, _ := path.Match(pat, rel); ok {
		return true
	}
	// Match the base name and every parent directory name, so "*.go" and
	// "vendor" work at any depth.
	for _, seg := range strings.Split(rel, "/") {
		if ok, _ := path.Match(pat, seg); ok {
			return true
		}
	}
	return false
}
