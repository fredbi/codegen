// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package formatting

// Option configures [Format].
type Option func(*options)

type options struct {
	groups  []string
	goFumpt bool
}

// WithImportGroups adds one import group per prefix, between the standard library and the rest.
//
// An import belongs to the first prefix it starts with, so pass the more specific prefix first.
// Without this option the output has two groups: the standard library, then everything else.
func WithImportGroups(prefixes ...string) Option {
	return func(o *options) {
		for _, prefix := range prefixes {
			if prefix == "" {
				continue
			}
			o.groups = append(o.groups, prefix)
		}
	}
}

// WithGoFumpt applies the gofumpt rules before printing.
//
// Blank-import github.com/go-openapi/codegen/formatting/enable/gofumpt to make the rules available.
// Without it [Format] returns [ErrNoGoFumpt] rather than printing without them.
func WithGoFumpt() Option {
	return func(o *options) {
		o.goFumpt = true
	}
}

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	return o
}
