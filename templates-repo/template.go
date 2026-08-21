// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"fmt"
	"io"
	"text/template"
)

// Template is a compiled template, resolved against every other template of its [Repository].
//
// It is obtained from [Repository.Get] and cannot be built otherwise. The zero value reports an
// empty name and fails to execute.
//
// A [Template] exposes execution and nothing else, on purpose: the methods of a
// [text/template.Template] that alter a template would alter what the repository serves, for
// every holder of it. There is no ExecuteTemplate either, since resolving a name is the job of
// [Repository.Get].
//
// # Concurrency
//
// A [Template] is immutable and may be executed concurrently. Concurrent executions sharing a
// single [io.Writer] interleave their output, as they do with [text/template.Template].
type Template struct {
	tpl *template.Template
}

// Name returns the name the template is registered under, or an empty string for the zero value.
func (t Template) Name() string {
	if t.tpl == nil {
		return ""
	}

	return t.tpl.Name()
}

// Execute applies the template to data and writes the result to w.
//
// A template that refers to another one resolves it in the repository the template comes from.
// The zero [Template] reports an error.
func (t Template) Execute(w io.Writer, data any) error {
	if t.tpl == nil {
		return fmt.Errorf("zero template cannot be executed: %w", ErrTemplateRepo)
	}

	if err := t.tpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing template %q: %w: %w", t.tpl.Name(), err, ErrTemplateRepo)
	}

	return nil
}
