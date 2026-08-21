// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import (
	"maps"
	"slices"
	"strings"
	"text/template"
	"text/template/parse"
)

// currentData is the path the dot stands for at the top of a template.
const currentData = "."

// rootVariable is the variable holding the data a template was executed on.
const rootVariable = "$"

// builtins are the functions [text/template] always provides.
//
// They are left out of what a template reports calling: a caller replacing a template has to
// supply the functions of the func map, never these. A func map defining one of these names
// shadows the builtin, and calling it is then a dependency on the map after all.
var builtins = map[string]struct{}{
	"and": {}, "call": {}, "eq": {}, "ge": {}, "gt": {}, "html": {}, "index": {}, "js": {},
	"le": {}, "len": {}, "lt": {}, "ne": {}, "not": {}, "or": {}, "print": {}, "printf": {},
	"println": {}, "slice": {}, "urlquery": {},
}

// scope holds the path each variable in view stands for.
//
// A variable whose path could not be placed is held with an empty one, so that reading through it
// is counted rather than mistaken for a variable that was never declared.
type scope map[string]string

// walker follows the dot through a parse tree and records what the template does with it.
type walker struct {
	reads      map[string]struct{}
	rootReads  map[string]struct{}
	funcs      map[string]struct{}
	calls      map[Call]struct{}
	funcMap    template.FuncMap
	unresolved int
	dynamic    bool
}

// contractOf reports what a template reads and calls.
func contractOf(root parse.Node, funcs template.FuncMap) Contract {
	w := &walker{
		reads:     make(map[string]struct{}),
		rootReads: make(map[string]struct{}),
		funcs:     make(map[string]struct{}),
		calls:     make(map[Call]struct{}),
		funcMap:   funcs,
	}
	w.walk(root, currentData, make(scope))

	contract := Contract{
		Reads:      sortedKeys(w.reads),
		RootReads:  sortedKeys(w.rootReads),
		Funcs:      sortedKeys(w.funcs),
		Calls:      make([]Call, 0, len(w.calls)),
		Unresolved: w.unresolved,
		Dynamic:    w.dynamic,
	}

	for call := range w.calls {
		contract.Calls = append(contract.Calls, call)
	}

	slices.SortFunc(contract.Calls, func(a, b Call) int {
		if a.Name != b.Name {
			return strings.Compare(a.Name, b.Name)
		}

		return strings.Compare(a.Data, b.Data)
	})

	return contract
}

// walk visits a node, knowing what the dot stands for where it sits and what the variables in view
// stand for.
func (w *walker) walk(node parse.Node, dot string, vars scope) {
	switch typed := node.(type) {
	case *parse.ListNode:
		if typed == nil {
			return
		}

		// a variable declared in a list is in view for the rest of it, and no further
		inner := maps.Clone(vars)
		for _, child := range typed.Nodes {
			w.walk(child, dot, inner)
		}

	case *parse.ActionNode:
		w.pipe(typed.Pipe, dot, vars)
		w.declare(typed.Pipe, w.selected(typed.Pipe, dot, vars), vars)

	case *parse.IfNode:
		w.pipe(typed.Pipe, dot, vars)
		w.branch(typed.List, dot, w.selected(typed.Pipe, dot, vars), typed.Pipe, vars)
		w.walk(typed.ElseList, dot, vars)

	case *parse.WithNode:
		// a with rebinds the dot to whatever it selects, and binds its variable to the same
		w.pipe(typed.Pipe, dot, vars)
		selected := w.selected(typed.Pipe, dot, vars)
		w.branch(typed.List, selected, selected, typed.Pipe, vars)
		w.walk(typed.ElseList, dot, vars)

	case *parse.RangeNode:
		// a range rebinds the dot to an element of whatever it selects
		w.pipe(typed.Pipe, dot, vars)
		item := element(w.selected(typed.Pipe, dot, vars))
		w.rangeBranch(typed, item, vars)
		w.walk(typed.ElseList, dot, vars)

	case *parse.TemplateNode:
		handed := currentData
		if typed.Pipe != nil {
			handed = w.selected(typed.Pipe, dot, vars)
		}

		w.calls[Call{Name: typed.Name, Data: handed}] = struct{}{}
		w.pipe(typed.Pipe, dot, vars)
	}
}

// branch walks the body of an if or a with, with the variable the statement declares in view.
func (w *walker) branch(body parse.Node, dot, declared string, pipeline *parse.PipeNode, vars scope) {
	inner := maps.Clone(vars)
	w.declare(pipeline, declared, inner)
	w.walk(body, dot, inner)
}

// rangeBranch walks the body of a range, with the variables it declares in view.
//
// A range declaring one variable binds it to the element, and a range declaring two binds the
// first to the index, which stands for no path at all.
func (w *walker) rangeBranch(node *parse.RangeNode, item string, vars scope) {
	inner := maps.Clone(vars)

	if node.Pipe != nil {
		switch declarations := node.Pipe.Decl; len(declarations) {
		case 1:
			inner[declarations[0].Ident[0]] = item
		case 2: //nolint:mnd // an index and an element
			inner[declarations[0].Ident[0]] = ""
			inner[declarations[1].Ident[0]] = item
		}
	}

	w.walk(node.List, item, inner)
}

// declare binds the variables a pipeline declares to the path it selects.
func (w *walker) declare(pipeline *parse.PipeNode, selected string, vars scope) {
	if pipeline == nil {
		return
	}

	for _, declared := range pipeline.Decl {
		if len(declared.Ident) > 0 {
			vars[declared.Ident[0]] = selected
		}
	}
}

// pipe records what a pipeline reads and which functions it calls.
func (w *walker) pipe(pipeline *parse.PipeNode, dot string, vars scope) {
	if pipeline == nil {
		return
	}

	for _, command := range pipeline.Cmds {
		for _, argument := range command.Args {
			w.argument(argument, dot, vars)
		}
	}
}

// argument records a single term of a command.
func (w *walker) argument(argument parse.Node, dot string, vars scope) {
	switch typed := argument.(type) {
	case *parse.FieldNode:
		w.read(dot, typed.Ident)

	case *parse.ChainNode:
		w.chain(typed, dot, vars)

	case *parse.VariableNode:
		w.variable(typed, vars)

	case *parse.IdentifierNode:
		w.function(typed.Ident)

	case *parse.PipeNode:
		w.pipe(typed, dot, vars)
	}
}

// function records a call to a function of the func map.
//
// A builtin is not recorded, since it is always there, unless the func map shadows it. "call" is
// never recorded, and marks the template dynamic instead: it invokes a function held by the data,
// which the analysis cannot follow into, so what it reads stays unknown.
func (w *walker) function(name string) {
	if name == "call" {
		w.dynamic = true

		return
	}

	_, isBuiltin := builtins[name]
	_, shadowed := w.funcMap[name]

	if isBuiltin && !shadowed {
		return
	}

	w.funcs[name] = struct{}{}
}

// chain records a path read from the result of an expression, as in (index .Items 0).Name.
//
// The base resolves to a path of its own when it is a field, the dot or a variable. Otherwise it
// comes out of a function call, and the fields hanging from it belong to no path this analysis can
// name, so they are counted rather than reported under the current dot.
func (w *walker) chain(node *parse.ChainNode, dot string, vars scope) {
	if path := w.path(node, dot, vars); path != "" {
		w.reads[path] = struct{}{}

		return
	}

	w.argument(node.Node, dot, vars)
	w.unresolved++
}

// variable records what is read through a variable.
//
// "$" stands for the data the template was executed on, wherever it is read from: a range or a
// with rebinds the dot, never "$". Any other variable stands for the path it was declared with.
func (w *walker) variable(node *parse.VariableNode, vars scope) {
	if len(node.Ident) < 2 { //nolint:mnd // a lone variable reads no path of its own
		return
	}

	if node.Ident[0] == rootVariable {
		path := join(currentData, node.Ident[1:])
		w.reads[path] = struct{}{}
		w.rootReads[path] = struct{}{}

		return
	}

	w.read(vars[node.Ident[0]], node.Ident[1:])
}

// read records a path, unless the dot it hangs from could not be placed.
func (w *walker) read(dot string, idents []string) {
	if dot == "" {
		w.unresolved++

		return
	}

	w.reads[join(dot, idents)] = struct{}{}
}

// selected reports the path a range, a with or a template statement selects.
//
// It is empty when the selection is anything more involved than a field, the dot itself or a path
// from the root, since the analysis cannot follow a value through a function.
func (w *walker) selected(pipeline *parse.PipeNode, dot string, vars scope) string {
	if pipeline == nil || len(pipeline.Cmds) != 1 || len(pipeline.Cmds[0].Args) != 1 {
		return ""
	}

	return w.path(pipeline.Cmds[0].Args[0], dot, vars)
}

// path reports the data path a term stands for, empty when it stands for none the analysis can name.
func (w *walker) path(node parse.Node, dot string, vars scope) string {
	switch typed := node.(type) {
	case *parse.FieldNode:
		return join(dot, typed.Ident)

	case *parse.DotNode:
		return dot

	case *parse.ChainNode:
		base := w.path(typed.Node, dot, vars)
		if base == "" {
			return ""
		}

		return join(base, typed.Field)

	case *parse.PipeNode:
		// a term in parentheses, which stands for a path when it holds nothing but one
		return w.selected(typed, dot, vars)

	case *parse.VariableNode:
		if len(typed.Ident) == 0 {
			return ""
		}

		if typed.Ident[0] == rootVariable {
			return join(currentData, typed.Ident[1:])
		}

		if base, declared := vars[typed.Ident[0]]; declared && base != "" {
			return join(base, typed.Ident[1:])
		}
	}

	return ""
}

// join hangs a chain of field names from the path the dot stands for.
func join(dot string, idents []string) string {
	if len(idents) == 0 {
		return dot
	}

	if dot == currentData {
		return currentData + strings.Join(idents, ".")
	}

	return dot + "." + strings.Join(idents, ".")
}

// element turns the path of a collection into the path of one of its elements.
func element(path string) string {
	if path == "" {
		return ""
	}

	return path + "[]"
}

// sortedKeys returns the keys of a set, in order.
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}
