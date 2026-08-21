// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// makeTemplateDir writes a few templates in a temporary directory and returns its path.
func makeTemplateDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "validation"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "model.gotmpl"), []byte("model"), 0o600))
	require.NoError(t,
		os.WriteFile(filepath.Join(dir, "validation", "primitive.gotmpl"), []byte("primitive"), 0o600),
	)

	return dir
}

func TestFromFS(t *testing.T) {
	assets := fstest.MapFS{
		"primitive.gotmpl": {Data: []byte("primitive")},
		"format.gotmpl":    {Data: []byte("format")},
	}

	t.Run("should mount at the top of the tree", func(t *testing.T) {
		for _, mount := range []string{"", ".", "/"} {
			r, err := New(FromFS(assets, mount))
			require.NoErrorf(t, err, "mount point %q", mount)

			assert.Equal(t, []string{"format", "primitive"}, slices.Collect(r.Names()))
		}
	})

	t.Run("should mount under a directory of the tree", func(t *testing.T) {
		r, err := New(FromFS(assets, "validation"))
		require.NoError(t, err)

		assert.Equal(t, []string{"validationFormat", "validationPrimitive"}, slices.Collect(r.Names()))
	})

	t.Run("should mount several sources at different places", func(t *testing.T) {
		r, err := New(
			FromFS(assets, "validation"),
			FromFS(fstest.MapFS{"primitive.gotmpl": {Data: []byte("client")}}, "client"),
		)
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"clientPrimitive", "validationFormat", "validationPrimitive"},
			slices.Collect(r.Names()),
		)
		assert.Equal(t, "client", render(t, r, "clientPrimitive"))
		assert.Equal(t, "primitive", render(t, r, "validationPrimitive"))
	})

	t.Run("should report a nil file system", func(t *testing.T) {
		_, err := New(FromFS(nil, ""))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should report an invalid mount point", func(t *testing.T) {
		_, err := New(FromFS(assets, "../escape"))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
	})
}

func TestFromDir(t *testing.T) {
	dir := makeTemplateDir(t)

	t.Run("should read a local directory", func(t *testing.T) {
		r, err := New(FromDir(dir, ""))
		require.NoError(t, err)

		assert.Equal(t, []string{"model", "validationPrimitive"}, slices.Collect(r.Names()))
		assert.Equal(t, "primitive", render(t, r, "validationPrimitive"))
	})

	t.Run("should mount a local directory under the tree", func(t *testing.T) {
		r, err := New(FromDir(dir, "local"))
		require.NoError(t, err)

		assert.Equal(t, []string{"localModel", "localValidationPrimitive"}, slices.Collect(r.Names()))
	})

	t.Run("should read the directory once, when the repository is built", func(t *testing.T) {
		r, err := New(FromDir(dir, ""))
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(filepath.Join(dir, "model.gotmpl"), []byte("edited"), 0o600))

		assert.Equal(t, "model", render(t, r, "model"), "the repository holds what it read")

		fresh, err := New(FromDir(dir, ""))
		require.NoError(t, err)
		assert.Equal(t, "edited", render(t, fresh, "model"), "a new repository reads the directory again")
	})

	t.Run("should report a directory it cannot read", func(t *testing.T) {
		_, err := New(FromDir(filepath.Join(dir, "nowhere"), ""))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
	})

	t.Run("should report a path that is not a directory", func(t *testing.T) {
		_, err := New(FromDir(filepath.Join(dir, "model.gotmpl"), ""))

		require.Error(t, err)
		assert.ErrorContains(t, err, "is not a directory")
	})
}

func TestFromTemplate(t *testing.T) {
	t.Run("should register a template held in memory", func(t *testing.T) {
		r, err := New(FromTemplate("model.gotmpl", []byte("model")))
		require.NoError(t, err)

		assert.Equal(t, []string{"model"}, slices.Collect(r.Names()))
	})

	t.Run("should name a path like any other asset", func(t *testing.T) {
		r, err := New(FromTemplate("validation/primitive.gotmpl", []byte("primitive")))
		require.NoError(t, err)

		assert.Equal(t, []string{"validationPrimitive"}, slices.Collect(r.Names()))
	})

	t.Run("should register whatever the extension", func(t *testing.T) {
		r, err := New(FromTemplate("model", []byte("model")))
		require.NoError(t, err)

		assert.Equal(t, []string{"model"}, slices.Collect(r.Names()))
	})

	t.Run("should report an invalid name", func(t *testing.T) {
		for _, name := range []string{"", "/", "../escape", "."} {
			_, err := New(FromTemplate(name, []byte("nope")))

			require.Errorf(t, err, "name %q", name)
			assert.ErrorIs(t, err, ErrTemplateRepo)
		}
	})

	t.Run("should report a template that does not parse", func(t *testing.T) {
		_, err := New(FromTemplate("broken.gotmpl", []byte(`{{ if }}`)))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
		assert.ErrorContains(t, err, "broken.gotmpl")
	})
}

func TestWithExtensions(t *testing.T) {
	assets := fstest.MapFS{
		"model.gotmpl": {Data: []byte("gotmpl")},
		"other.tmpl":   {Data: []byte("tmpl")},
	}

	t.Run("should recognize only the declared extensions", func(t *testing.T) {
		r, err := New(FromFS(assets, ""), WithExtensions(".tmpl"))
		require.NoError(t, err)

		assert.Equal(t, []string{"other"}, slices.Collect(r.Names()))
	})

	t.Run("should recognize several extensions", func(t *testing.T) {
		r, err := New(FromFS(assets, ""), WithExtensions(".gotmpl", ".tmpl"))
		require.NoError(t, err)

		assert.Equal(t, []string{"model", "other"}, slices.Collect(r.Names()))
	})

	t.Run("should report an empty list of extensions", func(t *testing.T) {
		_, err := New(FromFS(assets, ""), WithExtensions())

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTemplateRepo)
	})
}

func TestTemplateName(t *testing.T) {
	settings, err := makeOptions(nil)
	require.NoError(t, err)

	// the paths of the template set this package was extracted for, plus a few shapes around them
	for _, testCase := range []struct {
		assetPath string
		expected  string
	}{
		{"model.gotmpl", "model"},
		{"validation/primitive.gotmpl", "validationPrimitive"},
		{"server/parameter.gotmpl", "serverParameter"},
		{"validation/multipleOf.gotmpl", "validationMultipleOf"},
		{"swagger_json_embed.gotmpl", "swaggerJsonEmbed"},
		{"contrib/stratoscale/client/client.gotmpl", "contribStratoscaleClientClient"},
		{"serializers/additionalpropertiesserializer.gotmpl", "serializersAdditionalpropertiesserializer"},
		{"simpleschema/defaultsinit.gotmpl", "simpleschemaDefaultsinit"},
		{"some-kebab/name.gotmpl", "someKebabName"},
		{"http/api_v2.gotmpl", "httpApiV2"},
		{"no-extension", "noExtension"},
	} {
		assert.Equalf(t, testCase.expected, settings.templateName(testCase.assetPath),
			"naming %q", testCase.assetPath)
	}
}
