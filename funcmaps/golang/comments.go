// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"reflect"
	"strings"
)

// lineComment renders its arguments as a complete Go line-comment block, emitting
// the "//" markers itself so call sites only pass content.
//
// Arguments are stringified and concatenated with fmt.Sprint semantics (a space is
// inserted only between two non-string operands); nil arguments are skipped, so
// optional template values drop out cleanly. This lets a composed comment be built
// inline, e.g. {{ lineComment "MinProperties: " .MinProperties }}.
//
// Besides factoring out comment construction, it harmonizes the output: every line
// is prefixed with "// " (a single space), CR/CRLF are normalized, and per-line
// trailing whitespace is trimmed.
//
// Blank lines are preserved (they matter to godoc and to codescan).
//
// A blank line left trailing the whole comment block is dropped by the Go formatter.
//
// Splitting on embedded newlines keeps multi-line content fully commented so it cannot break out of the comment,
// and the guaranteed space after "//" keeps content from accidentally (or maliciously)
// forming a compiler directive such as //go:embed or //line.
//
// Blank input yields no output.
func lineComment(args ...any) string {
	return renderLineComment("// ", args)
}

// linePadComment renders its arguments as a Go line-comment block like [lineComment],
// but indents every rendered line by pad after the "//" marker,
// including the continuation lines produced by a multi-line argument.
//
// This preserves a fixed indentation across wrapped lines, as required by the
// indentation-significant swagger:meta package doc block.
//
// pad is the indentation that follows "//": a pad of "  " yields "//  text". A
// leading space is inserted when pad does not already start with whitespace, so
// the marker can never accidentally (or maliciously) form a compiler directive.
// Empty content yields no output.
func linePadComment(pad string, args ...any) string {
	if pad == "" || (pad[0] != ' ' && pad[0] != '\t') {
		pad = " " + pad
	}

	return renderLineComment("//"+pad, args)
}

// wrapBlockComment renders text as a complete Go block comment.
//
// It emits the "/*" and "*/" markers itself so call sites only pass the text.
//
// It neutralizes any inner "*/" so spec text cannot terminate the comment early,
// normalizes newlines and trims trailing whitespace. Single-line text stays
// inline ("/* text */"); multi-line text is wrapped on its own lines.
//
// Empty input yields no output.
func wrapBlockComment(str string) string {
	str = strings.TrimRight(normalizeNewlines(str), " \t\n")
	if str == "" {
		return ""
	}

	str = strings.ReplaceAll(str, "*/", "[*]/")

	if strings.ContainsRune(str, '\n') {
		return "/*\n" + str + "\n*/"
	}

	return "/* " + str + " */"
}

// renderLineComment is the shared core of [lineComment] and [linePadComment].
//
// It stringifies args (fmt.Sprint semantics, nil arguments skipped), normalizes CR/CRLF,
// trims trailing whitespace from each line, then prefixes every line with marker (blank lines become a bare "//").
//
// Splitting on embedded newlines keeps multi-line content fully commented so it cannot break out of the comment.
//
// Blank lines are preserved, including a blank line trailing the content:
// they carry meaning for godoc paragraphs and for codescan.
//
// A blank line that ends up trailing the whole comment block is left to the Go formatter to drop:
// renderLineComment may be called several times and it cannot determine which trailing blank line will be the
// last block to trim.
//
// Content that is entirely blank yields no output.
func renderLineComment(marker string, args []any) string {
	kept := make([]any, 0, len(args))
	for _, arg := range args {
		if deref, ok := derefArg(arg); ok {
			kept = append(kept, deref)
		}
	}

	str := normalizeNewlines(fmt.Sprint(kept...))
	if strings.TrimSpace(str) == "" {
		return ""
	}

	lines := strings.Split(str, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			lines[i] = "//"

			continue
		}

		lines[i] = marker + line
	}

	return strings.Join(lines, "\n")
}

// derefArg unwraps pointer arguments so they stringify by value, mirroring how
// text/template prints a pointer field with {{ .X }}.
//
// A nil interface or a nil pointer is reported as absent (ok=false) so the caller can skip it.
// This keeps composed comments such as {{ lineComment "MinProperties: " .MinProperties }}
// printing the int64 value rather than the *int64 address.
func derefArg(arg any) (any, bool) {
	const limit = 1000 // guard against malicious overflow

	if arg == nil {
		return nil, false
	}

	v := reflect.ValueOf(arg)
	for i := 0; v.Kind() == reflect.Pointer; i++ {
		if i > limit {
			return nil, false
		}

		if v.IsNil() {
			return nil, false
		}

		v = v.Elem()
	}

	if v.CanInterface() {
		return v.Interface(), true
	}

	return nil, false
}

// normalizeNewlines rewrites CRLF and lone CR to LF so comment helpers can split reliably on "\n".
func normalizeNewlines(str string) string {
	if !strings.ContainsRune(str, '\r') {
		return str
	}

	str = strings.ReplaceAll(str, "\r\n", "\n")

	return strings.ReplaceAll(str, "\r", "\n")
}
