// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package funcmaps

import (
	"maps"
	"text/template"
)

// Merge [template.FuncMap] s into the target [template.FuncMap].
//
// Merge is performed with last wins, overwriting keys.
//
// Built-in template functions are always protected from an override.
func Merge(target template.FuncMap, merged ...template.FuncMap) template.FuncMap {
	m := mergeMaps(target, merged...)

	return omitBuiltins(m)
}

// Coalesce [template.FuncMap] s into the target [template.FuncMap].
//
// Coalesce is a merge with first wins, never overwriting keys.
//
// Built-in template functions are implied and are therefore always protected from an override.
func Coalesce(target template.FuncMap, coalesced ...template.FuncMap) template.FuncMap {
	m := coalesceMaps(target, coalesced...)

	return omitBuiltins(m)
}

// Pick cherrypicks a list of functions from a [template.FuncMap] and yields a clone with only
// the selected ones. Non-existent symbols are silently ignored.
func Pick(source template.FuncMap, picked ...string) template.FuncMap {
	m := make(template.FuncMap, len(source))

	for _, selected := range picked {
		reexported, ok := source[selected]
		if !ok {
			continue
		}

		m[selected] = reexported
	}

	return m
}

func omitBuiltins(m template.FuncMap) template.FuncMap {
	for _, builtin := range builtinFuncMap {
		delete(m, builtin)
	}

	return m
}

// builtins holds the name of all built-in functions.
//
// These are provided by the text/templates package.
//
// See https://pkg.go.dev/text/template@go1.26.5#hdr-Functions
var builtinFuncMap = []string{
	"and", "not", "or",
	"eq", "ge", "gt", "le", "lt", "ne",
	"call",
	"index", "slice", "len",
	"print", "printf", "println",
	"html", "js", "urlquery",
}

// mergeMaps merges maps into the target.
//
// If the target is nil, a new merged map is created.
//
// Merge semantics are: overwrite, last win.
func mergeMaps[M ~map[K]V, K comparable, V any](target M, merged ...M) M {
	if target == nil {
		var c int
		for _, m := range merged {
			c += len(m)
		}
		target = make(map[K]V, c)
	}

	for _, m := range merged {
		maps.Copy(target, m)
	}

	return target
}

// coalesceMaps merges maps into the target.
//
// If the target is nil, a new merged map is created.
//
// Merge semantics are: coalesce without overwrite, first win.
func coalesceMaps[M ~map[K]V, K comparable, V any](target M, coalesced ...M) M {
	if target == nil {
		var c int
		for _, m := range coalesced {
			c += len(m)
		}
		target = make(map[K]V, c)
	}

	for _, co := range coalesced {
		for k, v := range co {
			_, found := target[k]
			if found {
				continue
			}
			target[k] = v
		}
	}

	return target
}
