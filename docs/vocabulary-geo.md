# GeoJSON vocabulary

URI: `https://ron.dev/vocab/geo/v1`

GeoJSON vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyGeoV1)
```

GeoJSON values use `github.com/paulmach/orb/geojson` for validation.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#geo` | GeoJSON geometry, feature, or feature collection | `ron.GeoJSON` | `github.com/paulmach/orb/geojson` |

## Type notes

- `#geo` accepts RFC 7946 Geometry, Feature, and FeatureCollection payloads.
- Geometry payloads are validated with `geojson.UnmarshalGeometry`.
- Feature payloads are validated with `geojson.UnmarshalFeature`.
- FeatureCollection payloads are validated with `geojson.UnmarshalFeatureCollection`.
- `bbox` and foreign members are preserved in the RON value model.
- Typed values nested inside GeoJSON properties are parsed when their vocabularies are enabled.
