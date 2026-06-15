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

For repeated conversions, reuse a `bytes.Buffer`:

```go
var buf bytes.Buffer
jsonBody, err := ron.ToJSONInto(&buf, ronBody)
if err != nil {
    panic(err)
}
println(string(jsonBody))
buf.Reset()

ronBody, err = ron.FromJSONCompactInto(&buf, jsonBody)
if err != nil {
    panic(err)
}
println(string(ronBody))
buf.Reset()
```

Pretty JSON-to-RON renders root object members directly and can map JSON values to tagged RON values:

```go
ronBody, err := ron.FromJSON(
    []byte(`{"tx":"tx-48830","committed":"2026-06-13T00:00:00Z"}`),
    ron.IsCanonical(false),
    ron.MapJSONValues(func(path []ron.JSONPathSegment, value any) (any, bool) {
        if len(path) != 1 || path[0].IsIndex {
            return nil, false
        }
        switch path[0].Key {
        case "tx":
            return ron.Tagged("", value), true
        case "committed":
            return ron.Tagged("time", value), true
        default:
            return nil, false
        }
    }),
)
if err != nil {
    panic(err)
}
fmt.Print(string(ronBody))
```

Output:

```ron
tx {# tx-48830}
committed {#time 2026-06-13T00:00:00Z}
```

Supported typed vocabularies are enabled by default. Matching tagged values validate and map to native Go values, while unsupported tags remain ordinary RON objects:

```go
ronBody, err := ron.FromJSON(
    []byte(`{"id":{"#uid":"00112233-4455-6677-8899-aabbccddeeff"}}`),
)
```

See [typed vocabularies](./docs/vocabularies.md) for supported vocabulary pages, Go type mappings, and external libraries. [`VocabularyCoreV1`](./docs/vocabulary-core.md) supports `#uid`, `#url`, `#dec`, `#b64`, `#sha256`, `#`, and `#tag`.

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

Without Nix, set `RON_TESTDATA_DIR=/path/to/ron/testdata` or provide local `testdata/conformance`, `testdata/rfc8785`, and `testdata/vocabularies` directories. Otherwise testdata-backed tests are skipped.

To update to the latest reference corpus:

```sh
nix flake update ron
nix flake check
```

Commit `flake.lock` after the check passes.

## Why no git submodule?

Submodules make every clone responsible for `git submodule update --init`. The flake input pins the reference corpus in `flake.lock` and gives reproducible Nix checks without vendoring fixture files into this repo.
