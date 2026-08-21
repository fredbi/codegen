// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestDocstringsOf(t *testing.T) {
	analysis := analyze(t,
		"{{/* asset renders a model. */}}\n"+
			"{{/* It has a second line. */}}\n"+
			"\n"+
			"{{/* macro expands a schema. */}}\n"+
			`{{define "macro"}}{{/* a note, not a docstring */}}body{{end}}`+"\n"+
			"\n"+
			"{{/* other does something else. */}}\n"+
			`{{define "other"}}other{{end}}`+"\n"+
			"content\n"+
			"{{/* a trailing note */}}\n",
	)

	t.Run("should document the asset with the group it opens with", func(t *testing.T) {
		assert.Equal(t,
			[]string{"asset renders a model.", "It has a second line."},
			analysis.Docstrings["asset"],
		)
	})

	t.Run("should document a define with the group right before it", func(t *testing.T) {
		assert.Equal(t, []string{"macro expands a schema."}, analysis.Docstrings["macro"])
		assert.Equal(t, []string{"other does something else."}, analysis.Docstrings["other"])
	})

	t.Run("should document nothing with a note placed elsewhere", func(t *testing.T) {
		assert.Len(t, analysis.Docstrings, 3)
	})

	t.Run("should keep a blank line from joining two groups", func(t *testing.T) {
		analysis := analyze(t,
			"{{/* first group. */}}\n\n{{/* second group. */}}\n"+`{{define "macro"}}body{{end}}`,
		)

		assert.Equal(t, []string{"first group."}, analysis.Docstrings["asset"])
		assert.Equal(t, []string{"second group."}, analysis.Docstrings["macro"])
	})

	t.Run("should not document an asset whose comment comes after content", func(t *testing.T) {
		analysis := analyze(t, "content\n{{/* too late */}}")

		assert.Empty(t, analysis.Docstrings)
	})

	t.Run("should let white space stand between a group and the define it documents", func(t *testing.T) {
		analysis := analyze(t, "{{/* macro expands a schema. */}}\n\n\n"+`{{define "macro"}}body{{end}}`)

		assert.Equal(t, []string{"macro expands a schema."}, analysis.Docstrings["macro"])
	})
}
