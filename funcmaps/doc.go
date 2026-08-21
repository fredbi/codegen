// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package funcmaps exposes utilities to work with [template.FuncMap].
//
// * funcmap merging, with guards against unwary overrides (coalesce, protecting builtins)
// * a default funcmap for golang codegen, with common mangling for go identifiers, go comments handling etc
package funcmaps
