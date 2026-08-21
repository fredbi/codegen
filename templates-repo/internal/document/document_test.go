// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import (
	"testing"
	"text/template"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func analyze(t *testing.T, source string) Analysis {
	t.Helper()

	analysis, err := Analyze("asset.gotmpl", "asset", []byte(source), nil)
	require.NoError(t, err)

	return analysis
}

func TestContractReads(t *testing.T) {
	t.Run("should read a field of the data it runs on", func(t *testing.T) {
		contract := analyze(t, `{{ .Name }} {{ .Schema.GoType }}`).Contracts["asset"]

		assert.Equal(t, []string{".Name", ".Schema.GoType"}, contract.Reads)
	})

	t.Run("should follow the dot into a range", func(t *testing.T) {
		contract := analyze(t, `{{ range .Properties }}{{ .Name }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Properties", ".Properties[].Name"}, contract.Reads)
	})

	t.Run("should follow the dot into a with", func(t *testing.T) {
		contract := analyze(t, `{{ with .Schema }}{{ .GoType }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Schema", ".Schema.GoType"}, contract.Reads)
	})

	t.Run("should follow the dot through nested ranges", func(t *testing.T) {
		contract := analyze(t,
			`{{ range .AllOf }}{{ range .Properties }}{{ .Required }}{{ end }}{{ end }}`,
		).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".AllOf[].Properties[].Required")
	})

	t.Run("should keep the else branch at the outer dot", func(t *testing.T) {
		contract := analyze(t, `{{ range .Items }}{{ .Inner }}{{ else }}{{ .Fallback }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Fallback", ".Items", ".Items[].Inner"}, contract.Reads)
	})

	t.Run("should record the functions it calls", func(t *testing.T) {
		contract := analyze(t, `{{ pascalize (humanize .Name) }}`).Contracts["asset"]

		assert.Equal(t, []string{"humanize", "pascalize"}, contract.Funcs)
	})
}

func TestContractRootReads(t *testing.T) {
	t.Run("should record a reach back to the root from inside a range", func(t *testing.T) {
		contract := analyze(t, `{{ range .Properties }}{{ $.Package }}{{ .Name }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Package"}, contract.RootReads)
		assert.Contains(t, contract.Reads, ".Package", "a root read is a read like any other")
		assert.Contains(t, contract.Reads, ".Properties[].Name")
	})

	t.Run("should keep the root of a with at the template's own data", func(t *testing.T) {
		contract := analyze(t, `{{ with .Schema }}{{ $.Package }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Package"}, contract.RootReads)
	})
}

func TestContractCalls(t *testing.T) {
	t.Run("should record the data handed to a template", func(t *testing.T) {
		contract := analyze(t, `{{ template "docstring" . }}{{ template "schema" .Schema }}`).Contracts["asset"]

		assert.Equal(t,
			[]Call{{Name: "docstring", Data: "."}, {Name: "schema", Data: ".Schema"}},
			contract.Calls,
		)
	})

	t.Run("should record the data handed from inside a range", func(t *testing.T) {
		contract := analyze(t,
			`{{ range .AllOf }}{{ template "structfield" .Properties }}{{ end }}`,
		).Contracts["asset"]

		assert.Equal(t, []Call{{Name: "structfield", Data: ".AllOf[].Properties"}}, contract.Calls)
	})

	t.Run("should record a call with no argument as handing over its own data", func(t *testing.T) {
		contract := analyze(t, `{{ template "docstring" }}`).Contracts["asset"]

		assert.Equal(t, []Call{{Name: "docstring", Data: "."}}, contract.Calls)
	})

	t.Run("should analyse a define on its own data", func(t *testing.T) {
		analysis := analyze(t, `{{ define "macro" }}{{ .GoName }}{{ end }}{{ .Package }}`)

		assert.Equal(t, []string{".GoName"}, analysis.Contracts["macro"].Reads)
		assert.Equal(t, []string{".Package"}, analysis.Contracts["asset"].Reads)
	})
}

func TestContractUnresolved(t *testing.T) {
	t.Run("should count a dot that comes out of a function", func(t *testing.T) {
		contract := analyze(t, `{{ with pick .Schema }}{{ .GoType }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, 1, contract.Unresolved)
		assert.NotContains(t, contract.Reads, ".GoType", "the path has no known root")
	})

	t.Run("should count a read through a variable holding an unplaceable value", func(t *testing.T) {
		contract := analyze(t, `{{ $item := pick .Schema }}{{ $item.GoType }}`).Contracts["asset"]

		assert.Equal(t, 1, contract.Unresolved)
	})

	t.Run("should count a read through the index of a range", func(t *testing.T) {
		contract := analyze(t, `{{ range $index, $item := .Items }}{{ $index.Whatever }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, 1, contract.Unresolved)
	})

	t.Run("should leave a resolved template with nothing unresolved", func(t *testing.T) {
		contract := analyze(t, `{{ range .Items }}{{ .Name }}{{ end }}`).Contracts["asset"]

		assert.Zero(t, contract.Unresolved)
	})
}

func TestAnalyzeErrors(t *testing.T) {
	_, err := Analyze("broken.gotmpl", "broken", []byte(`{{ if }}`), nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "broken.gotmpl")
}

func TestContractVariables(t *testing.T) {
	t.Run("should follow a variable to the path it holds", func(t *testing.T) {
		contract := analyze(t, `{{ $ctx := .Ctx }}{{ $ctx.Title }} {{ $ctx.Description }}`).Contracts["asset"]

		assert.Equal(t, []string{".Ctx", ".Ctx.Description", ".Ctx.Title"}, contract.Reads)
		assert.Zero(t, contract.Unresolved)
	})

	t.Run("should follow a variable declared from another one", func(t *testing.T) {
		contract := analyze(t, `{{ $ctx := .Ctx }}{{ $inner := $ctx.Schema }}{{ $inner.GoType }}`).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".Ctx.Schema.GoType")
	})

	t.Run("should bind the element of a range to its variable", func(t *testing.T) {
		contract := analyze(t, `{{ range $prop := .Properties }}{{ $prop.Name }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Properties", ".Properties[].Name"}, contract.Reads)
	})

	t.Run("should bind the element of a range declaring an index too", func(t *testing.T) {
		contract := analyze(t,
			`{{ range $index, $prop := .Properties }}{{ $prop.Name }}{{ end }}`,
		).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".Properties[].Name")
	})

	t.Run("should bind the variable a with declares", func(t *testing.T) {
		contract := analyze(t, `{{ with $schema := .Schema }}{{ $schema.GoType }}{{ .Name }}{{ end }}`).Contracts["asset"]

		assert.Equal(t, []string{".Schema", ".Schema.GoType", ".Schema.Name"}, contract.Reads)
	})

	t.Run("should bind the variable an if declares", func(t *testing.T) {
		contract := analyze(t, `{{ if $schema := .Schema }}{{ $schema.GoType }}{{ end }}`).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".Schema.GoType")
	})

	t.Run("should hand the path a variable holds to a template it calls", func(t *testing.T) {
		contract := analyze(t, `{{ $ctx := .Ctx }}{{ template "docstring" $ctx }}`).Contracts["asset"]

		assert.Equal(t, []Call{{Name: "docstring", Data: ".Ctx"}}, contract.Calls)
	})

	t.Run("should range over the path a variable holds", func(t *testing.T) {
		contract := analyze(t, `{{ $all := .AllOf }}{{ range $all }}{{ .GoType }}{{ end }}`).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".AllOf[].GoType")
	})

	t.Run("should let a block shadow a variable without losing the outer one", func(t *testing.T) {
		contract := analyze(t,
			`{{ $x := .Outer }}{{ if . }}{{ $x := .Inner }}{{ $x.Leaf }}{{ end }}{{ $x.Leaf }}`,
		).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".Inner.Leaf", "the inner block reads the shadowing one")
		assert.Contains(t, contract.Reads, ".Outer.Leaf", "past the block, the outer one is back")
	})

	t.Run("should follow a variable reassigned to another path", func(t *testing.T) {
		contract := analyze(t, `{{ $x := .First }}{{ $x = .Second }}{{ $x.Leaf }}`).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".Second.Leaf")
	})

	t.Run("should keep $ bound to the data of the template", func(t *testing.T) {
		contract := analyze(t,
			`{{ range $item := .Items }}{{ $.Package }}{{ $item.Name }}{{ end }}`,
		).Contracts["asset"]

		assert.Equal(t, []string{".Package"}, contract.RootReads)
		assert.Contains(t, contract.Reads, ".Items[].Name")
	})
}

func TestContractFuncs(t *testing.T) {
	t.Run("should leave the builtins out", func(t *testing.T) {
		contract := analyze(t,
			`{{ if and (not .A) (or .B .C) }}{{ len .D }}{{ printf "%v" .E }}{{ eq .F 1 }}{{ end }}`,
		).Contracts["asset"]

		assert.Empty(t, contract.Funcs, "a caller supplies the func map, never the builtins")
		assert.Equal(t, []string{".A", ".B", ".C", ".D", ".E", ".F"}, contract.Reads)
	})

	t.Run("should record a builtin the func map shadows", func(t *testing.T) {
		const source = `{{ printf "%v" .A }}{{ len .B }}`

		plain, err := Analyze("a.gotmpl", "asset", []byte(source), nil)
		require.NoError(t, err)
		assert.Empty(t, plain.Contracts["asset"].Funcs)

		shadowing, err := Analyze("a.gotmpl", "asset", []byte(source), template.FuncMap{"printf": nil})
		require.NoError(t, err)
		assert.Equal(t, []string{"printf"}, shadowing.Contracts["asset"].Funcs,
			"a func map defining a builtin name makes calling it a dependency on the map")
	})

	t.Run("should record the functions of the func map", func(t *testing.T) {
		contract := analyze(t, `{{ printf "%q" .Field | myFunc }}`).Contracts["asset"]

		assert.Equal(t, []string{"myFunc"}, contract.Funcs)
		assert.Equal(t, []string{".Field"}, contract.Reads, "a pipeline is walked through")
	})

	t.Run("should report a template invoking a function held by its data", func(t *testing.T) {
		contract := analyze(t, `{{ call .Fn .Arg }}`).Contracts["asset"]

		assert.True(t, contract.Dynamic)
		assert.Equal(t, []string{".Arg", ".Fn"}, contract.Reads)
	})

	t.Run("should leave an ordinary template not dynamic", func(t *testing.T) {
		assert.False(t, analyze(t, `{{ .Name }}`).Contracts["asset"].Dynamic)
	})
}

func TestContractChains(t *testing.T) {
	t.Run("should read a chain hanging from a field", func(t *testing.T) {
		contract := analyze(t, `{{ (.Schema).GoType }}`).Contracts["asset"]

		assert.Equal(t, []string{".Schema.GoType"}, contract.Reads)
	})

	t.Run("should read a chain hanging from a variable", func(t *testing.T) {
		contract := analyze(t, `{{ $ctx := .Ctx }}{{ ($ctx).Title }}`).Contracts["asset"]

		assert.Contains(t, contract.Reads, ".Ctx.Title")
	})

	t.Run("should not root a chain hanging from a function at the current dot", func(t *testing.T) {
		contract := analyze(t, `{{ (index .Items 0).Name }}`).Contracts["asset"]

		assert.Equal(t, []string{".Items"}, contract.Reads, "the base is read, the chain is not a path")
		assert.NotContains(t, contract.Reads, ".Name")
		assert.Equal(t, 1, contract.Unresolved)
	})
}
