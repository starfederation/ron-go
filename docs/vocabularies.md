# Typed vocabularies

RON typed vocabularies are optional semantic layers over JSON-compatible single-key objects.

Language-neutral vocabulary definitions live in the reference repo:

- https://github.com/starfederation/ron/blob/main/docs/vocabularies.md

## Supported in ron-go

Supported vocabularies are enabled by default.

Use `ValidateVocabularyProfile` to reject profiles that require unknown or unsupported vocabularies. Optional unknown vocabularies are allowed and remain ordinary JSON/RON unless an option-scoped custom vocabulary handles them.

| Vocabulary | URI | Go types | External libraries |
| --- | --- | --- | --- |
| [Core](./vocabulary-core.md) | `https://ron.dev/vocab/core/v1` | `ron.UUID`, `*url.URL`, `*ron.Decimal`, `ron.Bytes`, `ron.SHA256`, `ron.EntityRef`, `ron.OpaqueTag` | `github.com/google/uuid`, `github.com/cockroachdb/apd/v3` |
| [Time](./vocabulary-time.md) | `https://ron.dev/vocab/time/v1` | `ron.Instant`, `ron.Duration` | Go stdlib `time` |
| [Network](./vocabulary-network.md) | `https://ron.dev/vocab/network/v1` | `ron.IPv4`, `ron.IPv6`, `ron.CIDR` | Go stdlib `net/netip` |
| [Math](./vocabulary-math.md) | `https://ron.dev/vocab/math/v1` | `ron.Int64`, `ron.Uint64`, `ron.Float64`, vectors, matrices, `ron.Quaternion`, `ron.Euler` | Local `components/math` |
| [Spatial](./vocabulary-spatial.md) | `https://ron.dev/vocab/spatial/v1` | `ron.LngLatAlt`, `ron.Box2`, `ron.Box3`, `ron.Sphere`, `ron.Plane`, `ron.Ray`, `ron.Line2`, `ron.Line3`, `ron.Triangle`, `ron.Frustum`, `ron.SphericalHarmonics3`, `ron.VoxelSet` | Local `components/math` |
| [GeoJSON](./vocabulary-geo.md) | `https://ron.dev/vocab/geo/v1` | `ron.GeoJSON` | Local `components/geo` |
| [Color](./vocabulary-color.md) | `https://ron.dev/vocab/color/v1` | `ron.Color` | `github.com/SCKelemen/color` |
| [Custom](./vocabulary-custom.md) | Option-scoped | `ron.CustomValue`, application types | None |
