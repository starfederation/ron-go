package ron

import (
	"bytes"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Marshal returns value encoded as compact RON without a trailing newline.
func Marshal(value any) ([]byte, error) {
	return marshalWithMode(value, Compact)
}

// MarshalPretty returns value encoded as pretty RON with a trailing newline.
func MarshalPretty(value any) ([]byte, error) {
	return marshalWithMode(value, Pretty)
}

// MarshalCanonical returns value encoded as canonical RON without a trailing newline.
func MarshalCanonical(value any) ([]byte, error) {
	return marshalWithMode(value, Canonical)
}

func marshalWithMode(value any, mode OutputMode) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeMarshaledValue(&buf, value, marshalOptions(Mode(mode))); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Encoder writes RON values to an output stream.
type Encoder struct {
	w    io.Writer
	opts optionState
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer, options ...Option) *Encoder {
	return &Encoder{
		w:    w,
		opts: marshalOptions(options...),
	}
}

// Encode writes the RON encoding of value followed by a newline.
func (e *Encoder) Encode(value any) error {
	var buf bytes.Buffer
	if err := writeMarshaledValue(&buf, value, e.opts); err != nil {
		return err
	}
	if e.opts.mode != Pretty {
		buf.WriteByte('\n')
	}

	written, err := e.w.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if written != buf.Len() {
		return io.ErrShortWrite
	}
	return nil
}

// SetIndent selects pretty RON output indentation. An empty indent uses two spaces.
func (e *Encoder) SetIndent(indent string) {
	if indent == "" {
		indent = "  "
	}
	e.opts.mode = Pretty
	e.opts.indent = indent
}

// SetMode selects pretty, compact, or canonical output.
func (e *Encoder) SetMode(mode OutputMode) {
	switch mode {
	case Pretty, Compact, Canonical:
		e.opts.mode = mode
		if mode == Pretty && e.opts.indent == "" {
			e.opts.indent = "  "
		}
	}
}

func marshalOptions(options ...Option) optionState {
	opts := optionState{
		formatOptions:  formatOptions{mode: Compact},
		vocabularyMask: defaultVocabularySet,
	}
	for _, option := range options {
		option(&opts)
	}
	if opts.mode == Pretty && opts.indent == "" {
		opts.indent = "  "
	}
	if opts.mode != Pretty {
		opts.indent = ""
	}
	return opts
}

func writeMarshaledValue(buf *bytes.Buffer, value any, opts optionState) error {
	normalized, err := marshalRONValue(reflect.ValueOf(value), opts.customRenderersList(), opts.mode == Canonical)
	if err != nil {
		return err
	}
	if opts.mode == Canonical {
		normalized, err = canonicalizeMarshaledValue(normalized)
		if err != nil {
			return err
		}
	}
	if opts.maxNestingDepth > 0 {
		if err := validateMarshaledDepth(normalized, opts.maxNestingDepth, 0); err != nil {
			return err
		}
	}

	if opts.mode == Pretty {
		switch object := normalized.(type) {
		case orderedObject:
			if len(object.Members) > 0 {
				writeObjectMembersWithCustom(buf, objectMembers(object, false), opts.indent, -1, false, opts.customRenderersList())
				buf.WriteByte('\n')
				return nil
			}
		case map[string]any:
			if len(object) > 0 {
				writeObjectMembersWithCustom(buf, objectMembers(object, false), opts.indent, -1, false, opts.customRenderersList())
				buf.WriteByte('\n')
				return nil
			}
		}
		writeValueWithCustom(buf, normalized, opts.indent, 0, false, opts.customRenderersList())
		buf.WriteByte('\n')
		return nil
	}

	writeCompactValueWithCustom(buf, normalized, true, opts.mode == Canonical, opts.customRenderersList())
	return nil
}

func validateMarshaledDepth(value any, maxDepth, depth int) error {
	switch value := value.(type) {
	case []any:
		if depth >= maxDepth {
			return ErrNestingTooDeep
		}
		for _, child := range value {
			if err := validateMarshaledDepth(child, maxDepth, depth+1); err != nil {
				return err
			}
		}
	case multilineArray:
		if depth >= maxDepth {
			return ErrNestingTooDeep
		}
		for _, child := range value {
			if err := validateMarshaledDepth(child, maxDepth, depth+1); err != nil {
				return err
			}
		}
	case map[string]any:
		if depth >= maxDepth {
			return ErrNestingTooDeep
		}
		for _, child := range value {
			if err := validateMarshaledDepth(child, maxDepth, depth+1); err != nil {
				return err
			}
		}
	case orderedObject:
		if depth >= maxDepth {
			return ErrNestingTooDeep
		}
		for _, member := range value.Members {
			if err := validateMarshaledDepth(member.Value, maxDepth, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func marshalRONValue(value reflect.Value, renderers []CustomRenderFunc, canonical bool) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}
	if value.CanInterface() {
		if normalized, ok, err := marshalKnownRONValue(value.Interface(), renderers, canonical); ok || err != nil {
			return normalized, err
		}
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil, nil
		}
		return marshalRONValue(value.Elem(), renderers, canonical)
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint(), nil
	case reflect.Float32, reflect.Float64:
		floatValue := value.Convert(reflect.TypeOf(float64(0))).Float()
		if math.IsInf(floatValue, 0) || math.IsNaN(floatValue) {
			return nil, fmt.Errorf("ron: unsupported float value %v", floatValue)
		}
		return floatValue, nil
	case reflect.Array:
		return marshalArrayValue(value, renderers, canonical)
	case reflect.Slice:
		if value.IsNil() {
			return nil, nil
		}
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return base64.StdEncoding.EncodeToString(value.Bytes()), nil
		}
		return marshalArrayValue(value, renderers, canonical)
	case reflect.Map:
		return marshalMapValue(value, renderers, canonical)
	case reflect.Struct:
		return marshalStructValue(value, renderers, canonical)
	}
	return nil, fmt.Errorf("ron: unsupported type %s", value.Type())
}

func marshalKnownRONValue(value any, renderers []CustomRenderFunc, canonical bool) (any, bool, error) {
	if member, ok := typedTaggedMemberWithCustom(value, renderers); ok {
		payload, err := marshalRONValue(reflect.ValueOf(member.Value), renderers, canonical)
		if err != nil {
			return nil, true, err
		}
		return orderedObject{
			Members: []objectMember{
				{
					Key:   member.Key,
					Value: payload,
				},
			},
		}, true, nil
	}

	if isNilPointerInterface(value) {
		return nil, true, nil
	}
	if marshaler, ok := value.(json.Marshaler); ok {
		body, err := marshaler.MarshalJSON()
		if err != nil {
			return nil, true, err
		}
		if canonical {
			body, err = canonicalJSON(body)
			if err != nil {
				return nil, true, err
			}
		}
		decoded, err := decodeJSON(body, nil)
		if err != nil {
			return nil, true, err
		}
		normalized, err := marshalRONValue(reflect.ValueOf(decoded), renderers, canonical)
		return normalized, true, err
	}
	if marshaler, ok := value.(encoding.TextMarshaler); ok {
		text, err := marshaler.MarshalText()
		if err != nil {
			return nil, true, err
		}
		return string(text), true, nil
	}

	switch value := value.(type) {
	case nil, bool, string, ronNumber, json.Number, int64, uint64, float64:
		return value, true, nil
	case []any:
		if value == nil {
			return nil, true, nil
		}
		array, err := marshalAnySlice(value, renderers, canonical)
		return array, true, err
	case multilineArray:
		if value == nil {
			return nil, true, nil
		}
		array, err := marshalAnySlice([]any(value), renderers, canonical)
		return multilineArray(array), true, err
	case map[string]any:
		if value == nil {
			return nil, true, nil
		}
		object, err := marshalStringMap(value, renderers, canonical)
		return object, true, err
	case orderedObject:
		object, err := marshalOrderedObject(value, renderers, canonical)
		return object, true, err
	}
	return nil, false, nil
}

func isNilPointerInterface(value any) bool {
	reflected := reflect.ValueOf(value)
	return reflected.IsValid() && reflected.Kind() == reflect.Pointer && reflected.IsNil()
}

func marshalAnySlice(value []any, renderers []CustomRenderFunc, canonical bool) ([]any, error) {
	array := make([]any, len(value))
	for i, child := range value {
		normalized, err := marshalRONValue(reflect.ValueOf(child), renderers, canonical)
		if err != nil {
			return nil, err
		}
		array[i] = normalized
	}
	return array, nil
}

func marshalStringMap(value map[string]any, renderers []CustomRenderFunc, canonical bool) (map[string]any, error) {
	object := make(map[string]any, len(value))
	for key, child := range value {
		normalized, err := marshalRONValue(reflect.ValueOf(child), renderers, canonical)
		if err != nil {
			return nil, err
		}
		object[key] = normalized
	}
	return object, nil
}

func marshalOrderedObject(value orderedObject, renderers []CustomRenderFunc, canonical bool) (orderedObject, error) {
	object := orderedObject{Members: make([]objectMember, 0, len(value.Members))}
	for _, member := range value.Members {
		normalized, err := marshalRONValue(reflect.ValueOf(member.Value), renderers, canonical)
		if err != nil {
			return orderedObject{}, err
		}
		object.Set(member.Key, normalized)
	}
	return object, nil
}

func marshalArrayValue(value reflect.Value, renderers []CustomRenderFunc, canonical bool) ([]any, error) {
	array := make([]any, value.Len())
	for i := 0; i < value.Len(); i++ {
		child, err := marshalRONValue(value.Index(i), renderers, canonical)
		if err != nil {
			return nil, err
		}
		array[i] = child
	}
	return array, nil
}

func marshalMapValue(value reflect.Value, renderers []CustomRenderFunc, canonical bool) (map[string]any, error) {
	if value.IsNil() {
		return nil, nil
	}
	object := make(map[string]any, value.Len())
	for _, key := range value.MapKeys() {
		keyString, err := marshalMapKeyString(key)
		if err != nil {
			return nil, err
		}

		child, err := marshalRONValue(value.MapIndex(key), renderers, canonical)
		if err != nil {
			return nil, err
		}
		object[keyString] = child
	}
	return object, nil
}

func marshalMapKeyString(key reflect.Value) (string, error) {
	if key.CanInterface() {
		if marshaler, ok := key.Interface().(encoding.TextMarshaler); ok {
			text, err := marshaler.MarshalText()
			if err != nil {
				return "", err
			}
			return string(text), nil
		}
	}

	switch key.Kind() {
	case reflect.String:
		return key.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(key.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(key.Uint(), 10), nil
	}
	return "", fmt.Errorf("ron: unsupported map key type %s", key.Type())
}

func marshalStructValue(value reflect.Value, renderers []CustomRenderFunc, canonical bool) (orderedObject, error) {
	object := orderedObject{Members: make([]objectMember, 0, value.NumField())}
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name, omitEmpty, skip := marshalFieldName(field)
		if skip {
			continue
		}

		fieldValue := value.Field(i)
		if omitEmpty && isEmptyMarshalValue(fieldValue) {
			continue
		}

		child, err := marshalRONValue(fieldValue, renderers, canonical)
		if err != nil {
			return orderedObject{}, err
		}
		object.Set(name, child)
	}
	return object, nil
}

func marshalFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("ron")
	if tag == "" {
		tag = field.Tag.Get("json")
	}
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return field.Name, false, false
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = field.Name
	}
	omitEmpty := false
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitEmpty = true
		}
	}
	return name, omitEmpty, false
}

func isEmptyMarshalValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return value.IsNil()
	}
	return false
}
