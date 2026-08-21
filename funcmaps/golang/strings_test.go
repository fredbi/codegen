// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestStringsMap(t *testing.T) {
	t.Parallel()

	m := stringsBase()

	const expectedSymbols = 10
	t.Run(fmt.Sprintf("stringsBase should contain %d functions", expectedSymbols), func(*testing.T) {
		require.Len(t, m, expectedSymbols)
	})

	t.Run("stringsBase should contain the expected functions", func(*testing.T) {
		symbols := []string{
			"cleanPath",
			"contains",
			"hasPrefix",
			"joinFilePath",
			"joinPath",
			"json",
			"pluralizeFirstWord",
			"prettyjson",
			"stringContains",
			"trimSpace",
		}
		for _, symbol := range symbols {
			assert.MapContainsT(t, m, symbol)
		}
	})
}

func TestAsJSON(t *testing.T) {
	t.Parallel()

	t.Run("asJSON and prettyJSON should expose signature func(any) (string,error)", func(t *testing.T) {
		t.Parallel()

		m := stringsBase()

		asJSON, ok := m["json"].(func(any) (string, error))
		require.TrueT(t, ok)
		require.NotNil(t, asJSON)

		asPrettyJSON, ok := m["prettyjson"].(func(any) (string, error))
		require.TrueT(t, ok)
		require.NotNil(t, asPrettyJSON)
	})

	t.Run("asJSON and prettyJSON should jsonify anything that is serializable to JSON", func(t *testing.T) {
		t.Parallel()

		for _, jsonFunc := range []func(any) (string, error){
			asJSON,
			asPrettyJSON,
		} {
			res, err := jsonFunc(struct {
				A string `json:"a"`
				B int
			}{A: "good", B: 3})
			require.NoError(t, err)
			assert.JSONEqT(t, `{"a":"good","B":3}`, res)

			_, err = jsonFunc(struct {
				A string `json:"a"`
				B func() string
			}{A: "good", B: func() string { return "" }})
			require.Error(t, err)
		}
	})
}

func TestPluralizeFirstWord(t *testing.T) {
	t.Parallel()

	t.Run("pluralizeFirstWord should expose signature func(string) string", func(t *testing.T) {
		t.Parallel()

		m := stringsBase()

		pluralize, ok := m["pluralizeFirstWord"].(func(string) string)
		require.True(t, ok)
		require.NotNil(t, pluralize)
	})

	t.Run("pluralizeFirstWord should plurarize an English word using inflect", func(t *testing.T) {
		t.Parallel()

		assert.EqualT(t, "ponies of the round table", pluralizeFirstWord("pony of the round table"))
		assert.EqualT(t, "dwarves", pluralizeFirstWord("dwarf"))
		assert.EqualT(t, "", pluralizeFirstWord(""))
	})
}
