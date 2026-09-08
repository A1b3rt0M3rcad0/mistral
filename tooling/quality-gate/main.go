package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type violation struct {
	file       string
	importPath string
	reason     string
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	violations, err := scan(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(violations) == 0 {
		fmt.Println("architecture quality gate passed")
		return
	}
	for _, v := range violations {
		fmt.Fprintf(os.Stderr, "%s imports %s: %s\n", v.file, v.importPath, v.reason)
	}
	os.Exit(1)
}

func scan(root string) ([]violation, error) {
	var violations []violation
	coreRoot := filepath.Join(root, "packages", "mistral-core")
	err := filepath.WalkDir(coreRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		normalized := filepath.ToSlash(path)
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if strings.Contains(importPath, "/packages/mistral-api") || strings.Contains(importPath, "/packages/mistral-workers") || strings.Contains(importPath, "/packages/mistral-frontend") {
				violations = append(violations, violation{normalized, importPath, "core cannot depend on host packages"})
				continue
			}
			if reason := layerViolation(normalized, importPath); reason != "" {
				violations = append(violations, violation{normalized, importPath, reason})
			}
		}
		return nil
	})
	return violations, err
}

func layerViolation(filePath, importPath string) string {
	if !strings.Contains(importPath, "/packages/mistral-core/") {
		return ""
	}

	switch {
	case strings.Contains(filePath, "/domain/"):
		if containsAny(importPath, "/application/", "/infra/", "/composition/", "/presentation/", "/entrypoint/") {
			return "domain may depend only on inward domain/shared contracts, never application or outward layers"
		}
	case strings.Contains(filePath, "/application/"):
		if containsAny(importPath, "/infra/", "/composition/", "/presentation/", "/entrypoint/") {
			return "application cannot depend on infrastructure, composition, presentation or entrypoint"
		}
	case strings.Contains(filePath, "/presentation/"):
		if containsAny(importPath, "/infra/", "/composition/", "/entrypoint/") {
			return "presentation cannot depend on infrastructure, composition or entrypoint"
		}
	case strings.Contains(filePath, "/infra/"):
		if containsAny(importPath, "/composition/", "/presentation/", "/entrypoint/") {
			return "infrastructure cannot depend on composition, presentation or entrypoint"
		}
	}
	return ""
}

func containsAny(value string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}
