package ron

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	colorlib "github.com/SCKelemen/color"
)

type vocabularySet uint16

const (
	vocabularyCore vocabularySet = 1 << iota
	vocabularyTime
	vocabularyNetwork
	vocabularyMath
	vocabularySpatial
	vocabularyGeo
	vocabularyColor

	defaultVocabularySet = vocabularyCore | vocabularyTime | vocabularyNetwork | vocabularyMath | vocabularySpatial | vocabularyGeo | vocabularyColor
)

// EnableVocabularies enables validation for supported typed vocabulary URIs.
// Supported vocabularies are enabled by default; use this for explicit profiles.
// Unsupported typed values remain ordinary JSON/RON objects unless their vocabulary is enabled.
func EnableVocabularies(uris ...string) Option {
	return func(opts *optionState) {
		for _, uri := range uris {
			if opts.enableBuiltInVocabulary(uri) {
				continue
			}
			if opts.vocabularies == nil {
				opts.vocabularies = make(map[string]struct{}, len(uris))
			}
			opts.vocabularies[uri] = struct{}{}
		}
	}
}

func defaultVocabularies() map[string]struct{} {
	return map[string]struct{}{
		VocabularyCoreV1:    {},
		VocabularyTimeV1:    {},
		VocabularyNetworkV1: {},
		VocabularyMathV1:    {},
		VocabularySpatialV1: {},
		VocabularyGeoV1:     {},
		VocabularyColorV1:   {},
	}
}

func (opts optionState) hasVocabularies() bool {
	return opts.vocabularyMask != 0 || len(opts.vocabularies) > 0
}

func (opts *optionState) enableBuiltInVocabulary(uri string) bool {
	switch uri {
	case VocabularyCoreV1:
		opts.vocabularyMask |= vocabularyCore
	case VocabularyTimeV1:
		opts.vocabularyMask |= vocabularyTime
	case VocabularyNetworkV1:
		opts.vocabularyMask |= vocabularyNetwork
	case VocabularyMathV1:
		opts.vocabularyMask |= vocabularyMath
	case VocabularySpatialV1:
		opts.vocabularyMask |= vocabularySpatial
	case VocabularyGeoV1:
		opts.vocabularyMask |= vocabularyGeo
	case VocabularyColorV1:
		opts.vocabularyMask |= vocabularyColor
	default:
		return false
	}
	return true
}

func (opts optionState) vocabularyEnabled(vocabulary vocabularySet, uri string) bool {
	if opts.vocabularyMask&vocabulary != 0 {
		return true
	}
	_, ok := opts.vocabularies[uri]
	return ok
}

func (opts optionState) parseVocabularies(value any) (any, error) {
	if err := opts.validateVocabularies(); err != nil {
		return nil, err
	}
	return opts.parseVocabularyValue(value)
}

func (opts optionState) validateVocabularies() error {
	for uri := range opts.vocabularies {
		if !opts.supportsVocabulary(uri) {
			return newError("unsupported vocabulary: " + uri)
		}
	}
	return nil
}

func (opts optionState) parseVocabularyValue(value any) (any, error) {
	switch value := value.(type) {
	case []any:
		for i, child := range value {
			parsed, err := opts.parseVocabularyValue(child)
			if err != nil {
				return nil, err
			}
			value[i] = parsed
		}
		return value, nil
	case map[string]any:
		for key, child := range value {
			if opts.isCoreTag(key) || opts.isTimeTag(key) || opts.isNetworkTag(key) || opts.isMathTag(key) || opts.isSpatialTag(key) || opts.isGeoTag(key) || opts.isColorTag(key) || opts.isCustomTag(key) {
				if len(value) != 1 {
					return nil, newError("typed vocabulary object must have exactly one member")
				}
				return opts.parseTypedPayload(key, child)
			}
		}
		for key, child := range value {
			parsed, err := opts.parseVocabularyValue(child)
			if err != nil {
				return nil, err
			}
			value[key] = parsed
		}
		return value, nil
	case orderedObject:
		if tag, payload, ok := opts.enabledTypedValue(value.Members); ok {
			if len(value.Members) != 1 {
				return nil, newError("typed vocabulary object must have exactly one member")
			}
			return opts.parseTypedPayload(tag, payload)
		}
		for i, member := range value.Members {
			parsed, err := opts.parseVocabularyValue(member.Value)
			if err != nil {
				return nil, err
			}
			value.Members[i].Value = parsed
		}
		return value, nil
	default:
		return value, nil
	}
}

func (opts optionState) enabledTypedValue(members []objectMember) (string, any, bool) {
	for _, member := range members {
		if opts.isCoreTag(member.Key) || opts.isTimeTag(member.Key) || opts.isNetworkTag(member.Key) || opts.isMathTag(member.Key) || opts.isSpatialTag(member.Key) || opts.isGeoTag(member.Key) || opts.isColorTag(member.Key) || opts.isCustomTag(member.Key) {
			return member.Key, member.Value, true
		}
	}
	return "", nil, false
}

func (opts optionState) parseTypedPayload(tag string, payload any) (any, error) {
	if opts.isCoreTag(tag) {
		return opts.parseCorePayload(tag, payload)
	}
	if opts.isTimeTag(tag) {
		return opts.parseTimePayload(tag, payload)
	}
	if opts.isNetworkTag(tag) {
		return opts.parseNetworkPayload(tag, payload)
	}
	if opts.isMathTag(tag) {
		return opts.parseMathPayload(tag, payload)
	}
	if opts.isSpatialTag(tag) {
		return opts.parseSpatialPayload(tag, payload)
	}
	if opts.isGeoTag(tag) {
		return opts.parseGeoPayload(tag, payload)
	}
	if opts.isColorTag(tag) {
		return opts.parseColorPayload(tag, payload)
	}
	if opts.isCustomTag(tag) {
		return opts.parseCustomPayload(tag, payload)
	}
	return nil, newError("unsupported typed tag")
}

func typedTaggedMemberWithCustom(value any, renderers []CustomRenderFunc) (objectMember, bool) {
	if len(renderers) == 0 {
		switch value.(type) {
		case nil, bool, string, ronNumber, json.Number, int64, uint64, float64, []any, map[string]any, orderedObject:
			return objectMember{}, false
		}
	}

	switch value := value.(type) {
	case UUID:
		return objectMember{Key: "#uid", Value: value.String()}, true
	case *url.URL:
		if value == nil {
			return objectMember{}, false
		}
		return objectMember{Key: "#url", Value: value.String()}, true
	case RegExp:
		return regExpTaggedMember(value), true
	case *RegExp:
		if value == nil {
			return objectMember{}, false
		}
		return regExpTaggedMember(*value), true
	case Decimal:
		return objectMember{Key: "#dec", Value: canonicalDecimalString(&value)}, true
	case *Decimal:
		if value == nil {
			return objectMember{}, false
		}
		return objectMember{Key: "#dec", Value: canonicalDecimalString(value)}, true
	case Bytes:
		return objectMember{Key: "#b64", Value: base64.RawURLEncoding.EncodeToString(value)}, true
	case SHA256:
		return objectMember{Key: "#sha256", Value: hex.EncodeToString(value[:])}, true
	case EntityRef:
		return objectMember{Key: "#", Value: value.Value}, true
	case OpaqueTag:
		return objectMember{
			Key: "#tag",
			Value: []any{
				value.Tag,
				value.Payload,
			},
		}, true
	case time.Time:
		if value.Location() != time.UTC {
			value = value.UTC()
		}
		return objectMember{Key: "#utc", Value: value.Format(time.RFC3339Nano)}, true
	case time.Duration:
		if value == 0 {
			return objectMember{Key: "#dur", Value: "PT0S"}, true
		}
		negative := value < 0
		if negative {
			value = -value
		}
		days := value / (24 * time.Hour)
		value -= days * 24 * time.Hour
		hours := value / time.Hour
		value -= hours * time.Hour
		minutes := value / time.Minute
		value -= minutes * time.Minute
		seconds := value / time.Second
		nanos := value - seconds*time.Second

		var b strings.Builder
		if negative {
			b.WriteByte('-')
		}
		b.WriteByte('P')
		if days > 0 {
			b.WriteString(strconv.FormatInt(int64(days), 10))
			b.WriteByte('D')
		}
		if hours > 0 || minutes > 0 || seconds > 0 || nanos > 0 || days == 0 {
			b.WriteByte('T')
			if hours > 0 {
				b.WriteString(strconv.FormatInt(int64(hours), 10))
				b.WriteByte('H')
			}
			if minutes > 0 {
				b.WriteString(strconv.FormatInt(int64(minutes), 10))
				b.WriteByte('M')
			}
			if seconds > 0 || nanos > 0 || (hours == 0 && minutes == 0 && days == 0) {
				b.WriteString(strconv.FormatInt(int64(seconds), 10))
				if nanos > 0 {
					fraction := strconv.FormatInt(int64(nanos)+1_000_000_000, 10)[1:]
					fraction = strings.TrimRight(fraction, "0")
					b.WriteByte('.')
					b.WriteString(fraction)
				}
				b.WriteByte('S')
			}
		}
		return objectMember{Key: "#dur", Value: b.String()}, true
	case IPv4:
		if !value.Addr.Is4() {
			return objectMember{}, false
		}
		return objectMember{Key: "#ip4", Value: value.Addr.String()}, true
	case IPv6:
		if !value.Addr.Is6() || value.Addr.Is4In6() {
			return objectMember{}, false
		}
		return objectMember{Key: "#ip6", Value: value.Addr.String()}, true
	case CIDR:
		if value.Prefix != value.Prefix.Masked() {
			return objectMember{}, false
		}
		return objectMember{Key: "#cdr", Value: value.Prefix.String()}, true
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
		var order string
		switch value.Order {
		case EulerOrderXYZ:
			order = "XYZ"
		case EulerOrderYXZ:
			order = "YXZ"
		case EulerOrderZXY:
			order = "ZXY"
		case EulerOrderZYX:
			order = "ZYX"
		case EulerOrderYZX:
			order = "YZX"
		case EulerOrderXZY:
			order = "XZY"
		default:
			return objectMember{}, false
		}
		return objectMember{Key: "#eul", Value: []any{value.X, value.Y, value.Z, order}}, true
	case Matrix2:
		return objectMember{Key: "#m2x", Value: floatSliceToAny(value[:])}, true
	case Matrix3:
		return objectMember{Key: "#m3x", Value: floatSliceToAny(value[:])}, true
	case Matrix4:
		return objectMember{Key: "#m4x", Value: floatSliceToAny(value[:])}, true
	case LngLatAlt:
		return objectMember{Key: "#lla", Value: []any{value.Point.X, value.Point.Y, value.Altitude}}, true
	case Spherical:
		return objectMember{Key: "#sph", Value: []any{value.Radius, value.Phi, value.Theta}}, true
	case Cylindrical:
		return objectMember{Key: "#cyl", Value: []any{value.Radius, value.Theta, value.Y}}, true
	case Box2:
		return objectMember{Key: "#bx2", Value: []any{anyVector2(value.Min), anyVector2(value.Max)}}, true
	case Box3:
		return objectMember{Key: "#bx3", Value: []any{anyVector3(value.Min), anyVector3(value.Max)}}, true
	case Sphere:
		return objectMember{Key: "#spr", Value: []any{anyVector3(value.Center), value.Radius}}, true
	case Plane:
		return objectMember{Key: "#pln", Value: []any{anyVector3(value.Normal), value.Constant}}, true
	case Ray:
		return objectMember{Key: "#ray", Value: []any{anyVector3(value.Origin), anyVector3(value.Dir)}}, true
	case Line2:
		return objectMember{Key: "#ln2", Value: []any{anyVector2(value.Start), anyVector2(value.End)}}, true
	case Line3:
		return objectMember{Key: "#ln3", Value: []any{anyVector3(value.Start), anyVector3(value.End)}}, true
	case Triangle:
		return objectMember{Key: "#tri", Value: []any{anyVector3(value.A), anyVector3(value.B), anyVector3(value.C)}}, true
	case Frustum:
		planes := make([]any, len(value))
		for i, plane := range value {
			planes[i] = []any{anyVector3(plane.Normal), plane.Constant}
		}
		return objectMember{Key: "#fru", Value: planes}, true
	case SphericalHarmonics3:
		vectors := make([]any, len(value))
		for i, vector := range value {
			vectors[i] = anyVector3(vector)
		}
		return objectMember{Key: "#sh3", Value: vectors}, true
	case VoxelSet:
		var object orderedObject
		object.Set("dimensions", value.Dimensions)
		object.Set("origin", VectorN(value.Origin))
		object.Set("cellSize", VectorN(value.CellSize))
		cells := make([]any, len(value.Cells))
		for i, cell := range value.Cells {
			cells[i] = []any{intSliceToAny(cell.Coordinate), cell.Value}
		}
		if len(cells) > 0 {
			object.Set("cells", multilineArray(cells))
		} else {
			object.Set("cells", cells)
		}
		return objectMember{Key: "#vox", Value: object}, true
	case GeoJSON:
		return objectMember{Key: "#geo", Value: value.Data}, true
	case Color:
		return colorMember(value), true
	case *colorlib.OKLCH:
		if value == nil {
			return objectMember{}, false
		}
		return colorMember(Color{
			Space: ColorSpaceOKLCH,
			Channels: []float64{
				value.L,
				value.C,
				value.H,
			},
			Value: value,
		}), true
	case colorlib.Color:
		if value == nil {
			return objectMember{}, false
		}
		oklch := colorlib.ToOKLCH(value)
		space := ColorSpaceOKLCH
		channels := []float64{
			oklch.L,
			oklch.C,
			oklch.H,
		}
		if oklch.Alpha() != 1 {
			space = ColorSpaceOKLCHA
			channels = append(channels, oklch.Alpha())
		}
		return colorMember(Color{Space: space, Channels: channels, Value: value}), true
	case CustomValue:
		return objectMember{Key: normalizeCustomTag(value.Tag), Value: value.Payload}, true
	case *CustomValue:
		if value == nil {
			return objectMember{}, false
		}
		return objectMember{Key: normalizeCustomTag(value.Tag), Value: value.Payload}, true
	}
	for _, render := range renderers {
		tag, payload, ok := render(value)
		if ok {
			return objectMember{Key: normalizeCustomTag(tag), Value: payload}, true
		}
	}
	return objectMember{}, false
}
