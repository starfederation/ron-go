# Math vocabulary

URI: `https://ron.dev/vocab/math/v1`

Math vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyMathV1)
```

Math values use `github.com/starfederation/ron-go/components/math`, a local copy of `github.com/delaneyj/geck/components/mathx`, so ron-go does not add an external runtime dependency for math values while preserving the math method set.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#i64` | Int64 | `ron.Int64` | None |
| `#u64` | Uint64 | `ron.Uint64` | None |
| `#f64` | Float64 | `ron.Float64` | None |
| `#ivN` | integer vector | `ron.IntVectorN` | None |
| `#vN` | float vector | `ron.VectorN` | None |
| `#iv2`, `#iv3`, `#iv4` | fixed integer vectors | `ron.IntVector2`, `ron.IntVector3`, `ron.IntVector4` | None |
| `#f2v`, `#f3v`, `#f4v` | fixed float vectors | `ron.Vector2`, `ron.Vector3`, `ron.Vector4` | Local `components/math` for 2D/3D |
| `#qat` | quaternion | `ron.Quaternion` | Local `components/math` |
| `#eul` | Euler rotation | `ron.Euler` | Local `components/math` |
| `#m2x`, `#m3x`, `#m4x` | matrices | `ron.Matrix2`, `ron.Matrix3`, `ron.Matrix4` | Local `components/math` for 3x3/4x4 |

## Type notes

- Integer scalar payloads are canonical base-10 strings.
- Integer vector payloads are JSON integer numbers.
- Float payloads must be finite.
- Matrix payloads are column-major.
- Euler order supports `XYZ`, `YXZ`, `ZXY`, `ZYX`, `YZX`, and `XZY`.
