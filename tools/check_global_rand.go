//go:build ignore

// check_global_rand prüft, dass kein Code in den angegebenen Packages
// math/rand direkt importiert. Nur RandSource-Injection ist erlaubt.
//
// Aufruf: go run tools/check_global_rand.go ./sim/...
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: check_global_rand <pattern> [pattern...]")
		os.Exit(1)
	}

	dirs, err := resolvePatterns(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving patterns: %v\n", err)
		os.Exit(1)
	}

	violations := 0
	for _, dir := range dirs {
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
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if importsPackage(path, "math/rand") {
				fmt.Printf("FAIL: %s: imports math/rand directly\n", path)
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

// resolvePatterns converts Go package patterns (e.g. ./sim/...) to filesystem directories.
func resolvePatterns(patterns []string) ([]string, error) {
	var dirs []string
	seen := map[string]bool{}

	for _, pattern := range patterns {
		// Strip leading ./
		trimmed := strings.TrimPrefix(pattern, "./")
		recursive := strings.HasSuffix(trimmed, "/...")
		base := strings.TrimSuffix(trimmed, "/...")
		base = strings.TrimSuffix(base, "...")

		if recursive {
			// Walk from base directory recursively
			err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil // skip inaccessible paths
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
			// Single directory
			dir := base
			if dir == "" {
				dir = "."
			}
			abs, err := filepath.Abs(dir)
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

// importsPackage returns true if the file at path imports exactPkg.
func importsPackage(path, exactPkg string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return false
	}
	for _, imp := range f.Imports {
		// Import paths are quoted strings; strip surrounding quotes.
		importPath := strings.Trim(imp.Path.Value, `"`)
		if importPath == exactPkg {
			return true
		}
	}
	return false
}
