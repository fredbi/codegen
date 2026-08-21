// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

// printImports takes a map of imports: keys are aliases and values are the target packages to import.
//
// The layout is current ordered by key.
func printImports(imports map[string]string) string {
	if len(imports) == 0 {
		return ""
	}

	result := make([]string, 0, len(imports))
	for k, v := range imports {
		_, name := path.Split(v)
		if name != k {
			result = append(result, fmt.Sprintf("\t%s %q", k, v))
		} else {
			result = append(result, fmt.Sprintf("\t%q", v))
		}
	}
	sort.Strings(result)
	return strings.Join(result, "\n")
}

var backticksReplacer = strings.NewReplacer("`", "`+\"`\"+`")

func escapeBackticks(arg string) string {
	return backticksReplacer.Replace(arg)
}

// escapeDoubleQuoted escapes arg so it can be safely interpolated inside a double-quoted Go string literal ("...").
//
// strconv.Quote produces a fully escaped literal.
// Stripping its surrounding quotes yields the inner form, so an embedded '"', '\' or newline in spec-derived text
// (e.g. an operation path baked into a diagnostic message) cannot break out of the literal.
//
// For text without special characters this is a no-op.
func escapeDoubleQuoted(arg string) string {
	quoted := strconv.Quote(arg)

	return quoted[1 : len(quoted)-1]
}

// jsonFieldTag renders a complete `json:"..."` struct tag from a field name.
//
// Templates that hand-write the backtick tag inline (server response headers, allOf / discriminator serializers) bypass
// PrintTags, so a backtick in the name would otherwise close the raw
// string early and inject arbitrary top-level Go.
//
// strconv.Quote escapes the name into the tag value; if the assembled tag can be backquoted it
// is emitted as a raw literal (byte-identical to the previous output for clean names),
// otherwise the whole tag is rendered as a double-quoted literal so no breakout is possible.
func jsonFieldTag(name string, omitEmpty, asString bool) string {
	value := name
	if omitEmpty {
		value += ",omitempty"
	}
	if asString {
		value += ",string"
	}

	tag := "json:" + strconv.Quote(value)
	if strconv.CanBackquote(tag) {
		return "`" + tag + "`"
	}

	return strconv.Quote(tag)
}

/*
	f["arrayInitializer"] = lang.ArrayInitializer
*/

var interfaceReplacer = strings.NewReplacer("interface {}", "any")

func printGoLiteral(arg any) string {
	return interfaceReplacer.Replace(fmt.Sprintf("%#v", arg))
}
