// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package repo_test

import (
	"fmt"
	"os"
	"slices"
	"testing/fstest"

	repo "github.com/go-openapi/codegen/templates-repo"
)

// oneAsset declares two templates: the asset itself, and the "define" it holds.
func oneAsset() repo.Option {
	return repo.FromTemplate("server/parameter.gotmpl", []byte(
		`{{ define "bind" }}bound{{ end }}param[{{ template "bind" }}]`,
	))
}

// A template is known by three strings, and only the last one executes it.
func Example_vocabulary() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	for address, name := range repository.Addresses() {
		asset, _ := repository.AssetOf(name)
		fmt.Printf("asset %-24s address %-24s name %s\n", asset, address, name)
	}

	// Output:
	// asset server/parameter.gotmpl  address server/parameter         name serverParameter
	// asset server/parameter.gotmpl  address server/parameter/bind    name serverParameterBind
}

// Names identify the templates a repository holds, and Get takes one.
func ExampleRepository_Names() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	for name := range repository.Names() {
		fmt.Println(name)
	}

	// Output:
	// serverParameter
	// serverParameterBind
}

// Get takes a name.
func ExampleRepository_Get() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	tpl, err := repository.Get("serverParameter")
	if err != nil {
		panic(err)
	}

	_ = tpl.Execute(os.Stdout, nil)

	// Output: param[bound]
}

// Lookup takes an address, and reaches the same template as Get does by name.
func ExampleRepository_Lookup() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	tpl, err := repository.Lookup("server/parameter/bind")
	if err != nil {
		panic(err)
	}

	_ = tpl.Execute(os.Stdout, nil)

	// Output: bound
}

// NameOf recases an address into the name it answers to. It computes, and does not look up: an
// address nothing declares still yields the name it would have.
func ExampleRepository_NameOf() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	fmt.Println(repository.NameOf("server/parameter/bind"))
	fmt.Println(repository.NameOf("server/parameter.gotmpl"))
	fmt.Println(repository.NameOf("nowhere/at/all"), repository.Has("nowhereAtAll"))

	// Output:
	// serverParameterBind
	// serverParameter
	// nowhereAtAll false
}

// AddressOf goes back, and reports whether the name is declared at all.
func ExampleRepository_AddressOf() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	fmt.Println(repository.AddressOf("serverParameterBind"))
	fmt.Println(repository.AddressOf("nowhereAtAll"))

	// Output:
	// server/parameter/bind true
	//  false
}

// AssetOf names the file a template was read from, extension and all.
func ExampleRepository_AssetOf() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	fmt.Println(repository.AssetOf("serverParameterBind"))

	// Output: server/parameter.gotmpl true
}

// Roots are names, the identity Get takes, and never addresses.
func ExampleWithRoots() {
	repository, err := repo.New(oneAsset(), repo.WithRoots("serverParameter"))
	if err != nil {
		panic(err)
	}

	fmt.Println(repository.Roots())

	// an address is not a name, and saying so is an error rather than an empty repository
	_, err = repo.New(oneAsset(), repo.WithRoots("server/parameter"))
	fmt.Println(err != nil)

	// a caller holding addresses converts them first
	naming, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	scoped, err := repo.Clone(naming, repo.WithRoots(naming.NameOf("server/parameter")))
	if err != nil {
		panic(err)
	}

	fmt.Println(scoped.Roots())

	// Output:
	// [serverParameter]
	// true
	// [serverParameter]
}

// Scoping takes names alone, where Lookup takes either identity.
func ExampleRepository_NameOf_scoping() {
	repository, err := repo.New(oneAsset())
	if err != nil {
		panic(err)
	}

	// a caller holding an address converts it, then scopes
	scoped, err := repo.Clone(repository, repo.WithRoots(repository.NameOf("server/parameter")))
	if err != nil {
		panic(err)
	}

	for name := range scoped.Names() {
		fmt.Println(name)
	}

	// Output:
	// serverParameter
	// serverParameterBind
}

// SkipDirectories matches a directory's own name, at any depth, and never a path or a template
// name.
func ExampleSkipDirectories() {
	assets := fstest.MapFS{
		"model.gotmpl":                   {Data: []byte("model")},
		"contrib/mine/model.gotmpl":      {Data: []byte("mine")},
		"server/legacy/contrib/x.gotmpl": {Data: []byte("legacy")},
	}

	skipped, err := repo.New(repo.FromFS(assets, "", repo.SkipDirectories("contrib")))
	if err != nil {
		panic(err)
	}

	fmt.Println(slices.Collect(skipped.Names()))

	// a path matches no directory name, so nothing is skipped
	byPath, err := repo.New(repo.FromFS(assets, "", repo.SkipDirectories("server/legacy")))
	if err != nil {
		panic(err)
	}

	fmt.Println(slices.Collect(byPath.Names()))

	// Output:
	// [model]
	// [contribMineModel model serverLegacyContribX]
}
