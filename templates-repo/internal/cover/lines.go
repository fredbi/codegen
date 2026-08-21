// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cover

import (
	"slices"
	"unicode"
)

// lineTable places an offset in a source, and reports how long a line is.
type lineTable struct {
	source []byte

	// starts holds the offset each line begins at, so a binary search places an offset.
	starts []int
}

// newLineTable indexes the lines of a source.
func newLineTable(source []byte) *lineTable {
	starts := []int{0}
	for offset, char := range source {
		if char == '\n' {
			starts = append(starts, offset+1)
		}
	}

	return &lineTable{source: source, starts: starts}
}

// at reports the line an offset falls on, counting from one.
func (t *lineTable) at(offset int) int {
	found, exact := slices.BinarySearch(t.starts, offset)
	if exact {
		return found + 1
	}

	return found
}

// length reports how many bytes a line holds, its line break left out.
func (t *lineTable) length(line int) int {
	if line < 1 || line > len(t.starts) {
		return 0
	}

	start := t.starts[line-1]
	end := len(t.source)
	if line < len(t.starts) {
		end = t.starts[line] - 1
	}

	if end > start && t.source[end-1] == '\r' {
		end--
	}

	return end - start
}

// skipSpace moves an offset forward to the first character that renders.
//
// A text node holds the line break that precedes it, so it is placed on the line of whatever came
// before unless its offset is moved past the white space it opens with.
func (t *lineTable) skipSpace(offset int) int {
	for offset < len(t.source) && unicode.IsSpace(rune(t.source[offset])) {
		offset++
	}

	return offset
}
