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

    fmt.Println(string(compactJSON))
    fmt.Println(string(prettyJSON))
    fmt.Println(string(prettyRON))
    fmt.Println(string(compactRON))
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
enc := ron.NewEncoder(&out, ron.IsPretty(false))
if err := enc.Encode(map[string]any{"person": Person{ID: 1538289, Name: "Ada"}}); err != nil {
    panic(err)
}
```

Byte-returning conversion and marshal APIs return exactly one encoded value without a trailing newline. `Marshal` emits pretty RON, `MarshalCompact` emits compact RON, and `NewEncoder` writes one RON value plus a trailing newline per `Encode` call. Reflection supports common JSON-shaped Go values and `json` struct tags including `omitempty`.

### Stream profiles

`NewNdronEncoder` and `NewNdronDecoder` implement `application/x-ndron` (`.ndron`) with one compact RON value per LF-terminated record. Decoders accept LF and CRLF. Empty lines are errors by default; use `IgnoreEmptyNdronLines(true)` to skip them.

`NewRonSequenceEncoder` and `NewRonSequenceDecoder` implement `application/ron-seq` with RS-prefixed, LF-terminated records. Sequence decoders consume malformed elements and recover at the next RS, so callers may continue after non-EOF errors. `Decode` uses `encoding/json` semantics for its destination.

Both profiles default to a 1 MiB record limit and 100 nesting levels. Override these bounded defaults with `MaxRecordSize` and `MaxNestingDepth`.

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

See [typed vocabularies](./docs/vocabularies.md) for supported vocabulary pages, Go type mappings, and external libraries. [`VocabularyCoreV1`](./docs/vocabulary-core.md) supports `#uid`, `#url`, `#rx`, `#dec`, `#b64`, `#sha256`, `#`, and `#tag`.

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

Without Nix, set `RON_TESTDATA_DIR=/path/to/ron/testdata` or provide local `testdata/conformance`, `testdata/rfc8785`, `testdata/sequences`, and `testdata/vocabularies` directories. Otherwise testdata-backed tests are skipped.

To update to the latest reference corpus:

```sh
nix flake update ron
nix flake check
```

Commit `flake.lock` after the check passes.

## Why no git submodule?

Submodules make every clone responsible for `git submodule update --init`. The flake input pins the reference corpus in `flake.lock` and gives reproducible Nix checks without vendoring fixture files into this repo.
