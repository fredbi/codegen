// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"strings"
	"testing"
	"testing/fstest"
	"text/template"

	"github.com/go-openapi/codegen/templates-repo/reports"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// documentedAssets exercise every placement a comment may have.
func documentedAssets() fstest.MapFS {
	return fstest.MapFS{
		"folder/documented.gotmpl": {Data: []byte(
			"{{/* documented renders a model. */}}\n" +
				"{{/* It has a second line of doc. */}}\n" +
				"\n" +
				"{{/* macro expands the body of a schema. */}}\n" +
				`{{define "macro"}}{{/* a note, not a docstring */}}body{{end}}` + "\n" +
				"\n" +
				"{{/* other does something else. */}}\n" +
				`{{define "other"}}other{{end}}` + "\n" +
				`content {{template "macro"}} {{template "other"}}` + "\n" +
				"{{/* a trailing note, documenting nothing */}}\n",
		)},
		"caller.gotmpl": {Data: []byte("{{/* caller uses the macro. */}}\n" + `{{template "folderDocumentedMacro"}}`)},
	}
}

func documentedRepository(t *testing.T) *Repository {
	t.Helper()

	r, err := New(FromFS(documentedAssets(), ""))
	require.NoError(t, err)

	return r
}

// docsOf collects the docstrings of a repository, by template name.
func docsOf(t *testing.T, r *Repository) map[string][]string {
	t.Helper()

	documentation, err := r.Documentation()
	require.NoError(t, err)

	docs := make(map[string][]string)
	for _, asset := range documentation.Assets {
		for _, tpl := range asset.Templates {
			if len(tpl.Doc) > 0 {
				docs[tpl.Name] = tpl.Doc
			}
		}
	}

	return docs
}

func TestDocstrings(t *testing.T) {
	r := documentedRepository(t)
	docstrings := docsOf(t, r)

	t.Run("should document an asset with the comments it opens with", func(t *testing.T) {
		assert.Equal(t,
			[]string{"documented renders a model.", "It has a second line of doc."},
			docstrings["folderDocumented"],
		)
	})

	t.Run("should document a define with the comments right before it", func(t *testing.T) {
		assert.Equal(t, []string{"macro expands the body of a schema."}, docstrings["folderDocumentedMacro"])
		assert.Equal(t, []string{"other does something else."}, docstrings["folderDocumentedOther"])
	})

	t.Run("should ignore a comment that documents nothing", func(t *testing.T) {
		for _, docs := range docstrings {
			for _, doc := range docs {
				assert.NotContains(t, doc, "not a docstring")
				assert.NotContains(t, doc, "documenting nothing")
			}
		}
	})

	t.Run("should record a docstring once", func(t *testing.T) {
		for name, docs := range docstrings {
			seen := make(map[string]bool, len(docs))
			for _, doc := range docs {
				assert.Falsef(t, seen[doc], "%q holds %q twice", name, doc)
				seen[doc] = true
			}
		}
	})

	t.Run("should strip the comment marks and the space around them", func(t *testing.T) {
		assert.Equal(t, "caller uses the macro.", docstrings["caller"][0])
	})

	t.Run("should not document an asset whose comment comes after content", func(t *testing.T) {
		plain, err := New(
			FromFS(fstest.MapFS{"a.gotmpl": {Data: []byte("content\n{{/* too late to be a docstring */}}")}}, ""),
		)
		require.NoError(t, err)

		assert.Empty(t, docsOf(t, plain)["a"])
	})
}

func TestDocumentation(t *testing.T) {
	r := documentedRepository(t)

	documentation, err := r.Documentation()
	require.NoError(t, err)

	t.Run("should group templates by asset, in order", func(t *testing.T) {
		require.Len(t, documentation.Assets, 2)
		assert.Equal(t, "caller.gotmpl", documentation.Assets[0].Path)
		assert.Equal(t, "folder/documented.gotmpl", documentation.Assets[1].Path)
	})

	t.Run("should list the template named after the asset first", func(t *testing.T) {
		templates := documentation.Assets[1].Templates
		require.Len(t, templates, 3)

		assert.Equal(t, "folderDocumented", templates[0].Name)
		assert.False(t, templates[0].Inner)
		assert.Equal(t,
			[]string{"folderDocumentedMacro", "folderDocumentedOther"},
			[]string{templates[1].Name, templates[2].Name},
		)
		assert.True(t, templates[1].Inner)
	})

	t.Run("should report direct dependencies and callers", func(t *testing.T) {
		templates := documentation.Assets[1].Templates

		assert.Equal(t,
			[]reports.Dependency{{Name: "folderDocumentedMacro", Data: "."}, {Name: "folderDocumentedOther", Data: "."}},
			templates[0].Dependencies,
		)
		assert.Equal(t, []string{"caller", "folderDocumented"}, templates[1].UsedBy)
		assert.Empty(t, templates[1].Dependencies)
	})

	t.Run("should recompute the documentation of a clone", func(t *testing.T) {
		clone, err := Clone(r, FromTemplate("caller.gotmpl", []byte(
			"{{/* caller has been replaced. */}}\n"+`{{template "folderDocumentedOther"}}`,
		)))
		require.NoError(t, err)

		documentation, err := clone.Documentation()
		require.NoError(t, err)

		assert.Equal(t, []string{"caller has been replaced."}, documentation.Assets[0].Templates[0].Doc)
		assert.Equal(t,
			[]reports.Dependency{{Name: "folderDocumentedOther", Data: "."}},
			documentation.Assets[0].Templates[0].Dependencies,
		)
	})
}

func TestDump(t *testing.T) {
	r := documentedRepository(t)

	t.Run("should produce the same document every time", func(t *testing.T) {
		seen := make(map[string]struct{})
		for range 20 {
			var out strings.Builder
			require.NoError(t, r.Dump(&out))
			seen[out.String()] = struct{}{}
		}

		assert.Len(t, seen, 1, "a document generated twice must be the same document twice")
	})

	t.Run("should hold an index, the docstrings and the graph", func(t *testing.T) {
		var out strings.Builder
		require.NoError(t, r.Dump(&out))
		document := out.String()

		assert.Contains(t, document, "- [folder/documented.gotmpl](#folderdocumentedgotmpl)")
		assert.Contains(t, document, "documented renders a model.")
		assert.Contains(t, document, "- `folderDocumentedMacro`, with `.`")
		assert.Contains(t, document, "**Called by** [caller](#caller), [folderDocumented](#folderdocumented)")
		assert.NotContains(t, document, "not a docstring")
		assert.NotContains(t, document, "**Folded** 0 paths", "a fold holding nothing is not reported")
	})

	t.Run("should lay the document out as the caller asks", func(t *testing.T) {
		var out strings.Builder
		require.NoError(t, r.Dump(&out,
			reports.WithTemplate(`{{ range .Assets }}{{ range .Templates }}{{ shout .Name }} {{ end }}{{ end }}`),
			reports.WithFuncMap(template.FuncMap{"shout": strings.ToUpper}),
		))

		assert.Equal(t, "CALLER FOLDERDOCUMENTED FOLDERDOCUMENTEDMACRO FOLDERDOCUMENTEDOTHER ", out.String())
	})

	t.Run("should report a dump template that does not parse", func(t *testing.T) {
		err := r.Dump(&strings.Builder{}, reports.WithTemplate(`{{ if }}`))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
		assert.ErrorContains(t, err, "dump template")
	})

	t.Run("should report an empty dump template", func(t *testing.T) {
		err := r.Dump(&strings.Builder{}, reports.WithTemplate("   "))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
	})

}

// TestEmptyBody covers an asset made of define statements alone: it declares a template under its
// own name, reachable like any other, whose body holds nothing.
func TestEmptyBody(t *testing.T) {
	r, err := New(
		FromFS(fstest.MapFS{
			"serializers/schema.gotmpl": {Data: []byte(
				"{{/* schemaSerializer writes a serializer. */}}\n" +
					`{{define "schemaSerializer"}}serializer{{end}}` + "\n",
			)},
			"model.gotmpl": {Data: []byte(`model {{template "serializersSchemaSchemaSerializer"}}`)},
		}, ""),
	)
	require.NoError(t, err)

	documentation, err := r.Documentation()
	require.NoError(t, err)

	t.Run("should still declare the template named after the asset", func(t *testing.T) {
		assert.True(t, r.Has("serializersSchema"))
		assert.Empty(t, strings.TrimSpace(render(t, r, "serializersSchema")),
			"an empty body is white space and comments, so it renders nothing of substance")
	})

	t.Run("should report the empty body of that template", func(t *testing.T) {
		templates := documentation.Assets[1].Templates
		require.Len(t, templates, 2)

		assert.Equal(t, "serializersSchema", templates[0].Name)
		assert.True(t, templates[0].Empty)
		assert.False(t, templates[0].Inner)

		assert.Equal(t, "serializersSchemaSchemaSerializer", templates[1].Name)
		assert.False(t, templates[1].Empty, "the define holds the body")
	})

	t.Run("should report a template with a body as not empty", func(t *testing.T) {
		assert.False(t, documentation.Assets[0].Templates[0].Empty)
	})

	t.Run("should say so in the document", func(t *testing.T) {
		var out strings.Builder
		require.NoError(t, r.Dump(&out))

		assert.Contains(t, out.String(),
			"This asset declares define statements only, so the template named after it renders nothing.")
	})
}

// TestTransitive covers what a template reads once the templates it calls are folded into it,
// which is what a reader wanting to feed it data actually needs.
func TestTransitive(t *testing.T) {
	r, err := New(FromFS(fstest.MapFS{
		"model.gotmpl": {Data: []byte(
			`{{ .GoName }}{{ range .Properties }}{{ template "field" . }}{{ end }}`,
		)},
		"field.gotmpl": {Data: []byte(
			`{{ define "field" }}{{ pascalize .Name }}{{ template "validation" .Schema }}{{ end }}`,
		)},
		"validation.gotmpl": {Data: []byte(`{{ define "validation" }}{{ .Maximum }}{{ end }}`)},
	}, ""), WithFuncMap(template.FuncMap{"pascalize": strings.ToUpper}))
	require.NoError(t, err)

	documentation, err := r.Documentation()
	require.NoError(t, err)

	var model reports.Template
	for _, asset := range documentation.Assets {
		for _, tpl := range asset.Templates {
			if tpl.Name == "model" {
				model = tpl
			}
		}
	}

	t.Run("should rebase what the templates it calls read", func(t *testing.T) {
		assert.Equal(t,
			[]string{
				".GoName",
				".Properties",
				".Properties[].Name",
				".Properties[].Schema",
				".Properties[].Schema.Maximum",
			},
			model.Transitive.Reads,
		)
	})

	t.Run("should keep the direct reads to what the template itself does", func(t *testing.T) {
		assert.Equal(t, []string{".GoName", ".Properties"}, model.Reads)
	})

	t.Run("should report the templates it reaches, directly or not", func(t *testing.T) {
		assert.Equal(t, []string{"field", "validation"}, model.Transitive.Reaches)
	})

	t.Run("should gather the functions reached through the calls", func(t *testing.T) {
		assert.Equal(t, []string{"pascalize"}, model.Transitive.Funcs)
		assert.Empty(t, model.Funcs, "the template itself calls none")
	})

	t.Run("should report a set of templates that loops", func(t *testing.T) {
		looping, err := New(FromFS(fstest.MapFS{
			"schema.gotmpl": {Data: []byte(
				`{{ .Name }}{{ range .Properties }}{{ template "schema" . }}{{ end }}`,
			)},
		}, ""))
		require.NoError(t, err)

		documentation, err := looping.Documentation()
		require.NoError(t, err)

		assert.True(t, documentation.Assets[0].Templates[0].Transitive.Recursive)
	})

	t.Run("should summarise the fold in the document", func(t *testing.T) {
		var out strings.Builder
		require.NoError(t, r.Dump(&out))

		document := out.String()

		assert.Contains(t, document, "**Folded** 5 paths, through 2 templates.")
		assert.Contains(t, document, "Mostly under `.Properties` (4).",
			"a field a subtree hangs from is reported with its weight")
		assert.Contains(t, document, "Read once: `.GoName`.",
			"a field read once is reported without a count of one")
		assert.Contains(t, document, "- `field`, with `.Properties[]` (3 paths)",
			"a call says how much of the fold it accounts for")
	})
}
