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
			}
			if strings.Contains(normalized, "/domain/") && isOutwardLayer(importPath) {
				violations = append(violations, violation{normalized, importPath, "domain cannot depend on infra, composition, presentation or entrypoint"})
			}
		}
		return nil
	})
	return violations, err
}

func isOutwardLayer(importPath string) bool {
	for _, marker := range []string{"/infra/", "/composition/", "/presentation/", "/entrypoint/"} {
		if strings.Contains(importPath, marker) {
			return true
		}
	}
	return false
}
