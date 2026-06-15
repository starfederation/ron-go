# geo

Local GeoJSON and geographic helpers used by ron-go typed vocabularies.

The package provides:

- `Value`, the native value wrapper for the RON `#geo` vocabulary.
- GeoJSON shape validation for RFC 7946 geometry, feature, and feature collection payloads.
- Public point and bound helpers based on local `components/math` `Vector2` and `Box2` types.
- Local geometry collection types based on `components/math` `Vector2` points.
- Local geodesic helpers for distance, bounds, length, and area functions.
- Local planar helpers for Euclidean distance, length, area, and point-in-polygon checks.

It intentionally avoids external GeoJSON and geometry packages so ron-go does not pull in unrelated transitive dependencies.
