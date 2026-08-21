// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cover

import (
	"strings"
	"testing"
	"text/template"
	"text/template/parse"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestLineTable(t *testing.T) {
	source := []byte("one\ntwo\n\nfour")
	table := newLineTable(source)

	t.Run("should place an offset on its line", func(t *testing.T) {
		assert.Equal(t, 1, table.at(0))
		assert.Equal(t, 1, table.at(2))
		assert.Equal(t, 2, table.at(4))
		assert.Equal(t, 3, table.at(8))
		assert.Equal(t, 4, table.at(9))
	})

	t.Run("should report how long a line is, its break left out", func(t *testing.T) {
		assert.Equal(t, 3, table.length(1))
		assert.Equal(t, 3, table.length(2))
		assert.Equal(t, 0, table.length(3), "an empty line holds nothing")
		assert.Equal(t, 4, table.length(4))
	})

	t.Run("should report nothing for a line that is not there", func(t *testing.T) {
		assert.Equal(t, 0, table.length(0))
		assert.Equal(t, 0, table.length(9))
	})

	t.Run("should leave a carriage return out of the length", func(t *testing.T) {
		windows := newLineTable([]byte("one\r\ntwo\r\n"))

		assert.Equal(t, 3, windows.length(1))
		assert.Equal(t, 2, windows.at(5), "the offsets follow the bytes of the source")
	})

	t.Run("should move an offset past the white space it opens with", func(t *testing.T) {
		spaced := newLineTable([]byte("{{ if . }}\n  shown\n"))

		assert.Equal(t, 1, spaced.at(10), "the text node opens with the line break of the line before")
		assert.Equal(t, 2, spaced.at(spaced.skipSpace(10)), "moved past it, the text is on its own line")
	})
}

func TestProfile(t *testing.T) {
	t.Run("should count a line it was told about", func(t *testing.T) {
		profile := NewProfile("example.com/x/")
		profile.register("a.gotmpl", 3, 10)

		counter := profile.counterFor("a.gotmpl", 3)
		require.NotNil(t, counter)
		counter.Add(2)

		var out strings.Builder
		require.NoError(t, profile.Flush(&out))
		assert.Equal(t, "mode: count\nexample.com/x/a.gotmpl:3.1,3.11 1 2\n", out.String())
	})

	t.Run("should know nothing of a line it was not told about", func(t *testing.T) {
		profile := NewProfile("")

		assert.Nil(t, profile.counterFor("nowhere.gotmpl", 1))
		assert.Nil(t, profile.counterFor("a.gotmpl", 4))
	})

	t.Run("should keep the first length it was given for a line", func(t *testing.T) {
		profile := NewProfile("")
		profile.register("a.gotmpl", 1, 5)
		profile.register("a.gotmpl", 1, 99)

		var out strings.Builder
		require.NoError(t, profile.Flush(&out))
		assert.Contains(t, out.String(), "a.gotmpl:1.1,1.6 1 0")
	})

	t.Run("should report what it counts", func(t *testing.T) {
		profile := NewProfile("")
		profile.register("a.gotmpl", 1, 5)
		profile.register("a.gotmpl", 2, 5)
		profile.counterFor("a.gotmpl", 1).Add(1)

		counted, reached := profile.Lines()
		assert.Equal(t, 2, counted)
		assert.Equal(t, 1, reached)
	})

	t.Run("should order a profile by asset then by line", func(t *testing.T) {
		profile := NewProfile("")
		profile.register("b.gotmpl", 1, 1)
		profile.register("a.gotmpl", 9, 1)
		profile.register("a.gotmpl", 2, 1)

		var out grabWriter
		require.NoError(t, profile.Flush(&out))
		assert.Equal(t,
			[]string{"mode: count", "a.gotmpl:2.1,2.2 1 0", "a.gotmpl:9.1,9.2 1 0", "b.gotmpl:1.1,1.2 1 0"},
			out.lines(),
		)
	})
}

func TestCallbackName(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		name := callbackName()

		assert.True(t, strings.HasPrefix(name, "cover"))
		seen[name] = struct{}{}
	}

	assert.Greater(t, len(seen), 90, "a name drawn at random rarely repeats")
}

func TestInstrument(t *testing.T) {
	const source = "{{ if . }}\nshown\n{{- end }}\n"

	parse := func(t *testing.T) map[string]*parse.Tree {
		t.Helper()

		parsed, err := template.New("page").Parse(source)
		require.NoError(t, err)

		return map[string]*parse.Tree{"page": parsed.Tree}
	}

	t.Run("should leave the trees it was given alone", func(t *testing.T) {
		trees := parse(t)
		before := trees["page"].Root.String()

		NewProfile("").Instrument("page.gotmpl", []byte(source), trees)

		assert.Equal(t, before, trees["page"].Root.String(), "the caller keeps what it passed")
	})

	t.Run("should bind the function the trees call", func(t *testing.T) {
		instrumented := NewProfile("").Instrument("page.gotmpl", []byte(source), parse(t))

		bound := instrumented.Bind()
		require.Len(t, bound, 1)
		assert.Contains(t, bound, instrumented.FuncName)
		assert.Contains(t, instrumented.Trees["page"].Root.String(), instrumented.FuncName)
	})

	t.Run("should render the counters it inserted", func(t *testing.T) {
		instrumented := NewProfile("").Instrument("page.gotmpl", []byte(source), parse(t))
		root := instrumented.Trees["page"].Root

		assert.Contains(t, root.String(), "{{"+instrumented.FuncName+" 1}}",
			"a counter renders as a call carrying its line")
		assert.Equal(t, root.String(), root.Copy().String(),
			"a copy of an instrumented tree renders the same")
	})

	t.Run("should record the lines that run, before any of them does", func(t *testing.T) {
		profile := NewProfile("")
		profile.Instrument("page.gotmpl", []byte(source), parse(t))

		counted, reached := profile.Lines()
		assert.Equal(t, 2, counted, "the if and the text it guards")
		assert.Zero(t, reached, "nothing has run yet")
	})
}

// grabWriter keeps what was written to it, line by line.
type grabWriter struct {
	written strings.Builder
}

func (w *grabWriter) Write(p []byte) (int, error) { return w.written.Write(p) }

func (w *grabWriter) lines() []string {
	return strings.Split(strings.TrimSpace(w.written.String()), "\n")
}
