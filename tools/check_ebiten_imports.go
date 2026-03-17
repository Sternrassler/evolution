//go:build ignore

// check_ebiten_imports prüft, dass sim/, gen/ und config/ kein ebiten importieren.
//
// Aufruf: go run tools/check_ebiten_imports.go ./...
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// restrictedPrefixes are the package path prefixes where ebiten must not appear.
var restrictedPrefixes = []string{"sim/", "gen/", "config/", "sim", "gen", "config"}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: check_ebiten_imports <pattern> [pattern...]")
		os.Exit(1)
	}

	dirs, err := resolvePatterns(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving patterns: %v\n", err)
		os.Exit(1)
	}

	violations := 0
	for _, dir := range dirs {
		if !isRestricted(dir) {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if importsEbiten(path) {
				fmt.Printf("FAIL: %s: imports ebiten in restricted package\n", path)
				violations++
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error walking %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	if violations > 0 {
		os.Exit(1)
	}
}

// isRestricted returns true if the directory is under sim/, gen/, or config/.
func isRestricted(dir string) bool {
	// Normalise to use forward slashes for comparison
	norm := filepath.ToSlash(dir)

	// Check if any path component matches a restricted package root
	for _, prefix := range []string{"sim", "gen", "config"} {
		// Match exact component or sub-path
		if strings.Contains(norm, "/"+prefix+"/") ||
			strings.Contains(norm, "/"+prefix) ||
			strings.HasSuffix(norm, "/"+prefix) ||
			norm == prefix {
			return true
		}
	}
	return false
}

// resolvePatterns converts Go package patterns (e.g. ./...) to filesystem directories.
func resolvePatterns(patterns []string) ([]string, error) {
	var dirs []string
	seen := map[string]bool{}

	for _, pattern := range patterns {
		trimmed := strings.TrimPrefix(pattern, "./")
		recursive := strings.HasSuffix(trimmed, "/...") || trimmed == "..."
		base := strings.TrimSuffix(trimmed, "/...")
		base = strings.TrimSuffix(base, "...")
		if base == "" {
			base = "."
		}

		if recursive {
			err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					abs, absErr := filepath.Abs(path)
					if absErr != nil {
						return absErr
					}
					if !seen[abs] {
						seen[abs] = true
						dirs = append(dirs, abs)
					}
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walking %s: %w", base, err)
			}
		} else {
			abs, err := filepath.Abs(base)
			if err != nil {
				return nil, err
			}
			if !seen[abs] {
				seen[abs] = true
				dirs = append(dirs, abs)
			}
		}
	}

	return dirs, nil
}

// importsEbiten returns true if the file imports any package containing "ebiten".
func importsEbiten(path string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return false
	}
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(importPath, "ebiten") {
			return true
		}
	}
	return false
}
