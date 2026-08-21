// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/go-openapi/codegen/funcmaps"
	"github.com/go-openapi/codegen/mangling"
)

// FuncMap returns a template.FuncMap containing common Go-specific template functions,
// plus a few other commonly utility functions to work with strings, maps and numbers.
//
// For advanced users, a "dict" function (similar to sprig.Dict) is provided.
//
// It injects a [mangling.GoMangler], which may be configured with options.
//
// Callers typically merge additional entries to suit the specific needs of a generator on top of this map.
func FuncMap(mangler mangling.GoMangler) template.FuncMap {
	return funcmaps.Merge(
		othersBase(),
		stringsBase(),
		numbersBase(),
		goBase(mangler),
	)
}

func goBase(mangler mangling.GoMangler) template.FuncMap {
	goMap := template.FuncMap{
		"asFile":   mangler.File,
		"asModule": mangler.Module,
		"asPackageName": func(pth string) string {
			_, name := mangler.Package(pth)

			return name
		},
		"asPackagePath": func(pth string) string {
			mangled, _ := mangler.Package(pth)

			return mangled
		},
		"camelize":           mangler.Camelize,
		"dasherize":          mangler.Kebabize,
		"enumName":           mangler.ConstName,
		"escapeBackticks":    escapeBackticks,
		"escapeDoubleQuoted": escapeDoubleQuoted,
		"humanize":           mangler.Humanize,
		"import":             printImports,
		"jsonFieldTag":       jsonFieldTag,
		"pascalize":          mangler.IdentExported,
		"snakize":            mangler.Snakize,
		"varName":            mangler.IdentUnexported,
		"lineComment":        lineComment,
		"linePadComment":     linePadComment,
		"blockComment":       wrapBlockComment,
		"printGoLiteral":     printGoLiteral,
	}

	return goMap
}

func stringsBase() template.FuncMap {
	return template.FuncMap{
		"cleanPath":          path.Clean,
		"contains":           slices.Contains[[]string, string],
		"hasPrefix":          strings.HasPrefix,
		"joinFilePath":       filepath.Join,
		"joinPath":           path.Join,
		"json":               asJSON,
		"pluralizeFirstWord": pluralizeFirstWord,
		"prettyjson":         asPrettyJSON,
		"stringContains":     strings.Contains,
		"trimSpace":          strings.TrimSpace,
	}
}

func numbersBase() template.FuncMap {
	return template.FuncMap{
		"isInteger": isInteger,
		"gt0":       gt0,
	}
}

func othersBase() template.FuncMap {
	return template.FuncMap{
		"dict": dict,
	}
}
