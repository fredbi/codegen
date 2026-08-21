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

// modelSet is a self-contained set: it declares a macro and calls it, knowing nothing of any other.
func modelSet(t *testing.T) *Repository {
	t.Helper()

	r, err := New(FromFS(fstest.MapFS{
		"schema.gotmpl": {Data: []byte(`{{define "body"}}MODEL-BODY{{end}}schema=[{{template "body"}}]`)},
		"model.gotmpl":  {Data: []byte(`model=[{{template "schema"}}]`)},
	}, ""))
	require.NoError(t, err)

	return r
}

// serverSet declares a macro under the same name as modelSet, which is what rebasing keeps apart.
func serverSet(t *testing.T) *Repository {
	t.Helper()

	r, err := New(FromFS(fstest.MapFS{
		"handler.gotmpl": {Data: []byte(`{{define "body"}}SERVER-BODY{{end}}handler=[{{template "body"}}]`)},
	}, ""))
	require.NoError(t, err)

	return r
}

func TestRebase(t *testing.T) {
	t.Run("should address every template under the base", func(t *testing.T) {
		moved, err := Rebase(modelSet(t), "models")
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"modelsModel", "modelsSchema", "modelsSchemaBody"},
			slices.Collect(moved.Names()),
		)

		address, found := moved.AddressOf("modelsSchemaBody")
		assert.True(t, found)
		assert.Equal(t, "models/schema/body", address)
	})

	t.Run("should keep what the templates refer to", func(t *testing.T) {
		// everything moved together, so a reference written before the move still finds its target
		moved, err := Rebase(modelSet(t), "models")
		require.NoError(t, err)

		assert.Equal(t, "model=[schema=[MODEL-BODY]]", render(t, moved, "modelsModel"))
	})

	t.Run("should leave the repository it derives from alone", func(t *testing.T) {
		origin := modelSet(t)
		_, err := Rebase(origin, "models")
		require.NoError(t, err)

		assert.Equal(t, "model=[schema=[MODEL-BODY]]", render(t, origin, "model"))
	})

	t.Run("should refuse a base that addresses nothing", func(t *testing.T) {
		_, err := Rebase(modelSet(t), "")
		require.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should refuse a nil repository", func(t *testing.T) {
		_, err := Rebase(nil, "models")
		require.ErrorIs(t, err, ErrTemplateRepo)
	})
}

func TestMerge(t *testing.T) {
	t.Run("should assemble sets rebased apart", func(t *testing.T) {
		models, err := Rebase(modelSet(t), "models")
		require.NoError(t, err)
		servers, err := Rebase(serverSet(t), "server")
		require.NoError(t, err)

		all, err := Merge(models, servers)
		require.NoError(t, err)

		// both sets declare a "body" macro, and each still reaches its own
		assert.Equal(t, "model=[schema=[MODEL-BODY]]", render(t, all, "modelsModel"))
		assert.Equal(t, "handler=[SERVER-BODY]", render(t, all, "serverHandler"))
	})

	t.Run("should let the last to declare an address win, and report it", func(t *testing.T) {
		first, err := New(FromTemplate("leaf.gotmpl", []byte("FIRST")))
		require.NoError(t, err)
		second, err := New(FromTemplate("leaf.gotmpl", []byte("SECOND")))
		require.NoError(t, err)

		all, err := Merge(first, second)
		require.NoError(t, err)

		assert.Equal(t, "SECOND", render(t, all, "leaf"))

		report, err := all.Audit()
		require.NoError(t, err)
		require.Len(t, report.Overridden, 1)
		assert.Equal(t, "leaf", report.Overridden[0].Name)
	})

	t.Run("should merge the functions the templates may call", func(t *testing.T) {
		first, err := New(
			FromTemplate("leaf.gotmpl", []byte(`{{ shout "x" }}{{ mine }}`)),
			WithFuncMap(template.FuncMap{
				"shout": func(string) string { return "FIRST" },
				"mine":  func() string { return "MINE" },
			}),
		)
		require.NoError(t, err)

		second, err := New(WithFuncMap(template.FuncMap{"shout": func(string) string { return "SECOND" }}))
		require.NoError(t, err)

		all, err := Merge(first, second)
		require.NoError(t, err)

		assert.Equal(t, "SECONDMINE", render(t, all, "leaf"))
	})

	t.Run("should refuse a nil repository among those it assembles", func(t *testing.T) {
		_, err := Merge(modelSet(t), nil)
		require.ErrorIs(t, err, ErrTemplateRepo)
	})
}

// A scaffolding that calls into the sets it is assembled with cannot stand on its own, so the sets
// are declared as sources of the same build rather than merged after the fact.
func TestFromRepository(t *testing.T) {
	t.Run("should assemble a scaffolding with the sets it calls into", func(t *testing.T) {
		all, err := New(
			FromTemplate("app.gotmpl", []byte(`app=[{{template "modelsModel"}} {{template "serverHandler"}}]`)),
			FromRepository(modelSet(t), "models"),
			FromRepository(serverSet(t), "server"),
		)
		require.NoError(t, err)

		assert.Equal(t, "app=[model=[schema=[MODEL-BODY]] handler=[SERVER-BODY]]", render(t, all, "app"))
	})

	t.Run("should keep each set reaching its own macros", func(t *testing.T) {
		all, err := New(
			FromRepository(modelSet(t), "models"),
			FromRepository(serverSet(t), "server"),
		)
		require.NoError(t, err)

		// both declare "body", and neither captures the other
		assert.Equal(t, "schema=[MODEL-BODY]", render(t, all, "modelsSchema"))
		assert.Equal(t, "handler=[SERVER-BODY]", render(t, all, "serverHandler"))
	})

	t.Run("should refuse a nil repository", func(t *testing.T) {
		_, err := New(FromRepository(nil, "models"))
		require.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should leave the repository it reads alone", func(t *testing.T) {
		origin := modelSet(t)
		_, err := New(FromRepository(origin, "models"))
		require.NoError(t, err)

		assert.Equal(t, "model=[schema=[MODEL-BODY]]", render(t, origin, "model"))
	})
}

func TestCoalesce(t *testing.T) {
	t.Run("should let the first to declare an address win", func(t *testing.T) {
		first, err := New(FromTemplate("leaf.gotmpl", []byte("FIRST")))
		require.NoError(t, err)
		second, err := New(
			FromTemplate("leaf.gotmpl", []byte("SECOND")),
			FromTemplate("other.gotmpl", []byte("OTHER")),
		)
		require.NoError(t, err)

		all, err := Coalesce(first, second)
		require.NoError(t, err)

		assert.Equal(t, "FIRST", render(t, all, "leaf"), "what the first holds stands")
		assert.Equal(t, "OTHER", render(t, all, "other"), "and the rest fills in")
		report, err := all.Audit()
		require.NoError(t, err)
		assert.Empty(t, report.Overridden, "nothing was replaced, so nothing is reported")
	})

	t.Run("should coalesce the functions the templates may call", func(t *testing.T) {
		first, err := New(
			FromTemplate("leaf.gotmpl", []byte(`{{ shout "x" }}`)),
			WithFuncMap(template.FuncMap{"shout": func(string) string { return "FIRST" }}),
		)
		require.NoError(t, err)

		second, err := New(WithFuncMap(template.FuncMap{"shout": func(string) string { return "SECOND" }}))
		require.NoError(t, err)

		all, err := Coalesce(first, second)
		require.NoError(t, err)

		assert.Equal(t, "FIRST", render(t, all, "leaf"))
	})
}

// The shape a package publishing templates takes: it hands out sources, not a repository, and
// whoever assembles them says where each one lands.
//
// genmodels ships model templates; genclient ships client templates that call into the models and
// cannot stand on their own. Neither knows where the other will be mounted.
func genmodelsSources(opts ...SourceOption) Option {
	return Sources(
		FromFS(fstest.MapFS{
			"schema.gotmpl":  {Data: []byte(`{{define "body"}}MODEL-BODY{{end}}schema=[{{template "body"}}]`)},
			"goModel.gotmpl": {Data: []byte(`goModel=[{{template "schema"}}]`)},
		}, "", opts...),
		FromFS(fstest.MapFS{
			"goModel/target.gotmpl": {Data: []byte(`models`)},
		}, "paths", opts...),
	)
}

func genclientSources(opts ...SourceOption) Option {
	return Sources(
		FromFS(fstest.MapFS{
			// its own macro, under the name the model set also uses
			"operation.gotmpl": {Data: []byte(`{{define "body"}}OP-BODY{{end}}operation=[{{template "body"}}]`)},
			// and a call into the model set, which only resolves once both are assembled
			"client.gotmpl": {Data: []byte(`client=[{{template "operation"}} {{template "modelsGoModel"}}]`)},
		}, "", opts...),
	)
}

func TestPublishedSources(t *testing.T) {
	t.Run("should assemble packages that know nothing of each other", func(t *testing.T) {
		templates, err := New(
			genmodelsSources(Rebased("models")),
			genclientSources(Rebased("client")),
		)
		require.NoError(t, err)

		assert.Equal(t,
			"client=[operation=[OP-BODY] goModel=[schema=[MODEL-BODY]]]",
			render(t, templates, "clientClient"),
		)
	})

	t.Run("should keep each package reaching its own macros", func(t *testing.T) {
		templates, err := New(
			genmodelsSources(Rebased("models")),
			genclientSources(Rebased("client")),
		)
		require.NoError(t, err)

		// both declare "body", and each still gets its own
		assert.Equal(t, "schema=[MODEL-BODY]", render(t, templates, "modelsSchema"))
		assert.Equal(t, "operation=[OP-BODY]", render(t, templates, "clientOperation"))
	})

	t.Run("should carry a rebase into every source a package publishes", func(t *testing.T) {
		templates, err := New(genmodelsSources(Rebased("models")))
		require.NoError(t, err)

		// the second source of the package mounts at "paths", under the base the caller asked for
		assert.True(t, templates.Has("modelsPathsGoModelTarget"))
	})

	t.Run("should let the assembler add sources of its own", func(t *testing.T) {
		templates, err := New(
			genmodelsSources(Rebased("models")),
			genclientSources(Rebased("client")),
			FromTemplate("app.gotmpl", []byte(`app=[{{template "clientClient"}}]`)),
		)
		require.NoError(t, err)

		assert.Equal(t,
			"app=[client=[operation=[OP-BODY] goModel=[schema=[MODEL-BODY]]]]",
			render(t, templates, "app"),
		)
	})

	t.Run("should mount a package wherever the assembler says", func(t *testing.T) {
		// the same packages, laid out differently: only the assembler's call changes
		templates, err := New(
			genmodelsSources(Rebased("v2/models")),
			FromTemplate("app.gotmpl", []byte(`app=[{{template "v2ModelsGoModel"}}]`)),
		)
		require.NoError(t, err)

		assert.Equal(t, "app=[goModel=[schema=[MODEL-BODY]]]", render(t, templates, "app"))
	})
}

func TestLookup(t *testing.T) {
	r, err := New(FromFS(fstest.MapFS{
		"server/parameter.gotmpl": {Data: []byte(`{{define "bind-primitive"}}BIND{{end}}param=[{{template "bind-primitive"}}]`)},
	}, ""))
	require.NoError(t, err)

	t.Run("should find a template by the address it was declared at", func(t *testing.T) {
		tpl, err := r.Lookup("server/parameter")
		require.NoError(t, err)

		var out strings.Builder
		require.NoError(t, tpl.Execute(&out, nil))
		assert.Equal(t, "param=[BIND]", out.String())
	})

	t.Run("should find a define under the asset declaring it", func(t *testing.T) {
		tpl, err := r.Lookup("server/parameter/bind-primitive")
		require.NoError(t, err)

		var out strings.Builder
		require.NoError(t, tpl.Execute(&out, nil))
		assert.Equal(t, "BIND", out.String())
	})

	t.Run("should take the extension the address may carry", func(t *testing.T) {
		_, err := r.Lookup("server/parameter.gotmpl")
		require.NoError(t, err)
	})

	t.Run("should read a backslash as a separator", func(t *testing.T) {
		_, err := r.Lookup(`server\parameter`)
		require.NoError(t, err)
	})

	t.Run("should report an address nothing is declared at", func(t *testing.T) {
		_, err := r.Lookup("server/nowhere")
		require.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should panic on an address nothing is declared at", func(t *testing.T) {
		assert.Panics(t, func() { r.MustLookup("server/nowhere") })
	})

	t.Run("should agree with Get", func(t *testing.T) {
		for address, key := range r.Addresses() {
			byAddress, err := r.Lookup(address)
			require.NoError(t, err)
			byKey, err := r.Get(key)
			require.NoError(t, err)

			assert.Equal(t, byKey.Name(), byAddress.Name())
		}
	})
}

func TestSeparators(t *testing.T) {
	t.Run("should hold the same address whatever separator a caller writes", func(t *testing.T) {
		r, err := New(
			FromTemplate(`server\parameter.gotmpl`, []byte("PARAM")),
			FromTemplate("server/other.gotmpl", []byte(`other=[{{template "parameter"}}]`)),
		)
		require.NoError(t, err)

		assert.True(t, r.Has("serverParameter"))
		assert.Equal(t, "other=[PARAM]", render(t, r, "serverOther"))
	})

	t.Run("should read a backslash in a mount point", func(t *testing.T) {
		r, err := New(FromFS(fstest.MapFS{"leaf.gotmpl": {Data: []byte("LEAF")}}, `a\b`))
		require.NoError(t, err)

		assert.True(t, r.Has("aBLeaf"))
	})
}
