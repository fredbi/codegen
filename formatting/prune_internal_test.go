// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package formatting

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func TestImportNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		importPath string
		expected   []string
	}{
		{
			name:       "should name a standard library package after its path",
			importPath: "strings",
			expected:   []string{"strings"},
		},
		{
			name:       "should name a package after the last element",
			importPath: "github.com/go-openapi/strfmt",
			expected:   []string{"strfmt"},
		},
		{
			name:       "should offer the directory above a major version suffix",
			importPath: "github.com/go-openapi/testify/v2",
			expected:   []string{"v2", "testify"},
		},
		{
			name:       "should keep the version itself, since a package may be named v1",
			importPath: "k8s.io/api/apps/v1",
			expected:   []string{"v1", "apps"},
		},
		{
			name:       "should drop a gopkg.in version suffix",
			importPath: "gopkg.in/yaml.v3",
			expected:   []string{"yaml"},
		},
		{
			name:       "should drop a go- prefix",
			importPath: "github.com/jessevdk/go-flags",
			expected:   []string{"flags"},
		},
		{
			name:       "should name nothing when no candidate is an identifier",
			importPath: "example.com/my-pkg",
			expected:   nil,
		},
		{
			name:       "should not name a keyword",
			importPath: "example.com/range",
			expected:   nil,
		},
	}

	for _, toPin := range tests {
		test := toPin
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, importNames(test.importPath))
		})
	}
}
