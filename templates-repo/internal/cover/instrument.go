// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cover

import (
	"strconv"
	"sync"
	"text/template"
	"text/template/parse"
)

// Instrumented is a set of trees that count the lines they run, with the function to bind for them
// to count.
type Instrumented struct {
	// Trees holds the rewritten tree of every template the asset declares, by name.
	Trees map[string]*parse.Tree

	// FuncName is the name the trees call to count a line.
	FuncName string

	// Func is the function that name must be bound to before the trees run.
	Func func(int) string
}

// Instrument rewrites the trees of an asset so that running them counts the lines they reach.
//
// Every line a tree holds is recorded, whether it ever runs or not, so the profile distinguishes
// a line that was not reached from one that does not exist.
//
// The trees are rewritten rather than altered: what the caller passed is left as it was.
func (p *Profile) Instrument(assetPath string, source []byte, trees map[string]*parse.Tree) Instrumented {
	lines := newLineTable(source)
	name := callbackName()

	instrumented := Instrumented{
		Trees:    make(map[string]*parse.Tree, len(trees)),
		FuncName: name,
		Func: func(line int) string {
			if counter := p.counterFor(assetPath, line); counter != nil {
				counter.Add(1)
			}

			return ""
		},
	}

	writer := &rewriter{profile: p, assetPath: assetPath, lines: lines, funcName: name}
	for declared, tree := range trees {
		rewritten := tree.Copy()
		rewritten.Root.Nodes = writer.nodes(rewritten.Root)
		instrumented.Trees[declared] = rewritten
	}

	return instrumented
}

// rewriter walks a tree and puts a counter before everything that runs.
type rewriter struct {
	profile   *Profile
	assetPath string
	lines     *lineTable
	funcName  string
}

// nodes rewrites the children of a list, counting each of them.
func (w *rewriter) nodes(list *parse.ListNode) []parse.Node {
	if list == nil {
		return nil
	}

	rewritten := make([]parse.Node, 0, 2*len(list.Nodes)) //nolint:mnd // a counter before each node
	counted := 0

	for _, node := range list.Nodes {
		w.descend(node)

		// a line is counted once per list, so that text followed by an action on one line counts
		// once. A branch is a list of its own, and keeps a counter of its own
		if line := w.lineOf(node); line != 0 && line != counted {
			counted = line
			rewritten = append(rewritten, w.counter(line))
		}

		rewritten = append(rewritten, node)
	}

	return rewritten
}

// descend rewrites the bodies a node holds, so that a branch counts on its own.
func (w *rewriter) descend(node parse.Node) {
	switch typed := node.(type) {
	case *parse.IfNode:
		w.branch(typed.List, typed.ElseList)
	case *parse.RangeNode:
		w.branch(typed.List, typed.ElseList)
	case *parse.WithNode:
		w.branch(typed.List, typed.ElseList)
	}
}

// branch rewrites the two bodies of a branching node.
func (w *rewriter) branch(list, elseList *parse.ListNode) {
	if list != nil {
		list.Nodes = w.nodes(list)
	}

	if elseList != nil {
		elseList.Nodes = w.nodes(elseList)
	}
}

// lineOf reports the line a node is counted on, or zero when it is counted on none.
//
// A node rendering nothing of its own, such as the white space between two actions, is counted on
// no line: a mark on it would claim a line that holds no output. Nor is a line holding nothing,
// which would make a block of no width.
func (w *rewriter) lineOf(node parse.Node) int {
	if !runs(node) {
		return 0
	}

	line := w.lines.at(w.lines.skipSpace(int(node.Position())))
	if w.lines.length(line) == 0 {
		return 0
	}

	return line
}

// counter builds the node counting a line, and records that line so that the profile holds it
// whether it ever runs or not.
func (w *rewriter) counter(line int) parse.Node {
	w.profile.register(w.assetPath, line, w.lines.length(line))

	action := counterProto().Copy().(*parse.ActionNode)
	args := action.Pipe.Cmds[0].Args
	args[0].(*parse.IdentifierNode).Ident = w.funcName

	number := args[1].(*parse.NumberNode)
	number.Int64 = int64(line)
	number.Text = strconv.Itoa(line)

	return action
}

// counterPlaceholder is the identifier the prototype calls; [rewriter.counter] renames each copy
// to the callback the asset was instrumented with.
const counterPlaceholder = "coverPlaceholder"

// counterProto returns the action every counter is copied from.
//
// The nodes cannot be assembled from struct literals. A node carries an unexported pointer to the
// tree that parsed it, and since Go 1.27 [parse.ActionNode.String] reads the delimiters off that
// pointer and [parse.ActionNode.Copy] allocates through it, so a hand-built node panics on both.
// Parsing one action and copying it gives every counter a tree to point at.
//
// A copy keeps the position it was parsed at rather than the position of the node it counts, so
// that [parse.Tree.ErrorContext] indexes the prototype's own source and stays in range.
var counterProto = sync.OnceValue(func() *parse.ActionNode {
	tree := parse.New("cover")
	tree.Mode = parse.SkipFuncCheck // the callback is bound at run time, not known here

	parsed, err := tree.Parse("{{"+counterPlaceholder+" 0}}", "", "", make(map[string]*parse.Tree))
	if err != nil {
		panic(err) // a constant template: parsing it can only fail if this file is wrong
	}

	return parsed.Root.Nodes[0].(*parse.ActionNode)
})

// runs reports whether a node emits output of its own, or controls whether something else does.
func runs(node parse.Node) bool {
	switch typed := node.(type) {
	case *parse.TextNode:
		// the white space between two actions renders, but belongs to no line of its own
		return len(trimSpace(typed.Text)) > 0

	case *parse.ActionNode, *parse.IfNode, *parse.RangeNode, *parse.WithNode,
		*parse.TemplateNode, *parse.BreakNode, *parse.ContinueNode:
		return true

	default:
		return false
	}
}

// trimSpace is [bytes.TrimSpace], kept here to say what it is used for.
func trimSpace(text []byte) []byte {
	start, end := 0, len(text)
	for start < end && isSpace(text[start]) {
		start++
	}

	for end > start && isSpace(text[end-1]) {
		end--
	}

	return text[start:end]
}

// isSpace reports whether a byte is white space, as the template lexer defines it.
func isSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

// Bind returns the func map a set of instrumented trees needs to run.
func (i Instrumented) Bind() template.FuncMap {
	return template.FuncMap{i.FuncName: i.Func}
}
