// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package golang

import (
	"maps"
	"testing"
	"text/template"

	"github.com/go-openapi/codegen/mangling"
	"github.com/go-openapi/testify/v2/assert"
)

func TestFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("Specialized sub-maps should not overlap", func(t *testing.T) {
		type crossMaps struct {
			toCheck template.FuncMap
			others  []template.FuncMap
		}

		for _, tc := range []crossMaps{
			{
				toCheck: numbersBase(),
				others: []template.FuncMap{
					stringsBase(),
					testGoMap(),
					othersBase(),
				},
			},
			{
				toCheck: stringsBase(),
				others: []template.FuncMap{
					numbersBase(),
					testGoMap(),
					othersBase(),
				},
			},
			{
				toCheck: testGoMap(),
				others: []template.FuncMap{
					stringsBase(),
					numbersBase(),
					othersBase(),
				},
			},
		} {
			for _, againstMap := range tc.others {
				for key := range maps.Keys(againstMap) {
					assert.MapNotContainsT(t, tc.toCheck, key)
				}
			}
		}
	})

	t.Run("Each submap should be merged", func(t *testing.T) {
		t.Parallel()

		mangler := mangling.MakeGoMangler()
		fm := FuncMap(mangler)
		for _, key := range []string{
			// pick just one representative from each
			"pascalize",
			"dict",
			"gt0",
			"contains",
		} {
			assert.MapContainsTf(t, fm, key, "expected funcmap key %q", key)
		}
	})
}
