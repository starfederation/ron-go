# Custom vocabularies

Custom typed vocabularies are option-scoped. ron-go does not use a global registry.

Use `UseCustomVocabulary` to enable a vocabulary URI, declare the tags it owns, and optionally provide parse and render callbacks:

```go
ron.UseCustomVocabulary(ron.CustomVocabulary{
    URI:  "https://example.com/vocab/invoice/v1",
    Tags: []string{"#com.example/money"},
    Parse: func(tag string, payload any) (any, error) {
        return ron.Custom(tag, payload), nil
    },
    Render: func(value any) (string, any, bool) {
        money, ok := value.(Money)
        if !ok {
            return "", nil, false
        }
        return "#com.example/money", []any{money.Currency, money.Amount}, true
    },
})
```

## Type notes

- `UseCustomVocabulary` also enables the custom URI for the current conversion call.
- `Tags` decide which single-key tagged objects are handled by the custom vocabulary.
- `Parse` validates and maps tagged payloads while converting JSON or validating RON.
- If `Parse` is nil, matching custom tagged values map to `ron.CustomValue`.
- `Render` maps native values back to tagged payloads for the current conversion call.
- `ron.Custom(tag, payload)` creates a generic `ron.CustomValue` that renders without a render callback.
- Unknown custom vocabulary URIs passed only through `EnableVocabularies` are rejected.
