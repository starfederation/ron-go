# Spatial vocabulary

URI: `https://ron.dev/vocab/spatial/v1`

Spatial vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularySpatialV1)
```

Spatial values use `github.com/paulmach/orb` where it fits 2D geospatial shapes, and local `components/math` types for 3D geometry.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#lla` | longitude, latitude, altitude | `ron.LngLatAlt` with `orb.Point` | `github.com/paulmach/orb` |
| `#sph` | spherical coordinates | `ron.Spherical` | Local `components/math` |
| `#cyl` | cylindrical coordinates | `ron.Cylindrical` | Local `components/math` |
| `#bx2` | 2D box | `ron.Box2` alias of `orb.Bound` | `github.com/paulmach/orb` |
| `#bx3` | 3D box | `ron.Box3` | Local `components/math` |
| `#spr` | sphere | `ron.Sphere` | Local `components/math` |
| `#pln` | plane | `ron.Plane` | Local `components/math` |
| `#ray` | ray | `ron.Ray` | Local `components/math` |
| `#ln2` | 2D line segment | `ron.Line2` with `orb.LineString` | `github.com/paulmach/orb` |
| `#ln3` | 3D line segment | `ron.Line3` | Local `components/math` |
| `#tri` | triangle | `ron.Triangle` | Local `components/math` |
| `#fru` | frustum planes | `ron.Frustum` | Local `components/math` |
| `#sh3` | spherical harmonics | `ron.SphericalHarmonics3` | Local `components/math` |
| `#vox` | sparse voxel set | `ron.VoxelSet` | None |

## Type notes

- Spatial distances use meters unless a tag states otherwise.
- `#lla` stores longitude/latitude in an `orb.Point` and altitude separately.
- `#vox` stores sparse cells as coordinate/value pairs; cell values may contain typed values from enabled vocabularies.
