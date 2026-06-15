# Color vocabulary

URI: `https://ron.dev/vocab/color/v1`

Color vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyColorV1)
```

Vocabulary-aware parsing validates `#clr` payloads and maps them to `ron.Color`. Rendering `ron.Color` or values implementing `github.com/SCKelemen/color.Color` emits canonical `#clr` values.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#clr` | OKLCH color `[space, lightness, chroma, hue]` | `ron.Color` | `github.com/SCKelemen/color` |

## Type notes

- `#clr` currently supports the canonical `oklch` color space.
- Payload shape is `["oklch", lightness, chroma, hueDegrees]`.
- Numeric channels must be finite JSON numbers.
- `ron.Color.Value` carries the corresponding `github.com/SCKelemen/color.Color` value for color conversions and manipulation.
- `ron.Color.Channels` preserves the canonical payload values for lossless RON rendering.
