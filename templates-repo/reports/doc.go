// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package reports holds what a templates repository says about itself.
//
// A repository executes templates. Describing them is a separate job, and the types that describe
// them are the bulk of what a caller would otherwise import without ever executing anything. They
// live here so that a program rendering templates imports none of it.
//
// # Usage
//
// [github.com/go-openapi/codegen/templates-repo.Repository] builds these values, and this package
// declares them and renders them:
//
//	documentation, err := repository.Documentation()
//	if err != nil {
//		return err
//	}
//
//	err = reports.Dump(w, documentation)
//
// [Dump] writes markdown by default. Pass [WithTemplate] to lay a document out otherwise, or walk
// the [Documentation] and render it however a text template cannot.
//
// # What the reports cover
//
// [Documentation] describes the templates a repository holds: the comments on each one, the data
// paths it reads, the functions it calls, and the templates it calls with the data handed to each.
// It is grouped by asset, so it follows the tree an author edits, and it is ordered throughout, so
// a document generated twice from the same templates is the same document twice.
//
// [Audit] lists what compiles and runs but still deserves a look: a template two assets declared,
// a template nothing calls, a template that renders nothing, a template calling a function carried
// by its data, and a func map entry no template calls.
package reports
