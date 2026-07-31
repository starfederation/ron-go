# ron-go

Go reference implementation for RON v1, Readable Object Notation.

RON is documented in the language-neutral reference repo:

- https://github.com/starfederation/ron

RON v1 strings use JSON escapes in every bare, quoted, comma-prefixed, key, and value form. Literal backslashes require `\\`; malformed escapes, unpaired surrogates, and raw C0 string content are rejected.

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

    compactJSON, err := ron.ToJSON(ronBody, ron.Mode(ron.Compact))
    if err != nil {
        panic(err)
    }

    prettyJSON, err := ron.ToJSON(ronBody, ron.Mode(ron.Pretty))
    if err != nil {
        panic(err)
    }

    prettyRON, err := ron.FromJSON([]byte(`{"find":["?id","?name"]}`), ron.Mode(ron.Pretty), ron.Indent("  "))
    if err != nil {
        panic(err)
    }

    compactRON, err := ron.FromJSON([]byte(`{"find":["?id","?name"]}`), ron.Mode(ron.Compact))
    if err != nil {
        panic(err)
    }

    fmt.Println(string(compactJSON))
    fmt.Println(string(prettyJSON))
    fmt.Println(string(prettyRON))
    fmt.Println(string(compactRON))
}
```

For repeated conversions, reuse a `bytes.Buffer`:

```go
var buf bytes.Buffer
jsonBody, err := ron.ToJSONInto(&buf, ronBody, ron.Mode(ron.Compact))
if err != nil {
    panic(err)
}
println(string(jsonBody))
buf.Reset()

ronBody, err = ron.FromJSONInto(&buf, jsonBody, ron.Mode(ron.Compact))
if err != nil {
    panic(err)
}
println(string(ronBody))
buf.Reset()
```

Go values can be encoded directly to RON without a JSON byte round trip:

```go
type Person struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

ronBody, err := ron.Marshal(Person{ID: 1538289, Name: "Ada"})
if err != nil {
    panic(err)
}

var out bytes.Buffer
enc := ron.NewEncoder(&out, ron.Mode(ron.Compact))
if err := enc.Encode(map[string]any{"person": Person{ID: 1538289, Name: "Ada"}}); err != nil {
    panic(err)
}
```

Use `Mode(Pretty)`, `Mode(Compact)`, or `Mode(Canonical)` with conversion and encoder APIs. Pretty RON has a trailing newline. Compact RON preserves member order. Canonical RON and JSON validate I-JSON values, canonicalize numbers, and sort keys by RFC 8785 UTF-16 order. `Marshal` emits compact RON. `MarshalPretty` and `MarshalCanonical` emit the other modes. `NewEncoder` writes one RON value per `Encode` call. Reflection supports common JSON-shaped Go values and `json` struct tags including `omitempty`.

Pretty JSON-to-RON renders root object members directly and can map JSON values to tagged RON values:

```go
ronBody, err := ron.FromJSON(
    []byte(`{"tx":"tx-48830","committed":"2026-06-13T00:00:00Z"}`),
    ron.Mode(ron.Pretty),
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

See [typed vocabularies](./docs/vocabularies.md) for supported vocabulary pages, Go type mappings, and external libraries. [`VocabularyCoreV1`](./docs/vocabulary-core.md) supports `#uid`, `#url`, `#rx`, `#dec`, `#b64`, `#sha256`, `#`, and `#tag`.

## Conformance

Conformance tests use the reference corpus from `github.com/starfederation/ron` through `flake.nix`.

Run with Nix:

```sh
nix flake check
```

Plain Go tests use the sibling reference repository at `../ron/testdata`:

```sh
nix develop
go test ./...
```

Set `RON_TESTDATA_DIR` only when the reference repository is at a different path. Missing or stale fixtures fail the tests.

To update to the latest reference corpus:

```sh
nix flake update ron
nix flake check
```

Commit `flake.lock` after the check passes.

## Why no git submodule?

Submodules make every clone responsible for `git submodule update --init`. The flake input pins the reference corpus in `flake.lock` and gives reproducible Nix checks without vendoring fixture files into this repo.
