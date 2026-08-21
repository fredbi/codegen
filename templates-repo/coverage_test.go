// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// coveredAssets hold every shape instrumentation has to leave alone: trim markers, a define, a
// branch never taken, a range, and a template calling another.
func coveredAssets() fstest.MapFS {
	return fstest.MapFS{
		"page.gotmpl": {Data: []byte(
			"{{- define \"row\" -}}\n" +
				"row: {{ .Name }}\n" +
				"{{- end -}}\n" +
				"{{ if .Show }}\n" +
				"shown\n" +
				"{{- range .Items }}\n" +
				"{{ template \"row\" . }}\n" +
				"{{- end }}\n" +
				"{{- else }}\n" +
				"hidden\n" +
				"{{- end }}\n",
		)},
	}
}

type coverItem struct{ Name string }

type coverData struct {
	Show  bool
	Items []coverItem
}

func renderPage(t *testing.T, r *Repository, data coverData) string {
	t.Helper()

	tpl, err := r.Get("page")
	require.NoError(t, err)

	var out strings.Builder
	require.NoError(t, tpl.Execute(&out, data))

	return out.String()
}

// TestCoverageLeavesOutputAlone guards the property everything else rests on: a repository that
// counts renders exactly what a repository that does not renders.
func TestCoverageLeavesOutputAlone(t *testing.T) {
	for _, data := range []coverData{
		{Show: true, Items: []coverItem{{"a"}, {"b"}}},
		{Show: true},
		{Show: false},
	} {
		plain, err := New(FromFS(coveredAssets(), ""))
		require.NoError(t, err)

		counting, err := New(FromFS(coveredAssets(), ""), WithCoverage("example.com/templates"))
		require.NoError(t, err)

		assert.Equal(t, renderPage(t, plain, data), renderPage(t, counting, data),
			"instrumenting must not change what a template renders")
	}
}

func TestCoverageProfile(t *testing.T) {
	r, err := New(FromFS(coveredAssets(), ""), WithCoverage("example.com/templates"))
	require.NoError(t, err)

	require.NotNil(t, r.Coverage())
	_ = renderPage(t, r, coverData{Show: true, Items: []coverItem{{"a"}, {"b"}}})

	var out strings.Builder
	require.NoError(t, r.Coverage().Flush(&out))
	profile := out.String()

	t.Logf("profile:\n%s", profile)

	t.Run("should open with the mode the counts are in", func(t *testing.T) {
		assert.True(t, strings.HasPrefix(profile, "mode: count\n"))
	})

	t.Run("should hold a line that never ran, at zero", func(t *testing.T) {
		assert.Contains(t, profile, "example.com/templates/page.gotmpl:10.1,10.7 1 0",
			"the else branch never runs, and has to be in the profile all the same")
	})

	t.Run("should count a line as many times as it ran", func(t *testing.T) {
		assert.Contains(t, profile, "example.com/templates/page.gotmpl:2.1,2.17 1 2",
			"the row template runs once per item")
	})

	t.Run("should cover a whole line, never a block of no width", func(t *testing.T) {
		for _, line := range strings.Split(strings.TrimSpace(profile), "\n")[1:] {
			assert.NotContains(t, line, ".0,", "a column of zero makes go tool cover write broken html")
		}
	})

	t.Run("should leave a line holding only a define, an end or an else out", func(t *testing.T) {
		for _, line := range []string{":1.", ":3.", ":9.", ":11."} {
			assert.NotContains(t, profile, "page.gotmpl"+line,
				"a line that runs nothing renders as plain text, the way a go declaration does")
		}
	})

	t.Run("should report what it counts", func(t *testing.T) {
		counted, reached := r.Coverage().Lines()
		assert.Positive(t, counted)
		assert.Less(t, reached, counted, "the else branch is not reached")
	})

	t.Run("should write the same profile twice", func(t *testing.T) {
		var again strings.Builder
		require.NoError(t, r.Coverage().Flush(&again))
		assert.Equal(t, profile, again.String())
	})
}

func TestCoverageOptions(t *testing.T) {
	t.Run("should leave a plain repository without a profile", func(t *testing.T) {
		r, err := New(FromFS(coveredAssets(), ""))
		require.NoError(t, err)

		assert.Nil(t, r.Coverage())
	})

	t.Run("should refuse coverage without the path the templates live under", func(t *testing.T) {
		_, err := New(FromFS(coveredAssets(), ""), WithCoverage("  "))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should yield an instrumented twin of a plain repository", func(t *testing.T) {
		plain, err := New(FromFS(coveredAssets(), ""))
		require.NoError(t, err)

		counting, err := Clone(plain, WithCoverage("example.com/templates"))
		require.NoError(t, err)

		assert.Nil(t, plain.Coverage())
		require.NotNil(t, counting.Coverage())

		_ = renderPage(t, counting, coverData{Show: true})
		_, reached := counting.Coverage().Lines()
		assert.Positive(t, reached)
	})
}

// TestCoverageConcurrent runs the counters the way a generator would, under -race.
func TestCoverageConcurrent(t *testing.T) {
	r, err := New(FromFS(coveredAssets(), ""), WithCoverage("example.com/templates"))
	require.NoError(t, err)

	const runners, runs = 16, 25

	var wg sync.WaitGroup
	wg.Add(runners)

	for range runners {
		go func() {
			defer wg.Done()

			for range runs {
				_ = renderPage(t, r, coverData{Show: true, Items: []coverItem{{"a"}}})
			}
		}()
	}

	wg.Wait()

	var out strings.Builder
	require.NoError(t, r.Coverage().Flush(&out))
	assert.Contains(t, out.String(), "page.gotmpl:5.1,5.6 1 400",
		"every run of every runner is counted")
}
