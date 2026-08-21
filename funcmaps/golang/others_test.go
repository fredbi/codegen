// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestOthersMap(t *testing.T) {
	m := othersBase()
	const expectedSymbols = 1

	t.Run(fmt.Sprintf("othersBase should contain %d functions", expectedSymbols), func(*testing.T) {
		require.Len(t, m, expectedSymbols)
	})

	t.Run("dict should expose signature func(any) bool", func(t *testing.T) {
		t.Parallel()

		d, ok := m["dict"].(func(...any) (map[string]any, error))
		require.TrueT(t, ok)
		require.NotNil(t, d)
	})

	t.Run("dict should render values as a map", func(t *testing.T) {
		t.Parallel()

		d, err := dict("a", "b", "c", "d")
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"a": "b", "c": "d"}, d)

		// odd number of arguments
		_, err = dict("a", "b", "c")
		require.Error(t, err)

		// none-string key
		_, err = dict("a", "b", 3, "d")
		require.Error(t, err)
	})
}
