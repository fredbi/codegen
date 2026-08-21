# reports

What a [templates repository](../README.md) says about itself: the documentation of its templates, and the
audit of what it holds.

```go
import "github.com/go-openapi/codegen/templates-repo/reports"
```

A repository executes templates. Describing them is a separate job, and the types that describe
them outnumber the ones that run them. They live here so that a program which only renders
templates imports none of it.

## Documentation

`Repository.Documentation` returns a `Documentation`, grouped by asset so it follows the tree an
author edits, and ordered throughout so the same templates produce the same document every time.

Each `Template` in it carries the comments documenting it, the data paths it reads, the functions
it calls, and the templates it calls with the data passed to each. `Transitive` holds the same
once the templates it calls are folded in, with their paths rebased onto the data handed to them.

`Dump` renders a documentation as markdown:

```go
documentation, err := repository.Documentation()
if err != nil {
    return err
}

err = reports.Dump(w, documentation)
```

`WithTemplate` lays it out otherwise, and `WithFuncMap` adds functions such a layout may call.
Walk the `Documentation` directly for anything a text template cannot produce.

```go
err = reports.Dump(w, documentation, reports.WithTemplate(myLayout))
```

`Repository.Dump` is the same thing in one call, for the common case of rendering markdown once.

## Audit

`Repository.Audit` returns an `Audit`, listing what compiles and runs but still deserves a look:

| | |
|---|---|
| `Overridden` | templates more than one asset declared, and which definition stands |
| `Unused` | templates no other template calls |
| `Empty` | templates that render nothing |
| `Dynamic` | templates calling a function carried by their data |
| `UnusedFuncs` | funcmap entries no template calls |

None of it is an error. The repository rejects what it cannot resolve when it is built, so
everything reported here already works. A function no funcmap provides never reaches the audit
either: templates are parsed against the funcmap, so that fails the build instead.

`Unused` and `UnusedFuncs` are observations rather than verdicts. Nothing calls a generator's
entry points either, and a general-purpose funcmap is mostly unused by design. Scope the
repository to its roots and `Unused` answers for itself.

## Errors

Everything this package reports matches `ErrReport` and wraps its cause, so a caller may match the
underlying template error with `errors.Is` and `errors.As` all the same.

## Tests

```sh
go test ./...
golangci-lint run
```
