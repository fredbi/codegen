// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package document analyses template sources, rather than compiling them.
//
// A go template declares nothing about the data it runs on: which data model fits which template is
// knowledge that lives outside the language. The parse tree, however, records every access, and the
// data the current dot stands for can be followed through range, with and template statements. What
// a template expects is therefore recoverable, as the set of paths it reads.
//
// [Analyze] reads one asset and reports, per template it declares, the comments documenting it and
// the contract it works to.
//
// # What the result means
//
// The result closes over every branch, so it holds everything a template may read, not what one
// execution of it does read. A path guarded by a condition is reported like any other, and the
// tree cannot tell which of them a given run will reach.
//
// It is therefore what the data has to be able to answer, and not a list of what it must hold.
//
// # Limits
//
// An access the analysis cannot place is counted rather than dropped, so an incomplete contract
// is visible as such. A dot or a variable that comes out of a function call produces one,
// since a value cannot be followed through a function.
//
// A template invoking a function held by its data, with the builtin "call", is reported as
// dynamic: what such a function reads is decided when the template runs. Nothing else invokes a
// template or reads data by a name computed at run time, and a func map that did would be
// invisible here.
//
// A variable assigned inside a block keeps, for the rest of the block, the path it is assigned.
// Past the end of that block the analysis reports the path it was declared with, so an assignment
// intended to outlive its block is not followed.
package document
