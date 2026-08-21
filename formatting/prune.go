// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package formatting

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// cgoImport names the pseudo-package C, which carries a cgo preamble. Nothing may touch it.
const cgoImport = "C"

// prune deletes every import the file does not use.
//
// It never adds one, and it drops an import only when it can name the package the import declares.
// See [importNames] for what happens when it cannot.
func prune(fset *token.FileSet, file *ast.File) {
	used := usedQualifiers(file)

	type deletion struct{ name, path string }
	var unused []deletion

	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath == cgoImport {
			continue
		}

		if isUsed(spec, importPath, used) {
			continue
		}

		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		}
		unused = append(unused, deletion{name: name, path: importPath})
	}

	for _, spec := range unused {
		astutil.DeleteNamedImport(fset, file, spec.name, spec.path)
	}
}

// isUsed reports whether any name the import could answer to appears as a package qualifier.
func isUsed(spec *ast.ImportSpec, importPath string, used map[string]bool) bool {
	if spec.Name != nil {
		switch spec.Name.Name {
		case "_", ".":
			// a blank import runs an init, a dot import spills its names into the file scope, and
			// neither shows up as a qualifier. Keep both.
			return true
		default:
			// an alias names the package exactly, so there is nothing to guess
			return used[spec.Name.Name]
		}
	}

	names := importNames(importPath)
	if len(names) == 0 {
		return true // cannot name the package, so cannot call it unused
	}

	for _, name := range names {
		if used[name] {
			return true
		}
	}

	return false
}

// usedQualifiers collects every identifier the file uses to qualify a selector.
//
// An identifier the parser resolved to a declaration carries a non-nil Obj, which separates the
// package fmt in fmt.Println from a local variable named fmt. Since one shadowed use does not hide
// another, a package used anywhere in the file lands in the set.
//
// A file parsed in [fastMode] carries no Obj at all, so every selector base lands in the set. That
// is the same answer whenever no declaration in the file shares a name with a selector's qualifier.
// [needsResolution] checks that before the cheap parse is trusted.
func usedQualifiers(file *ast.File) map[string]bool {
	used := make(map[string]bool)

	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if ident, ok := selector.X.(*ast.Ident); ok && ident.Obj == nil {
			used[ident.Name] = true
		}

		return true
	})

	return used
}

// importNames lists every name an import without an alias could declare.
//
// The package name is not in the import path, so it has to be guessed, and a single guess drops
// imports that are in use: goimports reads "gopkg.in/yaml.v3" as yaml.v3 and misses yaml, while the
// version-stripping guess reads "k8s.io/api/apps/v1" as apps and misses v1. Returning both leaves
// the caller to keep the import if either name is used, and a surplus name only ever keeps an
// import.
//
// An empty result means no candidate is a legal identifier — "example.com/my-pkg" — and the caller
// keeps the import rather than delete one that may well be used.
func importNames(importPath string) []string {
	var names []string

	add := func(candidate string) {
		if !token.IsIdentifier(candidate) {
			return
		}
		for _, seen := range names {
			if seen == candidate {
				return
			}
		}
		names = append(names, candidate)
	}

	base := path.Base(importPath)

	if token.IsIdentifier(base) {
		add(base)

		// a major version suffix names no package: "github.com/go-openapi/testify/v2" declares
		// testify. The directory above it is the better guess, and base stays a candidate because
		// "k8s.io/api/apps/v1" really does declare v1.
		if isMajorVersion(base) {
			if dir := path.Dir(importPath); dir != "." {
				add(path.Base(dir))
			}
		}

		return names
	}

	// "gopkg.in/yaml.v3" declares yaml
	if dot := strings.IndexByte(base, '.'); dot >= 0 {
		add(base[:dot])
	}

	// "github.com/jessevdk/go-flags" declares flags
	add(strings.TrimPrefix(base, "go-"))

	return names
}

// isMajorVersion reports whether a path element is a module major version, as in v2 or v3.
func isMajorVersion(element string) bool {
	if len(element) < 2 || element[0] != 'v' {
		return false
	}

	_, err := strconv.Atoi(element[1:])

	return err == nil
}

// needsResolution reports whether telling a package qualifier from a shadowed name needs the
// parser's scopes.
//
// Only the names an import could declare are worth asking about, and there are a handful of those.
// It needs the scopes when the file writes one of them somewhere that is not a selector base — as a
// parameter, a range variable, the left of a :=, any declaration at all — because only scopes then
// say whether bytes.Buffer means the package or the local. When no imported name is written that
// way, an unresolved parse already separates them: a selector base the file never declares is a
// package, or a name a sibling file declares, and the parser resolves neither.
//
// Asking about every selector base instead would be useless. A method receiver is a selector base:
// m.ID is the same shape as fmt.Println, so m would collide with itself and no file would ever take
// the cheap parse.
func needsResolution(file *ast.File) bool {
	names := importedNames(file)
	if len(names) == 0 {
		return false
	}

	shadowed := false

	var walk func(ast.Node) bool
	walk = func(node ast.Node) bool {
		if shadowed {
			return false
		}

		switch typed := node.(type) {
		case *ast.ImportSpec:
			return false // the alias declares the name; nothing here shadows it

		case *ast.SelectorExpr:
			if _, ok := typed.X.(*ast.Ident); ok {
				return false // a qualifier or a value with a field, and neither declares a name
			}

			ast.Inspect(typed.X, walk) // m.Count.Value: walk the left, skip the field

			return false

		case *ast.Ident:
			if _, imported := names[typed.Name]; imported {
				shadowed = true
			}
		}

		return true
	}

	ast.Inspect(file, walk)

	return shadowed
}

// importedNames returns every name the file's imports could declare.
//
// An alias declares its name outright. Without one the package name is guessed, and [importNames]
// returns every guess, so a name that could belong to an import is in the set.
func importedNames(file *ast.File) map[string]struct{} {
	names := make(map[string]struct{}, len(file.Imports))

	for _, spec := range file.Imports {
		if spec.Name != nil {
			if spec.Name.Name != "_" && spec.Name.Name != "." {
				names[spec.Name.Name] = struct{}{}
			}

			continue
		}

		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}

		for _, name := range importNames(importPath) {
			names[name] = struct{}{}
		}
	}

	return names
}
