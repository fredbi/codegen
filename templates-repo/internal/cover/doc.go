// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package cover measures which lines of a template a program reaches when it runs.
//
// It works the way the go toolchain does: a counter is injected before every statement while the
// templates are compiled, and executing them counts. The result is a profile in the format of
// go test, which go tool cover renders.
//
// # Granularity
//
// Counting is per line. The parser gives the offset of a token and not its extent, so a block
// covers the whole line it starts on, and two branches written on one line share a counter.
//
// A line holding nothing but a define, an end or an else leaves no node in the parse tree, so
// nothing counts it and nothing is written for it. It renders as plain text, the way a go
// declaration does.
//
// # Reading the profile
//
// go tool cover finds the file a profile names by asking go list, so the paths have to read as an
// import path of a package that exists. The prefix a caller gives [NewProfile] is prepended to
// every asset path for that reason.
//
// Only the html output is worth aiming at: go tool cover -func reads the file as go source and
// stops at the first template action.
package cover
