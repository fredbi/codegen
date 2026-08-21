// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import (
	"bytes"
	"slices"
	"strings"
	"text/template/parse"
)

// commentGroup is a run of comments that documents whatever follows it.
//
// Comments are grouped the way a Go author expects: a blank line starts a new group, a single line
// break does not. The index is that of the last comment of the group in the nodes of the tree it
// was found in, and locates a template close enough after it to claim it.
type commentGroup struct {
	comments []string
	index    int
	position parse.Pos
}

// docstringsOf extracts the documentation an asset holds, per template it declares.
//
// A template declared by a "define" statement is documented by the comment group that precedes it,
// separated from it by nothing but white space. What is left of the leading group documents the
// asset itself.
//
// Comments anywhere else are comments: a note in the middle of a template, or one nested inside a
// branch, documents nothing and is ignored.
func docstringsOf(root *parse.Tree, declared map[string]*parse.Tree, name string) map[string][]string {
	nodes := root.Root.Nodes
	groups := commentGroupsOf(nodes)
	docstrings := make(map[string][]string, len(declared))

	// a "define" statement leaves no node behind, so it is placed by the position of its body
	for _, defined := range declaredInOrder(declared, name) {
		group := lastGroupBefore(groups, declared[defined].Root.Position())
		if group < 0 || !onlySpaceUntil(nodes, groups[group].index, declared[defined].Root.Position()) {
			continue
		}

		docstrings[defined] = groups[group].comments
		groups = slices.Delete(groups, group, group+1)
	}

	// the comment opening the asset documents the asset, and no "define" statement has claimed it
	if opening, found := openingComment(nodes); found && len(groups) > 0 && groups[0].position == opening {
		docstrings[name] = groups[0].comments
	}

	return docstrings
}

// commentGroupsOf collects the comment groups found at the top level of a tree.
func commentGroupsOf(nodes []parse.Node) []commentGroup {
	var (
		groups  []commentGroup
		current *commentGroup
	)

	closeGroup := func() {
		if current != nil {
			groups = append(groups, *current)
			current = nil
		}
	}

	for index, node := range nodes {
		switch typed := node.(type) {
		case *parse.CommentNode:
			if current == nil {
				current = &commentGroup{position: typed.Position()}
			}

			current.comments = append(current.comments, commentText(typed.Text))
			current.index = index

		case *parse.TextNode:
			// a blank line starts a new group, and anything that is not white space ends one
			if len(bytes.TrimSpace(typed.Text)) > 0 || bytes.Count(typed.Text, []byte("\n")) > 1 {
				closeGroup()
			}

		default:
			closeGroup()
		}
	}
	closeGroup()

	return groups
}

// lastGroupBefore returns the group closest to a position, or -1 when every group comes after it.
func lastGroupBefore(groups []commentGroup, position parse.Pos) int {
	found := -1
	for index, group := range groups {
		if group.position >= position {
			break
		}

		found = index
	}

	return found
}

// onlySpaceUntil reports whether nothing but white space stands between a node and a position.
func onlySpaceUntil(nodes []parse.Node, from int, until parse.Pos) bool {
	for _, node := range nodes[from+1:] {
		if node.Position() >= until {
			return true
		}

		text, isText := node.(*parse.TextNode)
		if !isText || len(bytes.TrimSpace(text.Text)) > 0 {
			return false
		}
	}

	return true
}

// openingComment returns the position of the comment a tree opens with, when it opens with one.
//
// A comment that content comes before documents nothing, so the leading group is the documentation
// of the asset only when it opens it.
func openingComment(nodes []parse.Node) (parse.Pos, bool) {
	for _, node := range nodes {
		switch typed := node.(type) {
		case *parse.CommentNode:
			return typed.Position(), true

		case *parse.TextNode:
			if len(bytes.TrimSpace(typed.Text)) > 0 {
				return 0, false
			}

		default:
			return 0, false
		}
	}

	return 0, false
}

// declaredInOrder returns the names an asset declares with a "define" statement, by position.
func declaredInOrder(declared map[string]*parse.Tree, assetName string) []string {
	names := make([]string, 0, len(declared))
	for name := range declared {
		if name == assetName {
			continue // the tree of the asset itself, which the parser adds to the set
		}

		names = append(names, name)
	}

	slices.SortFunc(names, func(a, b string) int {
		return int(declared[a].Root.Position()) - int(declared[b].Root.Position())
	})

	return names
}

// commentText strips the marks of a template comment and trims the white space around it.
func commentText(text string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/"))
}
