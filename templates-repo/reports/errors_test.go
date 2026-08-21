// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestSentinel(t *testing.T) {
	t.Run("should read as the text it is declared with", func(t *testing.T) {
		assert.Equal(t, "templates report", ErrReport.Error())
	})

	t.Run("should be found through a chain of wrapped errors", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrReport))

		require.ErrorIs(t, wrapped, ErrReport)
	})

	t.Run("should compare by value, so a copy is the same sentinel", func(t *testing.T) {
		// this is what lets it be a constant: nothing holds an address anyone could rebind
		copied := ErrReport

		assert.Equal(t, ErrReport, copied)
		require.ErrorIs(t, fmt.Errorf("wrapped: %w", copied), ErrReport)
	})

	t.Run("should leave the cause reachable alongside it", func(t *testing.T) {
		err := Dump(&strings.Builder{}, Documentation{}, WithTemplate(`{{ if }}`))

		require.ErrorIs(t, err, ErrReport)
		assert.ErrorContains(t, err, "missing value for if")
	})

	t.Run("should not match an error of another package", func(t *testing.T) {
		assert.False(t, errors.Is(errors.New("templates report"), ErrReport),
			"the text alone does not make a sentinel")
	})
}
