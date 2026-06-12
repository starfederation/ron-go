# ron-go

Go reference implementation for RON, Readable Object Notation.

RON is documented in the language-neutral reference repo:

- https://github.com/starfederation/ron

## Install

```sh
go get github.com/starfederation/ron-go
```

## API

```go
package main

import ron "github.com/starfederation/ron-go"

func main() {
    ronBody := []byte("find [?id ?name]")

    compactJSON, err := ron.ToJSON(ronBody)
    if err != nil {
        panic(err)
    }

    prettyJSON, err := ron.ToJSON(ronBody, ron.PrettyJSON("", "  "))
    if err != nil {
        panic(err)
    }

    prettyRON, err := ron.FromJSON([]byte(`{"find":["?id","?name"]}`), ron.Indent("  "))
    if err != nil {
        panic(err)
    }

    compactRON, err := ron.FromJSONCompact([]byte(`{"find":["?id","?name"]}`))
    if err != nil {
        panic(err)
    }

    fmt.Print(string(compactJSON))
    fmt.Print(string(prettyJSON))
    fmt.Print(string(prettyRON))
    fmt.Print(string(compactRON))
}
```

For repeated RON to JSON conversions, reuse `JSONBuilder`:

```go
var builder ron.JSONBuilder
jsonBody, err := ron.ToJSONInto(&builder, ronBody)
if err != nil {
    panic(err)
}
println(string(jsonBody))
builder.Reset()
```

## Conformance

Conformance tests use the reference corpus from `github.com/starfederation/ron` through `flake.nix`.

Run with Nix:

```sh
nix flake check
```

`nix develop` creates a local `testdata` symlink to the pinned flake input, so plain Go tests work inside the shell:

```sh
nix develop
go test ./...
```

Without Nix, set `RON_TESTDATA_DIR=/path/to/ron/testdata` or provide a local `testdata/conformance` directory. Otherwise conformance tests are skipped.

To update to the latest reference corpus:

```sh
nix flake lock --update-input ron
nix flake check
```

Commit `flake.lock` after the check passes.

## Why no git submodule?

Submodules make every clone responsible for `git submodule update --init`. The flake input pins the reference corpus in `flake.lock` and gives reproducible Nix checks without vendoring fixture files into this repo.
