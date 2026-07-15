// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package formatting

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestGolang_MangleFileName(t *testing.T) {
	o := &Options{}
	o.Init()
	res := o.MangleFileName("aFileEndingInOsNameWindows")
	assert.FalseT(t, strings.HasSuffix(res, "_windows"))
	assert.TrueT(t, strings.HasSuffix(res, "_windows_swagger"))

	o = GolangOpts()
	res = o.MangleFileName("aFileEndingInOsNameWindows")
	assert.TrueT(t, strings.HasSuffix(res, "_windows_swagger"))
	res = o.MangleFileName("aFileEndingInOsNameWindowsAmd64")
	assert.TrueT(t, strings.HasSuffix(res, "_windows_amd64_swagger"))
	res = o.MangleFileName("aFileEndingInTest")
	assert.TrueT(t, strings.HasSuffix(res, "_test_swagger"))
}

func TestGolang_ManglePackage(t *testing.T) {
	const defaultPackage = "default"
	o := GolangOpts()

	for _, v := range []struct {
		tested       string
		expectedPath string
		expectedName string
	}{
		{tested: "", expectedPath: defaultPackage, expectedName: defaultPackage},
		{tested: "select", expectedPath: "selectpkg", expectedName: "selectpkg"}, // a package path may use a go keyword?
		{tested: "x", expectedPath: "x", expectedName: "x"},
		{tested: "a/b/c-d/e_f/g", expectedPath: "a/b/c-d/e_f/g", expectedName: "g"},
		{tested: "a/b/c-d/e_f/g-h", expectedPath: "a/b/c-d/e_f/g-h", expectedName: "h"},
		{tested: "a/b/c-d/e_f/2A", expectedPath: "a/b/c-d/e_f/2-a", expectedName: "a"},
		{tested: "a/b/c-d/e_f/#", expectedPath: "a/b/c-d/e_f/hash", expectedName: "hash"},
		{tested: "#help", expectedPath: "hash-help", expectedName: "help"},
		{tested: "vendor", expectedPath: "vendorpkg", expectedName: "vendorpkg"},
		{tested: "internal", expectedPath: "internalpkg", expectedName: "internalpkg"},
	} {
		res := o.ManglePackagePath(v.tested, defaultPackage)
		assert.EqualTf(t, v.expectedPath, res, "expected ManglePackagePath(%q) to yield %q but go %q", v.tested, v.expectedPath, res)
		res = o.ManglePackageName(v.tested, defaultPackage)
		assert.EqualTf(t, v.expectedName, res, "expected ManglePackageName(%q) to yield %q but go %q", v.tested, v.expectedName, res)
	}
}

// Go literal initializer func.
func TestGolang_SliceInitializer(t *testing.T) {
	o := GolangOpts()
	goSliceInitializer := o.ArrayInitializerFunc

	a0 := []any{"a", "b"}
	res, err := goSliceInitializer(a0)
	require.NoError(t, err)
	assert.EqualT(t, `{"a","b",}`, res)

	a1 := []any{[]any{"a", "b"}, []any{"c", "d"}}
	res, err = goSliceInitializer(a1)
	require.NoError(t, err)
	assert.EqualT(t, `{{"a","b",},{"c","d",},}`, res)

	a2 := map[string]any{"a": "y", "b": "z"}
	res, err = goSliceInitializer(a2)
	require.NoError(t, err)
	assert.EqualT(t, `{"a":"y","b":"z",}`, res)

	_, err = goSliceInitializer(struct {
		A string `json:"a"`
		B func() string
	}{A: "good", B: func() string { return "" }})
	require.Error(t, err)

	a3 := []any{}
	res, err = goSliceInitializer(a3)
	require.NoError(t, err)
	assert.EqualT(t, `{}`, res)
}

func TestGolang_Imports(t *testing.T) {
	o := GolangOpts()

	// empty map: returns ""
	assert.Empty(t, o.Imports(map[string]string{}))

	// unaliased import (name matches last path component)
	res := o.Imports(map[string]string{formatImport: formatImport})
	assert.StringContainsT(t, res, `"`+formatImport+`"`)

	// aliased import (name differs from last path component)
	res = o.Imports(map[string]string{"myalias": "github.com/example/pkg"})
	assert.StringContainsT(t, res, `myalias "github.com/example/pkg"`)
}

func TestDefaultGoFormatFunc(t *testing.T) {
	o := GolangOpts()

	src := []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }\n")
	res, err := o.FormatContent("test.go", src)
	require.NoError(t, err)
	assert.StringContainsT(t, string(res), "package main")
	assert.StringContainsT(t, string(res), `"`+formatImport+`"`)
}

func TestRelPathToRelGoPath(t *testing.T) {
	assert.EqualT(t, "", relPathToRelGoPath("/base", "."))
	assert.EqualT(t, "/sub/pkg", relPathToRelGoPath("/base", "/base/sub/pkg"))
	assert.EqualT(t, "/pkg", relPathToRelGoPath("/base", "/base/pkg"))
}

func TestCheckPrefixAndFetchRelativePath(t *testing.T) {
	ok, rel := CheckPrefixAndFetchRelativePath("/home/user/go/src/mypackage", "/home/user/go/src")
	assert.TrueT(t, ok)
	assert.EqualT(t, "mypackage", rel)

	ok, _ = CheckPrefixAndFetchRelativePath("/other/path", "/home/user/go/src")
	assert.FalseT(t, ok)
}
