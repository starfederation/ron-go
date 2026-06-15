package ron

import (
	"encoding/json"
	"math"
	"strconv"

	ronmath "github.com/starfederation/ron-go/components/math"
)

const (
	// VocabularyMathV1 is the RON math typed vocabulary URI.
	VocabularyMathV1 = "https://ron.dev/vocab/math/v1"
)

// Int64 is a math vocabulary #i64 value.
type Int64 int64

// Uint64 is a math vocabulary #u64 value.
type Uint64 uint64

// Float64 is a math vocabulary #f64 value.
type Float64 float64

// IntVectorN is a math vocabulary #ivN value.
type IntVectorN []int64

// VectorN is a math vocabulary #vN value.
type VectorN []float64

// IntVector2 is a math vocabulary #iv2 value.
type IntVector2 [2]int64

// IntVector3 is a math vocabulary #iv3 value.
type IntVector3 [3]int64

// IntVector4 is a math vocabulary #iv4 value.
type IntVector4 [4]int64

// Vector2 is a math vocabulary #f2v value.
type Vector2 = ronmath.Vector2[float64]

// Vector3 is a math vocabulary #f3v value.
type Vector3 = ronmath.Vector3[float64]

// Vector4 is a math vocabulary #f4v value.
type Vector4 [4]float64

// Quaternion is a math vocabulary #qat value.
type Quaternion = ronmath.Quaternion[float64]

// EulerOrder is the rotation order for a math vocabulary #eul value.
type EulerOrder = ronmath.EulerOrder

const (
	EulerOrderXYZ = ronmath.EULER_ORDER_XYZ
	EulerOrderYXZ = ronmath.EULER_ORDER_YXZ
	EulerOrderZXY = ronmath.EULER_ORDER_ZXY
	EulerOrderZYX = ronmath.EULER_ORDER_ZYX
	EulerOrderYZX = ronmath.EULER_ORDER_YZX
	EulerOrderXZY = ronmath.EULER_ORDER_XZY
)

// Euler is a math vocabulary #eul value.
type Euler = ronmath.Euler[float64]

// Matrix2 is a math vocabulary #m2x value.
type Matrix2 [4]float64

// Matrix3 is a math vocabulary #m3x value.
type Matrix3 = ronmath.Matrix3[float64]

// Matrix4 is a math vocabulary #m4x value.
type Matrix4 = ronmath.Matrix4[float64]

func (opts optionState) isMathTag(tag string) bool {
	if _, ok := opts.vocabularies[VocabularyMathV1]; !ok {
		return false
	}
	switch tag {
	case "#i64", "#u64", "#f64", "#ivN", "#vN", "#iv2", "#iv3", "#iv4", "#f2v", "#f3v", "#f4v", "#qat", "#eul", "#m2x", "#m3x", "#m4x":
		return true
	default:
		return false
	}
}

func (opts optionState) parseMathPayload(tag string, payload any) (any, error) {
	switch tag {
	case "#i64":
		value, ok := payload.(string)
		if !ok {
			return nil, newError("invalid #i64 payload")
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || strconv.FormatInt(parsed, 10) != value {
			return nil, newError("invalid #i64 payload")
		}
		return Int64(parsed), nil
	case "#u64":
		value, ok := payload.(string)
		if !ok {
			return nil, newError("invalid #u64 payload")
		}
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil || strconv.FormatUint(parsed, 10) != value {
			return nil, newError("invalid #u64 payload")
		}
		return Uint64(parsed), nil
	case "#f64":
		value, ok := numberAsFloat64(payload)
		if !ok {
			return nil, newError("invalid #f64 payload")
		}
		return Float64(value), nil
	case "#ivN":
		values, ok := payload.([]any)
		if !ok {
			return nil, newError("invalid #ivN payload")
		}
		parsed, err := parseIntVector(values)
		if err != nil {
			return nil, err
		}
		return IntVectorN(parsed), nil
	case "#vN":
		values, ok := payload.([]any)
		if !ok {
			return nil, newError("invalid #vN payload")
		}
		parsed, err := parseFloatVector(values)
		if err != nil {
			return nil, err
		}
		return VectorN(parsed), nil
	case "#iv2", "#iv3", "#iv4":
		values, ok := payload.([]any)
		if !ok {
			return nil, newError("invalid integer vector payload")
		}
		parsed, err := parseIntVector(values)
		if err != nil {
			return nil, err
		}
		switch tag {
		case "#iv2":
			if len(parsed) != 2 {
				return nil, newError("invalid #iv2 payload")
			}
			return IntVector2{parsed[0], parsed[1]}, nil
		case "#iv3":
			if len(parsed) != 3 {
				return nil, newError("invalid #iv3 payload")
			}
			return IntVector3{parsed[0], parsed[1], parsed[2]}, nil
		default:
			if len(parsed) != 4 {
				return nil, newError("invalid #iv4 payload")
			}
			return IntVector4{parsed[0], parsed[1], parsed[2], parsed[3]}, nil
		}
	case "#f2v", "#f3v", "#f4v", "#qat", "#m2x", "#m3x", "#m4x":
		values, ok := payload.([]any)
		if !ok {
			return nil, newError("invalid float vector payload")
		}
		parsed, err := parseFloatVector(values)
		if err != nil {
			return nil, err
		}
		switch tag {
		case "#f2v":
			if len(parsed) != 2 {
				return nil, newError("invalid #f2v payload")
			}
			return Vector2{X: parsed[0], Y: parsed[1]}, nil
		case "#f3v":
			if len(parsed) != 3 {
				return nil, newError("invalid #f3v payload")
			}
			return Vector3{X: parsed[0], Y: parsed[1], Z: parsed[2]}, nil
		case "#f4v":
			if len(parsed) != 4 {
				return nil, newError("invalid #f4v payload")
			}
			return Vector4{parsed[0], parsed[1], parsed[2], parsed[3]}, nil
		case "#qat":
			if len(parsed) != 4 {
				return nil, newError("invalid #qat payload")
			}
			return Quaternion{X: parsed[0], Y: parsed[1], Z: parsed[2], W: parsed[3]}, nil
		case "#m2x":
			if len(parsed) != 4 {
				return nil, newError("invalid #m2x payload")
			}
			return Matrix2{parsed[0], parsed[1], parsed[2], parsed[3]}, nil
		case "#m3x":
			if len(parsed) != 9 {
				return nil, newError("invalid #m3x payload")
			}
			return Matrix3(parsed), nil
		default:
			if len(parsed) != 16 {
				return nil, newError("invalid #m4x payload")
			}
			return Matrix4(parsed), nil
		}
	case "#eul":
		values, ok := payload.([]any)
		if !ok || len(values) != 4 {
			return nil, newError("invalid #eul payload")
		}
		var xyz [3]float64
		for i := range xyz {
			value, ok := numberAsFloat64(values[i])
			if !ok {
				return nil, newError("invalid #eul payload")
			}
			xyz[i] = value
		}
		orderName, ok := values[3].(string)
		if !ok {
			return nil, newError("invalid #eul payload")
		}
		order, ok := parseEulerOrder(orderName)
		if !ok {
			return nil, newError("invalid #eul payload")
		}
		return Euler{X: xyz[0], Y: xyz[1], Z: xyz[2], Order: order}, nil
	default:
		return nil, newError("unsupported math tag")
	}
}

func mathTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case Int64:
		return objectMember{Key: "#i64", Value: strconv.FormatInt(int64(value), 10)}, true
	case Uint64:
		return objectMember{Key: "#u64", Value: strconv.FormatUint(uint64(value), 10)}, true
	case Float64:
		return objectMember{Key: "#f64", Value: float64(value)}, true
	case IntVectorN:
		return objectMember{Key: "#ivN", Value: intSliceToAny(value)}, true
	case VectorN:
		return objectMember{Key: "#vN", Value: floatSliceToAny(value)}, true
	case IntVector2:
		return objectMember{Key: "#iv2", Value: []any{value[0], value[1]}}, true
	case IntVector3:
		return objectMember{Key: "#iv3", Value: []any{value[0], value[1], value[2]}}, true
	case IntVector4:
		return objectMember{Key: "#iv4", Value: []any{value[0], value[1], value[2], value[3]}}, true
	case Vector2:
		return objectMember{Key: "#f2v", Value: []any{value.X, value.Y}}, true
	case Vector3:
		return objectMember{Key: "#f3v", Value: []any{value.X, value.Y, value.Z}}, true
	case Vector4:
		return objectMember{Key: "#f4v", Value: []any{value[0], value[1], value[2], value[3]}}, true
	case Quaternion:
		return objectMember{Key: "#qat", Value: []any{value.X, value.Y, value.Z, value.W}}, true
	case Euler:
		order, ok := formatEulerOrder(value.Order)
		if !ok {
			return objectMember{}, false
		}
		return objectMember{Key: "#eul", Value: []any{value.X, value.Y, value.Z, order}}, true
	case Matrix2:
		return objectMember{Key: "#m2x", Value: floatSliceToAny(value[:])}, true
	case Matrix3:
		return objectMember{Key: "#m3x", Value: floatSliceToAny(value[:])}, true
	case Matrix4:
		return objectMember{Key: "#m4x", Value: floatSliceToAny(value[:])}, true
	default:
		return objectMember{}, false
	}
}

func parseIntVector(values []any) ([]int64, error) {
	parsed := make([]int64, len(values))
	for i, value := range values {
		integer, ok := numberAsInt64(value)
		if !ok {
			return nil, newError("invalid integer vector payload")
		}
		parsed[i] = integer
	}
	return parsed, nil
}

func parseFloatVector(values []any) ([]float64, error) {
	parsed := make([]float64, len(values))
	for i, value := range values {
		f, ok := numberAsFloat64(value)
		if !ok {
			return nil, newError("invalid float vector payload")
		}
		parsed[i] = f
	}
	return parsed, nil
}

func numberAsInt64(value any) (int64, bool) {
	switch value := value.(type) {
	case json.Number:
		integer, err := strconv.ParseInt(value.String(), 10, 64)
		return integer, err == nil && strconv.FormatInt(integer, 10) == value.String()
	case ronNumber:
		integer, err := strconv.ParseInt(string(value), 10, 64)
		return integer, err == nil && strconv.FormatInt(integer, 10) == string(value)
	case int64:
		return value, true
	default:
		return 0, false
	}
}

func numberAsFloat64(value any) (float64, bool) {
	switch value := value.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(value.String(), 64)
		return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
	case ronNumber:
		f, err := strconv.ParseFloat(string(value), 64)
		return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
	case float64:
		return value, !math.IsNaN(value) && !math.IsInf(value, 0)
	case int64:
		return float64(value), true
	case uint64:
		return float64(value), true
	default:
		return 0, false
	}
}

func intSliceToAny[S ~[]int64](values S) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func floatSliceToAny[S ~[]float64](values S) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func parseEulerOrder(value string) (EulerOrder, bool) {
	switch value {
	case "XYZ":
		return EulerOrderXYZ, true
	case "YXZ":
		return EulerOrderYXZ, true
	case "ZXY":
		return EulerOrderZXY, true
	case "ZYX":
		return EulerOrderZYX, true
	case "YZX":
		return EulerOrderYZX, true
	case "XZY":
		return EulerOrderXZY, true
	default:
		return ronmath.EULER_ORDER_DEFAULT, false
	}
}

func formatEulerOrder(value EulerOrder) (string, bool) {
	switch value {
	case EulerOrderXYZ:
		return "XYZ", true
	case EulerOrderYXZ:
		return "YXZ", true
	case EulerOrderZXY:
		return "ZXY", true
	case EulerOrderZYX:
		return "ZYX", true
	case EulerOrderYZX:
		return "YZX", true
	case EulerOrderXZY:
		return "XZY", true
	default:
		return "", false
	}
}
