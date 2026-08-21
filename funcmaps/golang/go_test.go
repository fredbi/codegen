// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"testing"
	"text/template"

	"github.com/go-openapi/codegen/mangling"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestGoMap(t *testing.T) {
	const helloTitle = "Hello"
	t.Run("pascalize should use verbalise prefix", func(t *testing.T) {
		t.Parallel()

		fm := testGoMap()
		pascalize, ok := fm["pascalize"].(func(string) string)
		require.TrueT(t, ok)
		require.NotNil(t, pascalize)

		for _, tc := range []struct {
			Input    string
			Expected string
		}{
			{Expected: "One", Input: "+1"},
			{Expected: "Plus", Input: "+"},
			{Expected: "MinusOne", Input: "-1"},
			{Expected: "Empty", Input: "-"},
			{Expected: "Eight", Input: "8"},
			{Expected: "Star", Input: "*"},
			{Expected: "Slash", Input: "/"},
			{Expected: "Equal", Input: "="},
			{Expected: "PlusHello", Input: "+hello"},
			// other values from swag rules
			{Expected: "At8", Input: "@8"},
			{Expected: "Bang8", Input: "!8"},
			{Expected: "At", Input: "@"},
			// # values
			{Expected: "HashHello", Input: "#hello"},
			{Expected: "HashBangHello", Input: "#!hello"},
			{Expected: "Hash8", Input: "#8"},
			{Expected: "Hash", Input: "#"},
			// single '_'
			{Expected: "Empty", Input: "_"},
			{Expected: helloTitle, Input: "_hello"},
			// remove spaces
			{Expected: "HashHelloWorld", Input: "# hello world"},
			{Expected: "Hash8HelloWorld", Input: "# 8 hello world"},
			{Expected: "Empty", Input: ""},
		} {
			result := pascalize(tc.Input)
			assert.EqualTf(t, tc.Expected, result, "given %q, expected pascalize to yield %q, but got %q", tc.Input, tc.Expected, result)
		}
	})

	t.Run("enumName should transliterate special characters", func(t *testing.T) {
		t.Parallel()

		gomap := testGoMap()
		enumName, ok := gomap["enumName"].(func(string) string)
		require.TrueT(t, ok)
		require.NotNil(t, enumName)

		assert.EqualT(t, "TwoDotFourGhz", enumName("2.4Ghz"))
		assert.EqualT(t, "One", enumName("+1"))
		assert.EqualT(t, "ABHashC", enumName("a-b#c"))
		assert.EqualT(t, "Plain", enumName("plain"))
		assert.EqualT(t, "Equal", enumName("=="))
		assert.EqualT(t, "Match", enumName("=~"))
		assert.EqualT(t, "GreaterOrEqual", enumName(">="))
		assert.EqualT(t, "LessOrEqual", enumName("<="))
		assert.EqualT(t, "NotEqual", enumName("!="))
		assert.EqualT(t, "NotMatch", enumName("!~"))
	})

	t.Run("escapeBackicks should escape backticks in strings", func(t *testing.T) {
		t.Parallel()

		fm := testGoMap()
		escapeBackticks, ok := fm["escapeBackticks"].(func(string) string)
		require.TrueT(t, ok)
		require.NotNil(t, escapeBackticks)

		assert.EqualT(t, "no ticks", escapeBackticks("no ticks"))
		assert.EqualT(t, "has`+\"`\"+`tick", escapeBackticks("has`tick"))
	})

	t.Run("go literal", func(t *testing.T) {
		fm := testGoMap()

		fn, ok := fm["printGoLiteral"].(func(any) string)
		require.TrueT(t, ok)

		assert.EqualT(t, `"hello"`, fn("hello"))
		assert.EqualT(t, "42", fn(42))
	})

	t.Run("printImports should render imports", func(t *testing.T) {
		// empty map: returns ""
		assert.Empty(t, printImports(map[string]string{}))

		// unaliased import (name matches last path component)
		res := printImports(map[string]string{"fmt": "fmt"})
		assert.StringContainsT(t, res, `"fmt"`)

		// aliased import (name differs from last path component)
		res = printImports(map[string]string{"myalias": "github.com/example/pkg"})
		assert.StringContainsT(t, res, `myalias "github.com/example/pkg"`)
	})

	t.Run("jsonFieldTag should render a safe struct tag", func(t *testing.T) {
		t.Parallel()

		// clean names render as a raw backtick literal, byte-identical to the
		// previous hand-written template output.
		assert.EqualT(t, "`json:\"name\"`", jsonFieldTag("name", false, false))
		assert.EqualT(t, "`json:\"name,omitempty\"`", jsonFieldTag("name", true, false))
		assert.EqualT(t, "`json:\"name,omitempty,string\"`", jsonFieldTag("name", true, true))
		assert.EqualT(t, "`json:\"name,string\"`", jsonFieldTag("name", false, true))

		// a backtick in the name cannot be represented in a raw literal, so the
		// whole tag falls back to a double-quoted literal: the injected payload
		// can no longer close the tag and inject top-level Go.
		got := jsonFieldTag("evil` }; func init(){ println(\"pwned\") }; var _ = `", false, false)
		assert.EqualT(t, `"json:\"evil`+"`"+` }; func init(){ println(\\\"pwned\\\") }; var _ = `+"`"+`\""`, got)
	})

	t.Run("escapeDoubleQuoted should render the inner part of the quoted string", func(t *testing.T) {
		assert.EqualT(t, "fi`\\\"xed", escapeDoubleQuoted("fi`\"xed"))
	})
}

func testGoMap() template.FuncMap {
	mangler := mangling.MakeGoMangler()

	return goBase(mangler)
}
