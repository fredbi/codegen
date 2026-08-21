// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"text/template"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// targetedAssets declare two generation targets sharing a template, plus one no target reaches.
func targetedAssets() fstest.MapFS {
	return fstest.MapFS{
		"client.gotmpl":      {Data: []byte(`client=[{{template "shared" .}}{{template "clientCall"}}]`)},
		"client/call.gotmpl": {Data: []byte(`call`)},
		"server.gotmpl":      {Data: []byte(`server=[{{template "shared" .}}]`)},
		"shared.gotmpl":      {Data: []byte(`shared`)},
		"orphan.gotmpl":      {Data: []byte(`orphan`)},
	}
}

func TestWithRoots(t *testing.T) {
	t.Run("should keep the roots and what they reach", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""), WithRoots("client"))
		require.NoError(t, err)

		assert.Equal(t, []string{"client", "clientCall", "shared"}, slices.Collect(r.Names()))
		assert.Equal(t, "client=[sharedcall]", render(t, r, "client"))
	})

	t.Run("should prune what no root reaches", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""), WithRoots("server"))
		require.NoError(t, err)

		assert.Equal(t, []string{"server", "shared"}, slices.Collect(r.Names()))

		assert.False(t, r.Has("client"))
		assert.False(t, r.Has("orphan"))

		_, err = r.Get("orphan")
		require.ErrorIs(t, err, ErrTemplateRepo)

		_, declared := r.AssetOf("orphan")
		assert.False(t, declared)
	})

	t.Run("should keep every template when no root is named", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""))
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"client", "clientCall", "orphan", "server", "shared"},
			slices.Collect(r.Names()),
		)
	})

	t.Run("should keep a root that reaches nothing", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""), WithRoots("orphan"))
		require.NoError(t, err)

		assert.Equal(t, []string{"orphan"}, slices.Collect(r.Names()))
	})

	t.Run("should keep an inner template a root reaches", func(t *testing.T) {
		r, err := New(FromFS(fstest.MapFS{
			"schema.gotmpl": {Data: []byte(
				`{{define "schemaBody"}}body{{template "schemaType"}}{{end}}{{define "schemaType"}}type{{end}}`,
			)},
			"other.gotmpl": {Data: []byte(`other`)},
		}, ""), WithRoots("schemaSchemaBody"))
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"schemaSchemaBody", "schemaSchemaType"},
			slices.Collect(r.Names()),
		)
		assert.Equal(t, "bodytype", render(t, r, "schemaSchemaBody"))
	})

	t.Run("should follow a loop of templates without hanging", func(t *testing.T) {
		r, err := New(FromFS(fstest.MapFS{
			"schema.gotmpl": {Data: []byte(`{{if .}}{{template "schemaBody" false}}{{end}}`)},
			"schema/body.gotmpl": {Data: []byte(
				`body{{if .}}{{template "schema" false}}{{end}}`,
			)},
			"orphan.gotmpl": {Data: []byte(`orphan`)},
		}, ""), WithRoots("schema"))
		require.NoError(t, err)

		assert.Equal(t, []string{"schema", "schemaBody"}, slices.Collect(r.Names()))
		assert.Equal(t, "body", render(t, r, "schemaBody"))
	})

	t.Run("should build although a pruned template refers to an undeclared one", func(t *testing.T) {
		assets := fstest.MapFS{
			"client.gotmpl": {Data: []byte(`client`)},
			"server.gotmpl": {Data: []byte(`{{template "neverDeclared"}}`)},
		}

		_, err := New(FromFS(assets, ""))
		require.ErrorIs(t, err, ErrTemplateRepo)
		assert.Contains(t, err.Error(), "neverDeclared")

		r, err := New(FromFS(assets, ""), WithRoots("client"))
		require.NoError(t, err)

		assert.Equal(t, []string{"client"}, slices.Collect(r.Names()))
	})

	t.Run("should still report an undeclared template a root reaches", func(t *testing.T) {
		_, err := New(FromFS(fstest.MapFS{
			"client.gotmpl": {Data: []byte(`{{template "neverDeclared"}}`)},
		}, ""), WithRoots("client"))

		require.ErrorIs(t, err, ErrTemplateRepo)
		assert.Contains(t, err.Error(), "neverDeclared")
	})

	t.Run("should still parse the assets it prunes away", func(t *testing.T) {
		_, err := New(FromFS(fstest.MapFS{
			"client.gotmpl": {Data: []byte(`client`)},
			"broken.gotmpl": {Data: []byte(`{{ this does not parse`)},
		}, ""), WithRoots("client"))

		require.ErrorIs(t, err, ErrTemplateRepo)
		assert.Contains(t, err.Error(), "broken.gotmpl")
	})

	t.Run("should report a root no source declares", func(t *testing.T) {
		_, err := New(FromFS(targetedAssets(), ""), WithRoots("client", "typo", "alsoATypo"))

		require.ErrorIs(t, err, ErrTemplateRepo)
		assert.Contains(t, err.Error(), `"alsoATypo"`)
		assert.Contains(t, err.Error(), `"typo"`)
	})

	t.Run("should refuse a filter naming no root", func(t *testing.T) {
		_, err := New(FromFS(targetedAssets(), ""), WithRoots())

		require.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should refuse an unnamed root", func(t *testing.T) {
		_, err := New(FromFS(targetedAssets(), ""), WithRoots("client", "  "))

		require.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should honour an override of a pruned template", func(t *testing.T) {
		// the override is read, and stands, whether the template it replaces is retained or not
		r, err := New(
			FromFS(targetedAssets(), ""),
			FromTemplate("shared", []byte("overridden")),
			WithRoots("server"),
		)
		require.NoError(t, err)

		assert.Equal(t, "server=[overridden]", render(t, r, "server"))
	})

	t.Run("should follow the calls of the definition that stands", func(t *testing.T) {
		// the override calls a template the original does not, and pulls it in
		r, err := New(
			FromFS(targetedAssets(), ""),
			FromTemplate("shared", []byte(`shared+{{template "orphan"}}`)),
			WithRoots("server"),
		)
		require.NoError(t, err)

		assert.Equal(t, []string{"orphan", "server", "shared"}, slices.Collect(r.Names()))
		assert.Equal(t, "server=[shared+orphan]", render(t, r, "server"))
	})
}

func TestWithExtraRoots(t *testing.T) {
	t.Run("should widen the scope of a repository that has one", func(t *testing.T) {
		client, err := New(FromFS(targetedAssets(), ""), WithRoots("client"))
		require.NoError(t, err)

		both, err := Clone(client, WithExtraRoots("server"))
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"client", "clientCall", "server", "shared"},
			slices.Collect(both.Names()),
		)
	})

	t.Run("should leave a repository that keeps everything alone", func(t *testing.T) {
		full, err := New(FromFS(targetedAssets(), ""))
		require.NoError(t, err)

		still, err := Clone(full, WithExtraRoots("server"))
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"client", "clientCall", "orphan", "server", "shared"},
			slices.Collect(still.Names()),
		)
		assert.Empty(t, still.Roots())
	})

	t.Run("should keep a template added to a repository, scoped or not", func(t *testing.T) {
		// the caller writes the same thing either way, which is the point of it
		for _, scope := range [][]Option{{}, {WithRoots("server")}} {
			base, err := New(append([]Option{FromFS(targetedAssets(), "")}, scope...)...)
			require.NoError(t, err)

			mine, err := Clone(base, FromTemplate("mine", []byte("mine")), WithExtraRoots("mine"))
			require.NoError(t, err)

			assert.TrueT(t, mine.Has("mine"))
			assert.TrueT(t, mine.Has("server"), "what it was scoped to is still there")
		}
	})

	t.Run("should refuse an unnamed root", func(t *testing.T) {
		_, err := New(FromFS(targetedAssets(), ""), WithRoots("server"), WithExtraRoots(" "))

		require.ErrorIs(t, err, ErrTemplateRepo)
	})
}

func TestRoots(t *testing.T) {
	t.Run("should report no root when the repository holds everything", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""))
		require.NoError(t, err)

		assert.Empty(t, r.Roots())
	})

	t.Run("should report the roots it is scoped to", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""), WithRoots("client"), WithRoots("server", "orphan"))
		require.NoError(t, err)

		assert.Equal(t, []string{"server", "orphan"}, r.Roots())
	})

	t.Run("should not let a caller alter the scope it reports", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""), WithRoots("server"))
		require.NoError(t, err)

		r.Roots()[0] = "client"

		assert.Equal(t, []string{"server"}, r.Roots())
	})
}

func TestWithRootsOnClone(t *testing.T) {
	t.Run("should prune a clone of a whole repository", func(t *testing.T) {
		full, err := New(FromFS(targetedAssets(), ""))
		require.NoError(t, err)

		client, err := Clone(full, WithRoots("client"))
		require.NoError(t, err)

		assert.Equal(t, []string{"client", "clientCall", "shared"}, slices.Collect(client.Names()))

		// the repository it derives from is untouched
		assert.True(t, full.Has("server"))
	})

	t.Run("should carry the roots over to a clone", func(t *testing.T) {
		client, err := New(FromFS(targetedAssets(), ""), WithRoots("client"))
		require.NoError(t, err)

		patched, err := Clone(client, FromTemplate("shared", []byte("patched")))
		require.NoError(t, err)

		assert.Equal(t, []string{"client", "clientCall", "shared"}, slices.Collect(patched.Names()))
		assert.Equal(t, "client=[patchedcall]", render(t, patched, "client"))
	})

	t.Run("should rescope a clone to the roots it names", func(t *testing.T) {
		client, err := New(FromFS(targetedAssets(), ""), WithRoots("client"))
		require.NoError(t, err)

		server, err := Clone(client, WithRoots("server"))
		require.NoError(t, err)

		assert.Equal(t, []string{"server", "shared"}, slices.Collect(server.Names()))
		assert.Equal(t, []string{"server"}, server.Roots())
	})

	t.Run("should keep the pruned assets available to a clone", func(t *testing.T) {
		// pruning decides what a repository holds, not what it retains to build a clone from
		client, err := New(FromFS(targetedAssets(), ""), WithRoots("client"))
		require.NoError(t, err)

		widened, err := Clone(client, WithRoots("orphan"))
		require.NoError(t, err)

		assert.True(t, widened.Has("orphan"))
		assert.Equal(t, "orphan", render(t, widened, "orphan"))
	})
}

func TestWithRootsAndCoverage(t *testing.T) {
	t.Run("should count no line of a pruned template", func(t *testing.T) {
		r, err := New(
			FromFS(targetedAssets(), ""),
			WithRoots("server"),
			WithCoverage("example.com/gen/templates"),
		)
		require.NoError(t, err)

		require.NoError(t, r.MustGet("server").Execute(&strings.Builder{}, nil))

		var profile strings.Builder
		require.NoError(t, r.Coverage().Flush(&profile))

		assert.Contains(t, profile.String(), "server.gotmpl")
		assert.Contains(t, profile.String(), "shared.gotmpl")
		assert.NotContains(t, profile.String(), "orphan.gotmpl")
		assert.NotContains(t, profile.String(), "client.gotmpl")
	})
}

func TestWithRootsAndDocumentation(t *testing.T) {
	t.Run("should document the retained templates only", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""), WithRoots("server"))
		require.NoError(t, err)

		documentation, err := r.Documentation()
		require.NoError(t, err)

		documented := make([]string, 0, len(documentation.Assets))
		for _, item := range documentation.Assets {
			documented = append(documented, item.Path)
		}

		assert.Equal(t, []string{"server.gotmpl", "shared.gotmpl"}, documented)
	})
}

func TestNameOf(t *testing.T) {
	r, err := New(FromFS(targetedAssets(), ""))
	require.NoError(t, err)

	t.Run("should name an asset the way it declares one", func(t *testing.T) {
		assert.Equal(t, "clientCall", r.NameOf("client/call.gotmpl"))
		assert.Equal(t, "serverParameter", r.NameOf("server/parameter.gotmpl"))
		assert.Equal(t, "swaggerJsonEmbed", r.NameOf("swagger_json_embed.gotmpl"))
	})

	t.Run("should name an asset no source declares", func(t *testing.T) {
		assert.Equal(t, "neverThere", r.NameOf("never/there.gotmpl"))
	})

	t.Run("should take a name back unchanged", func(t *testing.T) {
		for name := range r.Names() {
			assert.Equalf(t, name, r.NameOf(name), "naming %q again changed it", name)
		}
	})

	t.Run("should reverse AssetOf", func(t *testing.T) {
		for name := range r.Names() {
			path, declared := r.AssetOf(name)
			require.True(t, declared)

			assert.Equal(t, name, r.NameOf(path))
		}
	})

	t.Run("should trim the extensions the repository recognizes", func(t *testing.T) {
		other, err := New(FromFS(fstest.MapFS{"model.tmpl": {Data: []byte("model")}}, ""), WithExtensions(".tmpl"))
		require.NoError(t, err)

		assert.Equal(t, "model", other.NameOf("model.tmpl"))
		assert.Equal(t, "modelDotGotmpl", other.NameOf("model.gotmpl"))
	})
}

func TestExportedTemplateName(t *testing.T) {
	t.Run("should name an asset before any repository exists", func(t *testing.T) {
		assert.Equal(t, "validationPrimitive", TemplateName("validation/primitive.gotmpl"))
		assert.Equal(t, "swaggerJsonEmbed", TemplateName("swagger_json_embed.gotmpl"))
		assert.Equal(t, "serverParameter", TemplateName("serverParameter"))
	})

	t.Run("should take the extensions it is given", func(t *testing.T) {
		assert.Equal(t, "model", TemplateName("model.tmpl", ".tmpl"))
		assert.Equal(t, "modelDotGotmpl", TemplateName("model.gotmpl", ".tmpl"))
	})

	t.Run("should agree with the repository it names for", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""))
		require.NoError(t, err)

		for name := range r.Names() {
			path, _ := r.AssetOf(name)

			assert.Equal(t, r.NameOf(path), TemplateName(path))
		}
	})
}

func TestAudit(t *testing.T) {
	t.Run("should report nothing when no name is declared twice", func(t *testing.T) {
		r, err := New(FromFS(targetedAssets(), ""))
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)
		assert.Empty(t, report.Overridden)
	})

	t.Run("should report what a later source replaced", func(t *testing.T) {
		r, err := New(
			FromFS(targetedAssets(), ""),
			FromFS(fstest.MapFS{"shared.gotmpl": {Data: []byte("theirs")}}, ""),
			FromTemplate("shared", []byte("mine")),
		)
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)
		require.Len(t, report.Overridden, 1)
		assert.Equal(t, "shared", report.Overridden[0].Name)
		assert.Equal(t, "shared", report.Overridden[0].Standing)
		assert.Equal(t, []string{"shared.gotmpl", "shared.gotmpl"}, report.Overridden[0].Replaced)
	})

	t.Run("should report an inner define a later source took over", func(t *testing.T) {
		// a define is replaced at the address it lives at, which is under the asset declaring it
		r, err := New(
			FromFS(fstest.MapFS{
				"schema.gotmpl": {Data: []byte(`{{define "helper"}}shipped{{end}}{{template "helper"}}`)},
			}, ""),
			FromFS(fstest.MapFS{
				"schema.gotmpl": {Data: []byte(`{{define "helper"}}mine{{end}}{{template "helper"}}`)},
			}, ""),
		)
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)
		require.Len(t, report.Overridden, 2)
		assert.Equal(t, "schema", report.Overridden[0].Name)
		assert.Equal(t, "schemaHelper", report.Overridden[1].Name)
		assert.Equal(t, "schema.gotmpl", report.Overridden[1].Standing)

		// and it reaches the template that was calling the one replaced
		assert.Equal(t, "mine", render(t, r, "schema"))
	})

	t.Run("should not let a caller alter what it reports", func(t *testing.T) {
		r, err := New(
			FromFS(targetedAssets(), ""),
			FromTemplate("shared", []byte("mine")),
		)
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)
		report.Overridden[0].Name = "tampered"

		again, err := r.Audit()
		require.NoError(t, err)
		assert.Equal(t, "shared", again.Overridden[0].Name)
	})

	t.Run("should say nothing of a template pruned away", func(t *testing.T) {
		r, err := New(
			FromFS(targetedAssets(), ""),
			FromTemplate("orphan", []byte("mine")),
			WithRoots("server"),
		)
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)
		assert.Empty(t, report.Overridden)
	})
}

func TestAuditReport(t *testing.T) {
	t.Run("should report what renders nothing and what calls dynamically", func(t *testing.T) {
		r, err := New(FromFS(fstest.MapFS{
			"root.gotmpl":    {Data: []byte(`root=[{{template "used"}}]`)},
			"used.gotmpl":    {Data: []byte(`used`)},
			"orphan.gotmpl":  {Data: []byte(`orphan`)},
			"blank.gotmpl":   {Data: []byte(`{{define "inner"}}inner{{end}}`)},
			"dynamic.gotmpl": {Data: []byte(`{{ call .Fn }}`)},
		}, ""))
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)

		assert.Empty(t, report.Overridden)
		assert.Equal(t, []string{"blank"}, report.Empty, "an asset of defines alone renders nothing")
		assert.Equal(t, []string{"dynamic"}, report.Dynamic)

		// nothing calls these; whether that is a dead template or an entry point, only the caller knows
		assert.Equal(t, []string{"blank", "blankInner", "dynamic", "orphan", "root"}, report.Unused)
	})

	t.Run("should find nothing unused in a repository scoped to its roots", func(t *testing.T) {
		// scoping keeps the roots and whatever they reach, so every template left is one or the other
		r, err := New(FromFS(fstest.MapFS{
			"root.gotmpl":   {Data: []byte(`root=[{{template "used"}}]`)},
			"used.gotmpl":   {Data: []byte(`used`)},
			"orphan.gotmpl": {Data: []byte(`orphan`)},
		}, ""), WithRoots("root"))
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)

		assert.Empty(t, report.Unused)
	})

	t.Run("should report the func map entries no template calls", func(t *testing.T) {
		r, err := New(
			FromTemplate("leaf.gotmpl", []byte(`{{ shout "x" }}`)),
			WithFuncMap(template.FuncMap{
				"shout":  func(string) string { return "X" },
				"unused": func() string { return "" },
				"also":   func() string { return "" },
			}),
		)
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)

		assert.Equal(t, []string{"also", "unused"}, report.UnusedFuncs)
	})

	t.Run("should count an entry point of an unscoped repository as unused", func(t *testing.T) {
		// with no root declared, nothing tells an entry point from a template that outlived its callers
		r, err := New(FromFS(fstest.MapFS{
			"root.gotmpl": {Data: []byte(`root=[{{template "used"}}]`)},
			"used.gotmpl": {Data: []byte(`used`)},
		}, ""))
		require.NoError(t, err)

		report, err := r.Audit()
		require.NoError(t, err)

		assert.Equal(t, []string{"root"}, report.Unused)
	})
}
