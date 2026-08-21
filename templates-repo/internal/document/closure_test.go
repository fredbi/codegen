// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// contracts builds the contracts of a set of sources, keyed by template name, the way a repository
// hands them to the closure.
func contracts(t *testing.T, sources map[string]string) map[string]Contract {
	t.Helper()

	all := make(map[string]Contract)
	for name, source := range sources {
		analysis, err := Analyze(name+".gotmpl", name, []byte(source), nil)
		if err != nil {
			t.Fatalf("analysing %q: %v", name, err)
		}

		for declared, contract := range analysis.Contracts {
			all[declared] = contract
		}
	}

	return all
}

func TestClosure(t *testing.T) {
	t.Run("should rebase what a called template reads onto the data it is handed", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"caller": `{{ range .AllOf }}{{ template "field" .Properties }}{{ end }}`,
			"field":  `{{ define "field" }}{{ .GoName }}{{ .Required }}{{ end }}`,
		}))

		assert.Equal(t,
			[]string{
				".AllOf",
				".AllOf[].Properties", // the caller reads it to hand it over
				".AllOf[].Properties.GoName",
				".AllOf[].Properties.Required",
			},
			closed["caller"].Reads,
		)
		assert.Equal(t, []string{"field"}, closed["caller"].Reaches)
	})

	t.Run("should leave a template handed the current data as it is", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"caller": `{{ .Own }}{{ template "macro" . }}`,
			"macro":  `{{ define "macro" }}{{ .Shared }}{{ end }}`,
		}))

		assert.Equal(t, []string{".Own", ".Shared"}, closed["caller"].Reads)
	})

	t.Run("should fold a chain of calls", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"top":    `{{ template "middle" .Schema }}`,
			"middle": `{{ define "middle" }}{{ template "leaf" .Items }}{{ end }}`,
			"leaf":   `{{ define "leaf" }}{{ .Name }}{{ end }}`,
		}))

		assert.Equal(t, []string{".Schema", ".Schema.Items", ".Schema.Items.Name"}, closed["top"].Reads)
		assert.Equal(t, []string{"leaf", "middle"}, closed["top"].Reaches)
	})

	t.Run("should gather the functions of the templates it calls", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"caller": `{{ pascalize .Name }}{{ template "macro" .Schema }}`,
			"macro":  `{{ define "macro" }}{{ humanize .Title }}{{ end }}`,
		}))

		assert.Equal(t, []string{"humanize", "pascalize"}, closed["caller"].Funcs)
	})

	t.Run("should stop at a template that calls itself", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"schema": `{{ .Name }}{{ range .Properties }}{{ template "schema" . }}{{ end }}`,
		}))

		assert.True(t, closed["schema"].Recursive)
		assert.Equal(t, []string{".Name", ".Properties"}, closed["schema"].Reads)
	})

	t.Run("should not follow a loop between two templates", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"first":  `{{ .A }}{{ template "second" .Down }}`,
			"second": `{{ define "second" }}{{ .B }}{{ template "first" .Up }}{{ end }}`,
		}))

		assert.True(t, closed["first"].Recursive)
		assert.True(t, closed["second"].Recursive)

		assert.Equal(t, []string{".A", ".Down"}, closed["first"].Reads,
			"what the other template of the loop reads is left out")
		assert.Equal(t, []string{".B", ".Up"}, closed["second"].Reads)

		assert.Equal(t, []string{"second"}, closed["first"].Reaches,
			"the call graph still follows the loop all the way round")
	})

	t.Run("should fold a template calling into a loop from outside it", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"first":  `{{ .A }}{{ template "second" .Down }}`,
			"second": `{{ define "second" }}{{ .B }}{{ template "first" .Up }}{{ end }}`,
			"third":  `{{ .C }}{{ template "first" .Into }}`,
		}))

		assert.Equal(t, []string{".C", ".Into", ".Into.A", ".Into.Down"}, closed["third"].Reads)
		assert.True(t, closed["third"].Recursive, "it reaches a loop, so its fold is cut short too")
		assert.Equal(t, []string{"first", "second"}, closed["third"].Reaches)
	})

	t.Run("should fold the same way whichever template it starts from", func(t *testing.T) {
		sources := map[string]string{
			"first":  `{{ .A }}{{ template "second" .Down }}`,
			"second": `{{ define "second" }}{{ .B }}{{ template "first" .Up }}{{ end }}`,
			"third":  `{{ .C }}{{ template "first" .Into }}`,
		}

		// the templates are held in a map, so the fold starts wherever the range happens to begin
		reference := Closure(contracts(t, sources))
		for range 40 {
			assert.Equal(t, reference, Closure(contracts(t, sources)))
		}
	})

	t.Run("should count what a template called with unplaceable data reads", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"caller": `{{ template "macro" (index .Items 0) }}`,
			"macro":  `{{ define "macro" }}{{ .A }}{{ .B }}{{ end }}`,
		}))

		assert.NotContains(t, closed["caller"].Reads, ".A")
		assert.Equal(t, 2, closed["caller"].Unresolved, "the two paths the macro reads cannot be rebased")
		assert.Equal(t, []string{"macro"}, closed["caller"].Reaches)
	})

	t.Run("should carry the unresolved accesses of a template it calls", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"caller": `{{ template "macro" .Schema }}`,
			"macro":  `{{ define "macro" }}{{ (index .Items 0).Name }}{{ end }}`,
		}))

		assert.Equal(t, 1, closed["caller"].Unresolved)
	})

	t.Run("should close a template no other one calls", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"lonely": `{{ .Name }}`,
		}))

		assert.Equal(t, []string{".Name"}, closed["lonely"].Reads)
		assert.Empty(t, closed["lonely"].Reaches)
	})

	t.Run("should ignore a call to a template no source declares", func(t *testing.T) {
		closed := Closure(contracts(t, map[string]string{
			"caller": `{{ .Own }}{{ template "nowhere" .Schema }}`,
		}))

		assert.Equal(t, []string{".Own", ".Schema"}, closed["caller"].Reads,
			"the caller still reads the data it hands over")
		assert.Equal(t, []string{"nowhere"}, closed["caller"].Reaches)
	})
}
