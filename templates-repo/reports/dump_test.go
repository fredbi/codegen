// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"strings"
	"testing"
	"text/template"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func documented() Documentation {
	return Documentation{Assets: []Asset{{
		Path: "schema.gotmpl",
		Templates: []Template{{
			Name:         "schema",
			Doc:          []string{"schema renders a model."},
			Reads:        []string{".GoName", ".Properties[].Name"},
			Funcs:        []string{"pascalize"},
			Dependencies: []Dependency{{Name: "schemaBody", Data: ".", Folded: 3}},
			UsedBy:       []string{"model"},
		}, {
			Name:  "schemaBody",
			Inner: true,
			Empty: true,
		}},
	}}}
}

func TestDump(t *testing.T) {
	t.Run("should write markdown by default", func(t *testing.T) {
		var out strings.Builder
		require.NoError(t, Dump(&out, documented()))

		document := out.String()
		assert.Contains(t, document, "## schema.gotmpl")
		assert.Contains(t, document, "### schema")
		assert.Contains(t, document, "schema renders a model.")
		assert.Contains(t, document, "`.GoName`")
		assert.Contains(t, document, "**Called by** [model]")
		assert.Contains(t, document, "This template renders nothing.")
	})

	t.Run("should lay a document out as the caller asks", func(t *testing.T) {
		var out strings.Builder
		require.NoError(t, Dump(&out, documented(),
			WithTemplate(`{{ range .Assets }}{{ range .Templates }}{{ upper .Name }} {{ end }}{{ end }}`),
			WithFuncMap(template.FuncMap{"upper": strings.ToUpper}),
		))

		assert.Equal(t, "SCHEMA SCHEMABODY ", out.String())
	})

	t.Run("should report a layout that does not parse", func(t *testing.T) {
		err := Dump(&strings.Builder{}, documented(), WithTemplate(`{{ if }}`))

		require.ErrorIs(t, err, ErrReport)
		assert.ErrorContains(t, err, "could not parse")
	})

	t.Run("should report a layout naming what a document does not hold", func(t *testing.T) {
		err := Dump(&strings.Builder{}, documented(), WithTemplate(`{{ .Nowhere }}`))

		require.ErrorIs(t, err, ErrReport)
		assert.ErrorContains(t, err, "could not write")
	})

	t.Run("should refuse an empty layout", func(t *testing.T) {
		err := Dump(&strings.Builder{}, documented(), WithTemplate("  "))

		require.ErrorIs(t, err, ErrReport)
	})
}

func TestWeigh(t *testing.T) {
	t.Run("should tell the fields a fold hangs from apart by weight", func(t *testing.T) {
		weights := weigh([]string{
			".Properties[].Name", ".Properties[].Type", ".Properties[].GoName",
			".Items.Name", ".Items.Type",
			".Name",
		})

		assert.Equal(t, []Root{{Field: ".Properties", Paths: 3}, {Field: ".Items", Paths: 2}}, weights.Heavy)
		assert.Equal(t, []string{".Name"}, weights.Single)
	})

	t.Run("should hold nothing for no path at all", func(t *testing.T) {
		weights := weigh(nil)

		assert.Empty(t, weights.Heavy)
		assert.Empty(t, weights.Single)
	})
}

func TestAnchor(t *testing.T) {
	assert.Equal(t, "schemagotmpl", anchor("schema.gotmpl"))
	assert.Equal(t, "server-parameter", anchor("Server Parameter"))
	assert.Equal(t, "a_b-1", anchor("A_B 1"))
}

func TestPlural(t *testing.T) {
	assert.Equal(t, "1 path", plural(1, "path"))
	assert.Equal(t, "2 paths", plural(2, "path"))
	assert.Equal(t, "0 accesses", plural(0, "access"))
}
