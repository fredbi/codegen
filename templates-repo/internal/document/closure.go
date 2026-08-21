// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package document

import "slices"

// Transitive holds the data a template reads once the templates it calls are folded into it.
//
// Paths are rooted at the data the template itself is executed on, like those of a [Contract]: the
// paths a called template reads are rebased onto the data handed to it, so a template reading
// ".GoName" that is called with ".Properties[]" contributes ".Properties[].GoName".
type Transitive struct {
	// Reads lists the data paths the template may read, itself or through the templates it calls.
	Reads []string

	// Funcs lists the func map functions reached the same way.
	Funcs []string

	// Reaches lists the templates it calls, directly or not, sorted.
	//
	// Unlike the paths, this follows a loop all the way round: a template in a loop reaches every
	// other template of that loop.
	Reaches []string

	// Unresolved counts what could not be folded in: the accesses the template itself could not
	// place, and the contract of a template called with data that could not be placed.
	Unresolved int

	// Recursive reports whether the template is in a loop, calling itself directly or through
	// others.
	//
	// The paths of a template in the loop are left out of the fold: rebasing them would hang them
	// from themselves without end. Every template of a loop is treated alike, so what the fold
	// holds does not depend on which template it started from.
	Recursive bool
}

// Closure folds every template into the templates it calls.
//
// A template is folded once, against the data it is executed on, and its result is rebased wherever
// it is called. A template no other one calls is folded all the same, since what it reads is
// answered by whatever data reaches it.
//
// Templates calling one another in a loop are folded without following the loop. Which templates
// those are is a property of the call graph, not of the order the folding happens in, so the result
// is the same however this is called.
func Closure(contracts map[string]Contract) map[string]Transitive {
	c := &closer{
		contracts: contracts,
		loops:     loopsOf(contracts),
		folded:    make(map[string]Transitive, len(contracts)),
	}

	for name := range contracts {
		c.fold(name)
	}

	return c.folded
}

// closer folds templates into one another, remembering what it has already folded.
type closer struct {
	contracts map[string]Contract
	loops     map[string]int
	folded    map[string]Transitive
}

// fold gathers what a template reads and what the templates it calls read.
//
// Skipping a call that lands in the same loop leaves what is left a graph without loops, so a
// template needs folding only once, whichever other one is folded first.
func (c *closer) fold(name string) Transitive {
	if done, found := c.folded[name]; found {
		return done
	}

	contract, declared := c.contracts[name]
	if !declared {
		return Transitive{}
	}

	reads := setOf(contract.Reads)
	funcs := setOf(contract.Funcs)
	result := Transitive{Unresolved: contract.Unresolved}

	for _, call := range contract.Calls {
		if c.sameLoop(name, call.Name) {
			result.Recursive = true

			continue
		}

		called := c.fold(call.Name)
		result.Recursive = result.Recursive || called.Recursive

		for _, function := range called.Funcs {
			funcs[function] = struct{}{}
		}

		if call.Data == "" {
			// the called template reads against data this analysis could not name
			result.Unresolved += len(called.Reads) + called.Unresolved

			continue
		}

		result.Unresolved += called.Unresolved
		for _, path := range called.Reads {
			reads[rebase(path, call.Data)] = struct{}{}
		}
	}

	result.Reads = sortedKeys(reads)
	result.Funcs = sortedKeys(funcs)
	result.Reaches = c.reaches(name)
	c.folded[name] = result

	return result
}

// sameLoop reports whether two templates call one another, directly or through others.
func (c *closer) sameLoop(caller, called string) bool {
	loop, inALoop := c.loops[caller]

	return inALoop && loop == c.loops[called]
}

// reaches lists every template reached from one, following loops all the way round.
func (c *closer) reaches(name string) []string {
	found := make(map[string]struct{})
	c.walkCalls(name, found)
	delete(found, name)

	return sortedKeys(found)
}

// walkCalls collects the templates reached from one.
func (c *closer) walkCalls(name string, found map[string]struct{}) {
	for _, call := range c.contracts[name].Calls {
		if _, seen := found[call.Name]; seen {
			continue
		}

		found[call.Name] = struct{}{}
		c.walkCalls(call.Name, found)
	}
}

// loopsOf finds the templates that call one another, directly or through others.
//
// It is Tarjan's algorithm for strongly connected components, keeping only the components holding
// a loop: every template of one is given the same number, and a template in no loop is given none.
func loopsOf(contracts map[string]Contract) map[string]int {
	t := &tarjan{
		contracts: contracts,
		index:     make(map[string]int, len(contracts)),
		low:       make(map[string]int, len(contracts)),
		onStack:   make(map[string]struct{}, len(contracts)),
		loops:     make(map[string]int),
	}

	for _, name := range sortedKeys(setOf(namesOf(contracts))) {
		if _, visited := t.index[name]; !visited {
			t.visit(name)
		}
	}

	return t.loops
}

// tarjan holds the state of the search for loops.
type tarjan struct {
	contracts map[string]Contract
	index     map[string]int
	low       map[string]int
	onStack   map[string]struct{}
	stack     []string
	loops     map[string]int
	next      int
	found     int
}

// visit explores one template and the templates it calls.
func (t *tarjan) visit(name string) {
	t.index[name] = t.next
	t.low[name] = t.next
	t.next++
	t.stack = append(t.stack, name)
	t.onStack[name] = struct{}{}

	selfCalling := false
	for _, call := range t.contracts[name].Calls {
		if call.Name == name {
			selfCalling = true
		}

		switch _, visited := t.index[call.Name]; {
		case !visited:
			if _, declared := t.contracts[call.Name]; !declared {
				continue // a template no source declares, reported by the repository itself
			}

			t.visit(call.Name)
			t.low[name] = min(t.low[name], t.low[call.Name])

		default:
			if _, stacked := t.onStack[call.Name]; stacked {
				t.low[name] = min(t.low[name], t.index[call.Name])
			}
		}
	}

	if t.low[name] != t.index[name] {
		return
	}

	t.close(name, selfCalling)
}

// close pops the component a template roots, and records it when it holds a loop.
func (t *tarjan) close(root string, selfCalling bool) {
	var component []string

	for {
		last := len(t.stack) - 1
		name := t.stack[last]
		t.stack = t.stack[:last]
		delete(t.onStack, name)
		component = append(component, name)

		if name == root {
			break
		}
	}

	// a component of one template is a loop only when that template calls itself
	if len(component) == 1 && !selfCalling {
		return
	}

	t.found++
	for _, name := range component {
		t.loops[name] = t.found
	}
}

// rebase hangs a path read by a called template from the data it was handed.
func rebase(path, data string) string {
	if data == currentData {
		return path
	}

	if path == currentData {
		return data
	}

	// a path read by the called template already opens with the dot, which is the separator here
	return data + path
}

// namesOf lists the templates a set of contracts covers.
func namesOf(contracts map[string]Contract) []string {
	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// setOf turns a list into a set.
func setOf(list []string) map[string]struct{} {
	set := make(map[string]struct{}, len(list))
	for _, item := range list {
		set[item] = struct{}{}
	}

	return set
}
