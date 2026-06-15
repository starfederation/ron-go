# GeoJSON vocabulary

URI: `https://ron.dev/vocab/geo/v1`

GeoJSON vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyGeoV1)
```

GeoJSON values use local `components/geo` validation for RFC 7946 geometry, feature, and feature collection shapes. `components/geo` exposes point, bound, geometry collection, geodesic, and planar helpers through local types backed by `components/math`. No external GeoJSON or geometry dependency is required.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#geo` | GeoJSON geometry, feature, or feature collection | `ron.GeoJSON` alias of `geo.Value` | Local `components/geo` |

## Type notes

- `#geo` accepts RFC 7946 Geometry, Feature, and FeatureCollection payloads.
- `Point`, `MultiPoint`, `LineString`, `MultiLineString`, `Polygon`, `MultiPolygon`, `GeometryCollection`, `Feature`, and `FeatureCollection` are supported.
- `Feature.geometry` may be `null`.
- `bbox` and foreign members are preserved in the RON value model.
- Typed values nested inside GeoJSON properties are parsed when their vocabularies are enabled.
