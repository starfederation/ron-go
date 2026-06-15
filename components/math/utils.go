package math

import (
	stdmath "math"
	"math/rand"
)

const (
	DEG2RAD = stdmath.Pi / 180
	RAD2DEG = 180 / stdmath.Pi
	EPSILON = 7.0/3 - 4.0/3 - 1.0
)

type CoordinateSystem int

const (
	CoordinateSystemWebGL CoordinateSystem = iota
	CoordinateSystemWebGPU
)

var (
	coordinateSystem = CoordinateSystemWebGL
)

func Clamp[T Float](value, min, max T) T {
	return T(stdmath.Max(float64(min), stdmath.Min(float64(max), float64(value))))
}

// compute euclidean modulo of m % n
// https://en.wikipedia.org/wiki/Modulo_operation
func EuclideanModulo[T Float](n, m T) T {
	nf, mf := float64(n), float64(m)
	return T(stdmath.Mod(stdmath.Mod(nf, mf)+mf, mf))
}

// Linear mapping from range <a1, a2> to range <b1, b2>
func MapLinear[T Float](x, a1, a2, b1, b2 T) T {
	return b1 + (x-a1)*(b2-b1)/(a2-a1)
}

// https://www.gamedev.net/tutorials/programming/general-and-gameplay-programming/inverse-lerp-a-super-useful-yet-often-overlooked-function-r5230/
func InverseLerp[T Float](x, y, value T) T {
	if x != y {
		return (value - x) / (y - x)
	} else {
		return 0
	}
}

// https://en.wikipedia.org/wiki/Linear_interpolation
func Lerp[T Float](x, y, t T) T {
	return (1-t)*x + t*y
}

// http://www.rorydriscoll.com/2016/03/07/frame-rate-independent-damping-using-lerp/
func Damp[T Float](x, y, lambda, dt T) T {
	return Lerp(x, y, T(1-stdmath.Exp(float64(-lambda*dt))))
}

// https://www.desmos.com/calculator/vcsjnyz7x4
func PingPong[T Float](x, length T) T {
	return length - T(stdmath.Abs(float64(EuclideanModulo(x, length*2)-length)))
}

// http://en.wikipedia.org/wiki/Smoothstep
func Smoothstep[T Float](x, min, max T) T {
	if x <= min {
		return 0
	}
	if x >= max {
		return 1
	}
	x = (x - min) / (max - min)
	return x * x * (3 - 2*x)
}

func Smootherstep[T Float](x, min, max T) T {
	if x <= min {
		return 0
	}
	if x >= max {
		return 1
	}
	x = (x - min) / (max - min)
	return x * x * x * (x*(x*6-15) + 10)
}

// Random float from <-range/2, range/2> interval
func RandFloatSpread[T Float](rng T) T {
	return rng * (0.5 - T(rand.Float64()))
}

func DegToRad[T Float](degrees T) T {
	return degrees * DEG2RAD
}

func RadToDeg[T Float](radians T) T {
	return radians * RAD2DEG
}

func IsPowerOfTwo[T Integer](value T) bool {
	return value != 0 && (value&(value-1)) == 0
}

func CeilPowerOfTwo[T Integer](value T) T {
	return T(stdmath.Pow(2, stdmath.Ceil(stdmath.Log(float64(value))/stdmath.Ln2)))
}

func FloorPowerOfTwo[T Integer](value T) T {
	return T(stdmath.Pow(2, stdmath.Floor(stdmath.Log(float64(value))/stdmath.Ln2)))
}

func SetQuaternionFromProperEuler[T Float](q *Quaternion[T], a, b, c T, order string) {
	// Intrinsic Proper Euler Angles - see https://en.wikipedia.org/wiki/Euler_angles

	// rotations are applied to the axes in the order specified by 'order'
	// rotation by angle 'a' is applied first, then by angle 'b', then by angle 'c'
	// angles are in radians

	c2 := T(stdmath.Cos(float64(b / 2)))
	s2 := T(stdmath.Sin(float64(b / 2)))

	c13 := T(stdmath.Cos(float64((a + c) / 2)))
	s13 := T(stdmath.Sin(float64((a + c) / 2)))

	c1_3 := T(stdmath.Cos(float64((a - c) / 2)))
	s1_3 := T(stdmath.Sin(float64((a - c) / 2)))

	c3_1 := T(stdmath.Cos(float64((c - a) / 2)))
	s3_1 := T(stdmath.Sin(float64((c - a) / 2)))

	switch order {
	case "XYX":
		q.Set(c2*s13, s2*c1_3, s2*s1_3, c2*c13)
	case "YZY":
		q.Set(s2*s1_3, c2*s13, s2*c1_3, c2*c13)
	case "ZXZ":
		q.Set(s2*c1_3, s2*s1_3, c2*s13, c2*c13)
	case "XZX":
		q.Set(c2*s13, s2*s3_1, s2*c3_1, c2*c13)
	case "YXY":
		q.Set(s2*c3_1, c2*s13, s2*s3_1, c2*c13)
	case "ZYZ":
		q.Set(s2*s3_1, s2*c3_1, c2*s13, c2*c13)
	default:
		panic("Invalid order")
	}
}
