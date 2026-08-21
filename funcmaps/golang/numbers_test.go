// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/go-openapi/swag/conv"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestNumbersMap(t *testing.T) {
	t.Parallel()

	m := numbersBase()

	const expectedSymbols = 2
	t.Run(fmt.Sprintf("numbersBase should contain %d functions", expectedSymbols), func(*testing.T) {
		require.Len(t, m, expectedSymbols)
	})

	t.Run("numberBase should contain the expected functions", func(*testing.T) {
		symbols := []string{
			"isInteger",
			"gt0",
		}
		for _, symbol := range symbols {
			assert.MapContainsT(t, m, symbol)
		}
	})
}

func TestIsInteger(t *testing.T) {
	t.Parallel()

	t.Run("isInteger should expose signature func(any) bool", func(t *testing.T) {
		t.Parallel()

		m := numbersBase()
		isInteger, ok := m["isInteger"].(func(any) bool)
		require.TrueT(t, ok)
		require.NotNil(t, isInteger)
	})

	t.Run("isInteger should detect integer values", func(t *testing.T) {
		t.Parallel()

		for _, anInteger := range []any{
			int8(4),
			int16(4),
			int32(4),
			int64(4),
			int(4),
			conv.Pointer(int(4)),
			conv.Pointer(int32(4)),
			conv.Pointer(int64(4)),
			conv.Pointer(uint(4)),
			conv.Pointer(uint32(4)),
			conv.Pointer(uint64(4)),
			float32(12),
			float64(12),
			conv.Pointer(float32(12)),
			conv.Pointer(float64(12)),
			"12",
			conv.Pointer("12"),
			big.NewInt(12),
			big.NewFloat(12),
			big.NewRat(12, 1),
		} {
			val := anInteger
			require.Truef(t, isInteger(val), "expected %#v to be detected an integer value", val)
		}
	})

	t.Run("isInteger should detect non-integer values", func(t *testing.T) {
		t.Parallel()

		var (
			nilString *string
			nilInt    *int
			nilFloat  *float32
		)

		for _, notAnInteger := range []any{
			float32(12.5),
			float64(12.5),
			conv.Pointer(float32(12.5)),
			conv.Pointer(float64(12.5)),
			[]string{"a"},
			struct{}{},
			nil,
			map[string]int{"a": 1},
			"abc",
			"2.34",
			conv.Pointer("2.34"),
			nilString,
			nilInt,
			nilFloat,
			big.NewFloat(12.5),
			big.NewRat(12, 5),
		} {
			val := notAnInteger
			require.Falsef(t, isInteger(val), "did not expect %#v to be detected an integer value", val)
		}
	})
}

func TestGt0(t *testing.T) {
	t.Parallel()

	t.Run("gt0 should expose signature func(any) bool", func(t *testing.T) {
		t.Parallel()

		m := numbersBase()
		gt0, ok := m["isInteger"].(func(any) bool)
		require.TrueT(t, ok)
		require.NotNil(t, gt0)
	})

	t.Run("gt0 should work with any numeral", func(t *testing.T) {
		t.Parallel()

		// TODO more cases
		require.TrueT(t, gt0(conv.Pointer(int64(1))))
		require.FalseT(t, gt0(conv.Pointer(int64(0))))
		require.FalseT(t, gt0(nil))
	})
}
