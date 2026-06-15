# Core vocabulary

URI: `https://ron.dev/vocab/core/v1`

Core vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyCoreV1)
```

Vocabulary-aware parsing validates matching core tags and maps them to native Go values. Rendering native core values emits the matching tagged RON form.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#uid` | UUID | `ron.UUID` alias of `uuid.UUID` | `github.com/google/uuid` |
| `#url` | Absolute URL | `*url.URL` | Go stdlib `net/url` |
| `#dec` | Decimal | `*ron.Decimal` alias of `apd.Decimal` | `github.com/cockroachdb/apd/v3` |
| `#b64` | Bytes | `ron.Bytes` (`[]byte`) | Go stdlib `encoding/base64` |
| `#sha256` | SHA-256 hash | `ron.SHA256` (`[32]byte`) | Go stdlib `encoding/hex` |
| `#` | Entity/database reference | `ron.EntityRef` | None |
| `#tag` | Opaque implementation-defined tag | `ron.OpaqueTag` | None |

## Type notes

### `ron.UUID`

`ron.UUID` is an alias of `github.com/google/uuid.UUID`. RON accepts only lowercase RFC 4122 text and renders lowercase text.

### `*url.URL`

`#url` uses Go stdlib `net/url`. RON validates that the URL is absolute and preserves the URL string supplied by the renderer.

### `*ron.Decimal`

`ron.Decimal` is an alias of `github.com/cockroachdb/apd/v3.Decimal`. Parsed `#dec` values are returned as `*ron.Decimal` so callers can use APD arithmetic directly. Rendering reduces the decimal and emits non-exponent canonical text with `Text('f')`.

### `ron.Bytes`

`ron.Bytes` is decoded from RFC 4648 base64url without padding. Rendering emits base64url without padding.

### `ron.SHA256`

`ron.SHA256` stores the decoded 32-byte hash. Rendering emits 64 lowercase hex characters.

### `ron.EntityRef`

`ron.EntityRef.Value` is the validated integer or string payload. Integer payloads preserve the parsed JSON/RON numeric representation.

### `ron.OpaqueTag`

`ron.OpaqueTag` stores the first `#tag` array element in `Tag` and the second element in `Payload`. Payload values are recursively vocabulary-aware when supported vocabularies are enabled.
