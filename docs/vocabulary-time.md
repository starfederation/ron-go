# Time vocabulary

URI: `https://ron.dev/vocab/time/v1`

Time vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyTimeV1)
```

Vocabulary-aware parsing validates matching time tags and maps them to Go stdlib values. Rendering native time values emits the matching tagged RON form.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#utc` | UTC instant | `ron.Instant` alias of `time.Time` | Go stdlib `time` |
| `#dur` | Day-time duration | `ron.Duration` alias of `time.Duration` | Go stdlib `time` |

## Type notes

### `ron.Instant`

`ron.Instant` is an alias of `time.Time`. RON accepts RFC 3339 UTC instants with uppercase `T` and `Z`. Fractional seconds may have 1 to 9 digits and are rendered with trailing fractional zeroes trimmed.

### `ron.Duration`

`ron.Duration` is an alias of `time.Duration`. RON accepts restricted ISO 8601 day-time durations only:

```text
P[nD][T[nH][nM][n[.fraction]S]]
```

Rules:

- Optional leading `-` for negative durations.
- Years, months, and weeks are rejected.
- At least one component is required.
- Fractional units are only supported on seconds.
- Fractional seconds may have up to 9 digits.
- Zero renders as `PT0S`.
