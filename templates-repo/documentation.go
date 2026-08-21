// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/go-openapi/codegen/templates-repo/internal/document"
	"github.com/go-openapi/codegen/templates-repo/reports"
)

// Documentation returns the structure of the repository and the documentation of its templates.
//
// The analysis runs here rather than while the repository is built: comments are dropped from the
// trees a template executes, and the data a template reads is of no use to executing it. The
// repository holds its sources, so both are recovered by reading them again, which only a caller
// asking for documentation pays for.
//
// The result is built on demand and shared with nobody, so a caller may hold it, walk it, or
// render it in whatever format.
func (r *Repository) Documentation() (reports.Documentation, error) {
	analysed, err := r.analyse()
	if err != nil {
		return reports.Documentation{}, err
	}

	usedBy := r.reverseDependencies(analysed)

	contracts := make(map[string]document.Contract, len(analysed))
	for name, found := range analysed {
		contracts[name] = found.contract
	}
	closed := document.Closure(contracts)

	byAsset := make(map[string][]reports.Template, len(r.declarations))
	for _, name := range r.names {
		declared := r.declarations[name]
		contract := analysed[name].contract

		byAsset[declared.assetPath] = append(byAsset[declared.assetPath], reports.Template{
			Name:         name,
			Doc:          analysed[name].doc,
			Reads:        contract.Reads,
			RootReads:    contract.RootReads,
			Funcs:        contract.Funcs,
			Dependencies: dependenciesOfContract(contract, closed),
			UsedBy:       usedBy[name],
			Inner:        name != r.settings.templateName(declared.assetPath),
			Empty:        contract.Empty,
			Unresolved:   contract.Unresolved,
			Dynamic:      contract.Dynamic,
			Transitive:   transitiveOf(closed[name]),
		})
	}

	documentation := reports.Documentation{Assets: make([]reports.Asset, 0, len(byAsset))}
	for _, path := range slices.Sorted(maps.Keys(byAsset)) {
		templates := byAsset[path]

		// the template named after the asset comes first, the "define" statements follow by name
		slices.SortFunc(templates, func(a, b reports.Template) int {
			if a.Inner != b.Inner {
				if a.Inner {
					return 1
				}

				return -1
			}

			return strings.Compare(a.Name, b.Name)
		})

		documentation.Assets = append(documentation.Assets, reports.Asset{Path: path, Templates: templates})
	}

	return documentation, nil
}

// analysed holds the result of re-reading an asset, for one template name.
type analysed struct {
	doc      []string
	contract document.Contract
}

// analyse reads the assets again and keeps, per name, what the asset that declares it reported.
//
// An asset overridden by a later one still gets analysed, and its findings are then replaced, so
// the documentation stays aligned with the templates the repository actually holds.
func (r *Repository) analyse() (map[string]analysed, error) {
	found := make(map[string]analysed, len(r.names))

	for _, item := range r.assets {
		owner := r.settings.trimmedPath(item.path)

		analysis, err := document.Analyze(item.path, owner, item.data, r.settings.funcs)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", err, ErrTemplateRepo)
		}

		// the analysis reads the source again, so it sees what an author declared rather than the
		// address it landed at: each is placed back where the repository holds it
		for declaredName, contract := range analysis.Contracts {
			key := TemplateName(addressOf(owner, declaredName))
			if r.declarations[key].assetPath != item.path {
				continue // another asset declares this address
			}

			found[key] = analysed{
				doc:      analysis.Docstrings[declaredName],
				contract: r.resolvedContract(key, contract),
			}
		}
	}

	return found, nil
}

// resolvedContract names the templates a contract calls the way the repository holds them.
//
// A template refers to another one the way its author saw the tree, and the repository settled
// what that addresses when it was built. The analysis reads the source again, so it sees the
// references as written and needs the same answer.
func (r *Repository) resolvedContract(name string, contract document.Contract) document.Contract {
	resolved := r.resolutions[name]
	if len(resolved) == 0 {
		return contract
	}

	calls := make([]document.Call, 0, len(contract.Calls))
	for _, call := range contract.Calls {
		if key, found := resolved[call.Name]; found {
			call.Name = key
		}

		calls = append(calls, call)
	}

	contract.Calls = calls

	return contract
}

// transitiveOf maps a folded contract onto the exported model.
func transitiveOf(folded document.Transitive) reports.Transitive {
	return reports.Transitive{
		Reads:      folded.Reads,
		Funcs:      folded.Funcs,
		Reaches:    folded.Reaches,
		Unresolved: folded.Unresolved,
		Recursive:  folded.Recursive,
	}
}

// dependenciesOfContract turns the calls a template makes into its dependencies.
func dependenciesOfContract(contract document.Contract, closed map[string]document.Transitive) []reports.Dependency {
	dependencies := make([]reports.Dependency, 0, len(contract.Calls))
	for _, call := range contract.Calls {
		dependencies = append(dependencies, reports.Dependency{
			Name:   call.Name,
			Data:   call.Data,
			Folded: len(closed[call.Name].Reads),
		})
	}

	return dependencies
}

// reverseDependencies inverts the call graph, so that every template lists its callers.
func (r *Repository) reverseDependencies(analysed map[string]analysed) map[string][]string {
	usedBy := make(map[string][]string, len(r.names))

	for _, name := range r.names {
		for _, call := range analysed[name].contract.Calls {
			if !slices.Contains(usedBy[call.Name], name) {
				usedBy[call.Name] = append(usedBy[call.Name], name)
			}
		}
	}

	for dependency := range usedBy {
		slices.Sort(usedBy[dependency])
	}

	return usedBy
}

// Dump writes the documentation of the repository, as markdown by default.
//
// It is [reports.Dump] over what [Repository.Documentation] returns, which is the common way to
// ask. Use reports.Dump directly to lay out a document built once and rendered several ways.
func (r *Repository) Dump(w io.Writer, opts ...reports.DumpOption) error {
	documentation, err := r.Documentation()
	if err != nil {
		return err
	}

	if err := reports.Dump(w, documentation, opts...); err != nil {
		// a caller of this method matches the error of this package, whichever one reports it
		return fmt.Errorf("%w: %w", err, ErrTemplateRepo)
	}

	return nil
}
