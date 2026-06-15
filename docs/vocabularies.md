# Typed vocabularies

RON typed vocabularies are optional semantic layers over JSON-compatible single-key objects.

Language-neutral vocabulary definitions live in the reference repo:

- https://github.com/starfederation/ron/blob/main/docs/vocabularies.md

## Supported in ron-go

| Vocabulary | URI | Go types | External libraries |
| --- | --- | --- | --- |
| [Core](./vocabulary-core.md) | `https://ron.dev/vocab/core/v1` | `ron.UUID`, `*url.URL`, `*ron.Decimal`, `ron.Bytes`, `ron.SHA256`, `ron.EntityRef`, `ron.OpaqueTag` | `github.com/google/uuid`, `github.com/cockroachdb/apd/v3` |

## Planned vocabularies

These vocabularies are defined by the reference repo but are not implemented in ron-go yet:

| Vocabulary | URI | Status |
| --- | --- | --- |
| Time | `https://ron.dev/vocab/time/v1` | Planned |
| Network | `https://ron.dev/vocab/network/v1` | Planned |
| Math | `https://ron.dev/vocab/math/v1` | Planned |
| Spatial | `https://ron.dev/vocab/spatial/v1` | Planned |
| Geo | `https://ron.dev/vocab/geo/v1` | Planned |
| Color | `https://ron.dev/vocab/color/v1` | Planned |

Add one page per implemented vocabulary so users can see the wire tags, Go types, and any external packages used.
