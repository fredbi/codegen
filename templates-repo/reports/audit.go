// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package reports

// Audit lists what a repository holds that deserves a second look.
//
// None of it is an error. [New] rejects what it cannot resolve, so everything here compiles and
// runs. These are the things a set of templates gets wrong quietly:
// a macro replaced by accident, a template nobody calls any more, a name that renders nothing.
//
// A function no template can resolve is not among them. Templates are parsed against the func map,
// so calling a function nothing provides fails the build.
type Audit struct {
	// Overridden lists the templates that more than one asset declared, with the definition that
	// stands and the ones it replaced.
	Overridden []Override

	// Unused lists the templates nothing else calls, ordered by name.
	//
	// A repository scoped with [WithRoots] leaves its roots out of this, since a run starts there.
	// One that keeps every template it read cannot tell an entry point from a dead template, so
	// its entry points are listed too.
	Unused []string

	// Empty lists the templates that render nothing, ordered by name. An asset holding only
	// "define" statements declares one, under its own name.
	Empty []string

	// Dynamic lists the templates invoking a function held by their data, with the "call" builtin.
	//
	// Nothing settles such a call before the template runs, so what it reaches is unknown to the
	// repository and to the documentation alike.
	Dynamic []string

	// UnusedFuncs lists the func map entries no template calls, ordered by name.
	UnusedFuncs []string
}

// Override is a template a source declared and a later one replaced.
//
// Stacking sources exists in order to override, so this is not an error. It is worth reporting
// all the same: nothing else reveals a template set that replaced a definition by accident.
type Override struct {
	// Name is the template that was declared more than once.
	Name string

	// Standing is the path of the asset whose definition the repository holds.
	Standing string

	// Replaced holds the paths of the assets whose definitions it replaced, in the order they
	// were read.
	Replaced []string
}
