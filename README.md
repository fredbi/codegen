# Codegen

<!-- Badges: status  -->
[![Tests][test-badge]][test-url] [![Coverage][cov-badge]][cov-url] [![CI vuln scan][vuln-scan-badge]][vuln-scan-url] [![CodeQL][codeql-badge]][codeql-url]
<!-- Badges: release & docker images  -->
<!-- Badges: code quality  -->
<!-- Badges: license & compliance -->
[![Release][release-badge]][release-url] [![Go Report Card][gocard-badge]][gocard-url] [![CodeFactor Grade][codefactor-badge]][codefactor-url] [![License][license-badge]][license-url]
<!-- Badges: documentation & support -->
<!-- Badges: others & stats -->
[![GoDoc][godoc-badge]][godoc-url] [![Discord Channel][discord-badge]][discord-url] [![go version][goversion-badge]][goversion-url] ![Top language][top-badge] ![Commits since latest release][commits-badge]

---

Tools to generate and test golang code.

* [Contents](#contents)
* [Dependencies](#dependencies)
* [Change log](#change-log)
* [Licensing](#licensing)
* [Note to contributors](#note-to-contributors)
* [Roadmap](#roadmap)

## Status

Work in progress.

## Import this library in your project

```cmd
go get github.com/go-openapi/Codegen/{module}
```

## Contents

`go-openapi/Codegen` exposes a collection of code generation tools and utilities.

---

## Dependencies

The root module `github.com/go-openapi/Codegen` at the repo level maintains a few
dependencies outside of the standard library.

---

## Note to contributors

All kinds of contributions are welcome.

This repo is a go mono-repo.

More general guidelines are available [here](.github/CONTRIBUTING.md).

## Roadmap

TODO

## Change log

See <https://github.com/go-openapi/Codegen/releases>

## Licensing

This library ships under the [SPDX-License-Identifier: Apache-2.0](./LICENSE).

## Cutting a new release

Maintainers can cut a new release by either:

* running [this workflow](https://github.com/go-openapi/Codegen/actions/workflows/bump-release.yml)
* or pushing a semver tag
  * signed tags are preferred
  * The tag message is prepended to release notes

<!-- Badges: status  -->
[test-badge]: https://github.com/go-openapi/Codegen/actions/workflows/go-test.yml/badge.svg
[test-url]: https://github.com/go-openapi/Codegen/actions/workflows/go-test.yml
[cov-badge]: https://codecov.io/gh/go-openapi/Codegen/branch/master/graph/badge.svg
[cov-url]: https://codecov.io/gh/go-openapi/Codegen
[vuln-scan-badge]: https://github.com/go-openapi/Codegen/actions/workflows/scanner.yml/badge.svg
[vuln-scan-url]: https://github.com/go-openapi/Codegen/actions/workflows/scanner.yml
[codeql-badge]: https://github.com/go-openapi/Codegen/actions/workflows/codeql.yml/badge.svg
[codeql-url]: https://github.com/go-openapi/Codegen/actions/workflows/codeql.yml
<!-- Badges: release & docker images  -->
[release-badge]: https://badge.fury.io/gh/go-openapi%2FCodegen.svg
[release-url]: https://badge.fury.io/gh/go-openapi%2FCodegen
[gomod-badge]: https://badge.fury.io/go/github.com%2Fgo-openapi%2FCodegen.svg
[gomod-url]: https://badge.fury.io/go/github.com%2Fgo-openapi%2FCodegen
<!-- Badges: code quality  -->
[gocard-badge]: https://goreportcard.com/badge/github.com/go-openapi/Codegen
[gocard-url]: https://goreportcard.com/report/github.com/go-openapi/Codegen
[codefactor-badge]: https://img.shields.io/codefactor/grade/github/go-openapi/Codegen
[codefactor-url]: https://www.codefactor.io/repository/github/go-openapi/Codegen
<!-- Badges: documentation & support -->
[doc-badge]: https://img.shields.io/badge/doc-site-blue?link=https%3A%2F%2Fgoswagger.io%2Fgo-openapi%2F
[doc-url]: https://goswagger.io/go-openapi
[godoc-badge]: https://pkg.go.dev/badge/github.com/go-openapi/Codegen
[godoc-url]: http://pkg.go.dev/github.com/go-openapi/Codegen
[discord-badge]: https://img.shields.io/discord/1446918742398341256?logo=discord&label=discord&color=blue
[discord-url]: https://discord.gg/FfnFYaC3k5

<!-- Badges: license & compliance -->
[license-badge]: http://img.shields.io/badge/license-Apache%20v2-orange.svg
[license-url]: https://github.com/go-openapi/Codegen/?tab=Apache-2.0-1-ov-file#readme
<!-- Badges: others & stats -->
[goversion-badge]: https://img.shields.io/github/go-mod/go-version/go-openapi/Codegen
[goversion-url]: https://github.com/go-openapi/Codegen/blob/master/go.mod
[top-badge]: https://img.shields.io/github/languages/top/go-openapi/Codegen
[commits-badge]: https://img.shields.io/github/commits-since/go-openapi/Codegen/latest
<!-- Organization docs -->
[contributing-doc-site]: https://go-openapi.github.io/doc-site/contributing/contributing/index.html
[maintainers-doc-site]: https://go-openapi.github.io/doc-site/maintainers/index.html
[style-doc-site]: https://go-openapi.github.io/doc-site/contributing/style/index.html
