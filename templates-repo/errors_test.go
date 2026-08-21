// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestSentinel(t *testing.T) {
	t.Run("should read as the text it is declared with", func(t *testing.T) {
		assert.Equal(t, "template repository", ErrTemplateRepo.Error())
	})

	t.Run("should be found through a chain of wrapped errors", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrTemplateRepo))

		require.ErrorIs(t, wrapped, ErrTemplateRepo)
	})

	t.Run("should compare by value, so a copy is the same sentinel", func(t *testing.T) {
		// this is what lets it be a constant: nothing holds an address anyone could rebind
		copied := ErrTemplateRepo

		assert.Equal(t, ErrTemplateRepo, copied)
		require.ErrorIs(t, fmt.Errorf("wrapped: %w", copied), ErrTemplateRepo)
	})

	t.Run("should leave the cause reachable alongside it", func(t *testing.T) {
		_, err := New(FromFS(fstest.MapFS{
			"broken.gotmpl": {Data: []byte(`{{ this does not parse`)},
		}, ""))

		require.ErrorIs(t, err, ErrTemplateRepo)
		assert.ErrorContains(t, err, `function "this" not defined`)
	})

	t.Run("should not match an error of another package", func(t *testing.T) {
		assert.False(t, errors.Is(errors.New("template repository"), ErrTemplateRepo),
			"the text alone does not make a sentinel")
	})
}
