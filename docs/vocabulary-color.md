# Color vocabulary

URI: `https://ron.dev/vocab/color/v1`

Color vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyColorV1)
```

Vocabulary-aware parsing validates `#clr` payloads and maps them to `ron.Color`. Rendering `ron.Color` preserves its stored color space and channels. Rendering values implementing `github.com/SCKelemen/color.Color` emits `oklch` or `oklcha` payloads.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#clr` | Color `[space, channels...]` | `ron.Color` | `github.com/SCKelemen/color` |

## Type notes

- Supported spaces: `rgb`, `rgba`, `hsl`, `hsla`, `hsv`, `hsva`, `hwb`, `hwba`, `lab`, `laba`, `lch`, `lcha`, `oklab`, `oklaba`, `oklch`, `oklcha`, `xyz`, and `xyza`.
- Non-alpha spaces use three numeric channels.
- Alpha spaces use four numeric channels with alpha in `[0, 1]`.
- Numeric channels must be finite JSON numbers.
- `ron.Color.Value` carries the corresponding `github.com/SCKelemen/color.Color` value for color conversions and manipulation.
- `ron.Color.Channels` preserves payload channel values for lossless RON rendering.
