# Set vocabulary

URI: `https://ron.dev/vocab/set/v1`

Set vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularySetV1)
```

Vocabulary-aware parsing validates matching set tags and maps them to native Go values. Rendering native set values emits the matching tagged RON form.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#set` | Generic finite set of JSON/RON values | `ron.Set` | None |
| `#bits` | Uint32 bitset | `ron.Uint32BitSet` | None |

## Type notes

### `ron.Set`

`ron.Set` is a duplicate-free finite set of JSON-compatible values. Values canonicalize by RFC 8785 canonical JSON bytes, so object member order does not affect equality.

Rendering sorts values by canonical JSON byte order and removes duplicates:

```ron
{#set [admin reader writer]}
```

### `ron.Uint32BitSet`

`ron.Uint32BitSet` stores inclusive `ron.Uint32Range` values. Parsing accepts singleton uint32 numbers and inclusive two-element ranges, then canonicalizes by sorting and merging duplicates, overlaps, and adjacent runs.

Rendering emits logical ranges, not roaring bitmap bytes:

```ron
{#bits [0 3 [10 14] 1538289]}
```
