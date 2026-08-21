// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package funcmaps

import (
	"testing"
	"text/template"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func key(k string) func() string {
	return func() string {
		return k
	}
}

func buildTestCase() (template.FuncMap, template.FuncMap) {
	a := template.FuncMap{
		"1": key("1"),
		"2": key("2"),
	}
	b := template.FuncMap{
		"2": key("x"),
		"3": key("3"),
	}

	return a, b
}

func TestMerge(t *testing.T) {
	t.Run("should merge maps, with overwrite", func(t *testing.T) {
		a, b := buildTestCase()
		m := Merge(a, b)
		assert.MapContainsT(t, m, "1")
		require.MapContainsT(t, m, "2")
		assert.MapContainsT(t, m, "3")

		fn, ok := m["2"].(func() string)
		require.TrueT(t, ok)
		assert.Equalf(t, "x", fn(), "value for key %q should have been overwritten", "2")
	})

	t.Run("should merge maps, a builtin is not overwritten", func(t *testing.T) {
		a, b := buildTestCase()
		a["eq"] = key("eq")
		b["slice"] = key("slice")

		m := Merge(a, b)

		assert.MapContainsT(t, m, "1")
		require.MapContainsT(t, m, "2")
		assert.MapContainsT(t, m, "3")
		assert.MapNotContainsT(t, m, "eq")
		assert.MapNotContainsT(t, m, "slice")

		fn, ok := m["2"].(func() string)
		require.TrueT(t, ok)
		assert.Equalf(t, "x", fn(), "value for key %q should have been overwritten", "2")
	})
}

func TestCoalesce(t *testing.T) {
	t.Run("should merge maps, with overwrite", func(t *testing.T) {
		a, b := buildTestCase()

		m := Coalesce(a, b)
		assert.MapContainsT(t, m, "1")
		require.MapContainsT(t, m, "2")
		assert.MapContainsT(t, m, "3")

		fn, ok := m["2"].(func() string)
		require.TrueT(t, ok)
		assert.Equalf(t, "2", fn(), "value for key %q should NOT have been overwritten", "2")
	})

	t.Run("should coalesce maps, a builtin is not overwritten", func(t *testing.T) {
		a, b := buildTestCase()
		a["eq"] = key("eq")
		b["slice"] = key("slice")

		m := Merge(a, b)

		assert.MapContainsT(t, m, "1")
		require.MapContainsT(t, m, "2")
		assert.MapNotContainsT(t, m, "eq")
		assert.MapNotContainsT(t, m, "slice")

		fn, ok := m["2"].(func() string)
		require.TrueT(t, ok)

		assert.Equalf(t, "x", fn(), "value for key %q should have been overwritten", "2")
	})
}
