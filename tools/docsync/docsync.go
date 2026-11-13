package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A documentation page embeds code with a fenced block whose first line names
// the source:
//
//	```go
//	// examples/login.go            <- the whole file
//	```
//
//	```go
//	// examples/components.go#ButtonExample   <- one function
//	```
//
// docsync fills (or refreshes) such blocks from the real, tested source.
var blockRe = regexp.MustCompile("(?s)```go\n(// examples/[^\n]+)\n(.*?)```")

// syncAll refreshes every embedded block of the .md files under root/docs. It
// returns the files whose content differs from the sources; with write=true it
// also rewrites them.
func syncAll(root string, write bool) (stale []string, err error) {
	docs := filepath.Join(root, "docs")
	if _, statErr := os.Stat(docs); os.IsNotExist(statErr) {
		return nil, nil
	}
	err = filepath.WalkDir(docs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return walkErr
		}
		changed, err := syncFile(root, path, write)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if changed {
			stale = append(stale, path)
		}
		return nil
	})
	return stale, err
}

func syncFile(root, path string, write bool) (changed bool, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	old := normalize(string(raw))

	var firstErr error
	updated := blockRe.ReplaceAllStringFunc(old, func(block string) string {
		header := blockRe.FindStringSubmatch(block)[1]
		code, err := source(root, header)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", header, err)
			}
			return block
		}
		return "```go\n" + header + "\n" + code + "\n```"
	})
	if firstErr != nil {
		return false, firstErr
	}
	if updated == old {
		return false, nil
	}
	if write {
		err = os.WriteFile(path, []byte(updated), 0o644)
	}
	return true, err
}

// source returns the code a header refers to.
func source(root, header string) (string, error) {
	ref := strings.TrimPrefix(header, "// ")
	file, fn := ref, ""
	if i := strings.Index(ref, "#"); i >= 0 {
		file, fn = ref[:i], ref[i+1:]
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return "", err
	}
	text := normalize(string(data))
	if fn != "" {
		return function(file, text, fn)
	}
	// A whole file: its first line is already the header comment.
	return strings.TrimRight(strings.TrimPrefix(text, header+"\n"), "\n"), nil
}

// function extracts the declaration of fn (with its doc comment) from Go source.
func function(file, text, fn string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, text, parser.ParseComments)
	if err != nil {
		return "", err
	}
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Recv != nil || d.Name.Name != fn {
			continue
		}
		start := d.Pos()
		if d.Doc != nil {
			start = d.Doc.Pos()
		}
		return text[fset.Position(start).Offset:fset.Position(d.End()).Offset], nil
	}
	return "", fmt.Errorf("function %s not found in %s", fn, file)
}

func normalize(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
