// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import (
	"fmt"
	"text/template"
	"text/template/parse"
)

// Analysis holds the result of reading an asset, per template it declares.
type Analysis struct {
	// Docstrings holds the comments documenting a template, by template name.
	Docstrings map[string][]string

	// Contracts holds what a template reads and calls, by template name.
	Contracts map[string]Contract
}

// Contract holds the data a template reads, and the data it passes to the templates it calls.
//
// Paths are rooted at the data the template itself is executed on, which is the argument of the
// call that reached it, not the data of whoever called that caller.
//
// A contract closes over every branch, so it holds what the template may read rather than what any
// one execution of it reads.
type Contract struct {
	// Reads lists the data paths the template may read, sorted, across every branch.
	Reads []string

	// RootReads lists the paths read through "$", sorted.
	//
	// Inside a range or a with, "$" still stands for the data the template was executed on, so a
	// path listed here is a reach past the current dot, back to the top of the template.
	RootReads []string

	// Funcs lists the func map functions the template calls, sorted.
	//
	// The builtins of [text/template] are left out: a caller replacing a template supplies the
	// func map, never those.
	Funcs []string

	// Calls lists the templates it invokes, with the data handed to each, sorted by name.
	Calls []Call

	// Unresolved counts the accesses the analysis could not place.
	Unresolved int

	// Empty reports whether the template holds nothing but white space and comments, so that running
	// it renders nothing.
	Empty bool

	// Dynamic reports whether the template invokes a function held by the data, with "call".
	//
	// Such a function is resolved only when the template runs, so a contract reported for a template
	// that uses one is incomplete by construction.
	Dynamic bool
}

// Call is the invocation of one template by another.
type Call struct {
	// Name is the template invoked.
	Name string

	// Data is the path handed to it, rooted like the paths of the calling template.
	//
	// It is "." when the caller hands over its own data, and empty when the analysis could not
	// place it.
	Data string
}

// Analyze reads an asset and reports what the templates it declares document and do.
//
// name is the template name the asset itself is registered under, which the parser also uses for
// the tree holding whatever lies outside the define statements.
//
// funcs is the map the templates are bound to. It decides which calls are worth reporting: a
// builtin of [text/template] is left out, unless the map defines a function of that name, in which
// case calling it is a dependency on the map like any other.
func Analyze(path, name string, data []byte, funcs template.FuncMap) (Analysis, error) {
	tree := parse.New(name)
	tree.Mode = parse.ParseComments | parse.SkipFuncCheck

	declared := make(map[string]*parse.Tree)
	root, err := tree.Parse(string(data), "", "", declared, nil)
	if err != nil {
		return Analysis{}, fmt.Errorf("could not analyse asset %q: %w", path, err)
	}

	analysis := Analysis{
		Docstrings: docstringsOf(root, declared, name),
		Contracts:  make(map[string]Contract, len(declared)),
	}

	for declaredName, declaredTree := range declared {
		contract := contractOf(declaredTree.Root, funcs)
		contract.Empty = parse.IsEmptyTree(declaredTree.Root)
		analysis.Contracts[declaredName] = contract
	}

	return analysis, nil
}
