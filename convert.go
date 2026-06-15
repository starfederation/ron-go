package ron

import (
	"bytes"
	"encoding/json"
	"io"
)

// Option configures RON and JSON conversion.
type Option func(*optionState)

type optionState struct {
	formatOptions
	jsonValueMapper jsonValueMapper
	vocabularies    map[string]struct{}
}

type formatOptions struct {
	prefix      string
	indent      string
	isPretty    bool
	isCanonical bool
}

// JSONPathSegment identifies one object key or array index in a decoded JSON value.
type JSONPathSegment struct {
	Key     string
	Index   int
	IsIndex bool
}

type jsonValueMapper func(path []JSONPathSegment, value any) (any, bool)

// Tagged returns a single-member object that renders as a tagged RON form.
// An empty tag renders as #. Payload validation is left to application-specific typed hooks.
func Tagged(tag string, value any) map[string]any {
	return map[string]any{
		"#" + tag: value,
	}
}

// PrettyJSON enables indented JSON output.
func PrettyJSON(prefix, indent string) Option {
	return func(opts *optionState) {
		opts.isPretty = true
		opts.prefix = prefix
		opts.indent = indent
	}
}

// IsPretty selects multiline pretty output when true or compact output when false.
func IsPretty(pretty bool) Option {
	return func(opts *optionState) {
		opts.isPretty = pretty
	}
}

// IsCanonical selects RFC 8785 UTF-16 object key ordering when true.
// When false, source object order is preserved when available; unordered Go maps use canonical order.
func IsCanonical(canonical bool) Option {
	return func(opts *optionState) {
		opts.isCanonical = canonical
	}
}

// MapJSONValues transforms decoded JSON values before JSON-to-RON rendering.
func MapJSONValues(mapper func(path []JSONPathSegment, value any) (any, bool)) Option {
	return func(opts *optionState) {
		opts.jsonValueMapper = mapper
	}
}

// ToJSON converts RON to compact JSON unless pretty output is requested.
func ToJSON(src []byte, options ...Option) ([]byte, error) {
	var buf bytes.Buffer
	return ToJSONInto(&buf, src, options...)
}

// ToJSONInto appends RON converted to JSON to dst.
func ToJSONInto(dst *bytes.Buffer, src []byte, options ...Option) ([]byte, error) {
	if dst == nil {
		return ToJSON(src, options...)
	}

	opts := optionState{
		formatOptions: formatOptions{isCanonical: true},
		vocabularies:  defaultVocabularies(),
	}
	for _, option := range options {
		option(&opts)
	}
	if opts.isPretty && opts.indent == "" {
		opts.indent = "  "
	}
	if !opts.isPretty {
		opts.prefix = ""
		opts.indent = ""
	}
	if opts.hasVocabularies() {
		value, err := parse(src)
		if err != nil {
			return nil, err
		}
		if _, err := opts.parseVocabularies(value); err != nil {
			return nil, err
		}
	}

	p := parser{src: src}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] != '{' && p.src[p.pos] != '[' {
		start := p.pos
		bufStart := dst.Len()
		members := jsonMembers{Values: make([]jsonMember, 0, 4)}
		var values bytes.Buffer
		for {
			p.skipSpace()
			if p.pos == len(p.src) {
				if err := p.writeJSONObjectMembers(dst, members.Values, values.Bytes(), opts.prefix, opts.indent, 0, opts.isCanonical); err != nil {
					return nil, err
				}
				return dst.Bytes(), nil
			}
			if p.src[p.pos] == '{' || p.src[p.pos] == '[' {
				break
			}

			key, err := p.parseKeyCurrent()
			if err != nil {
				break
			}

			valueStart := values.Len()
			if err := p.writeJSONValue(&values, opts.prefix, opts.indent, 1, opts.isCanonical); err != nil {
				break
			}
			members.Set(key, valueStart, values.Len())
		}
		dst.Truncate(bufStart)
		p.pos = start
	}

	if err := p.writeJSONValue(dst, opts.prefix, opts.indent, 0, opts.isCanonical); err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos != len(p.src) {
		return nil, p.errorf("unexpected trailing data")
	}
	return dst.Bytes(), nil
}

// Indent sets the pretty RON indentation string.
func Indent(indent string) Option {
	return func(opts *optionState) {
		if indent == "" {
			opts.indent = "  "
			return
		}
		opts.indent = indent
	}
}

// FromJSON converts JSON to pretty RON unless compact output is requested.
func FromJSON(src []byte, options ...Option) ([]byte, error) {
	var buf bytes.Buffer
	return FromJSONInto(&buf, src, options...)
}

// FromJSONInto appends JSON converted to RON to dst.
func FromJSONInto(dst *bytes.Buffer, src []byte, options ...Option) ([]byte, error) {
	if dst == nil {
		return FromJSON(src, options...)
	}

	opts := optionState{
		formatOptions: formatOptions{
			indent:      "  ",
			isPretty:    true,
			isCanonical: true,
		},
		vocabularies: defaultVocabularies(),
	}
	for _, option := range options {
		option(&opts)
	}
	if opts.isPretty && opts.indent == "" {
		opts.indent = "  "
	}
	value, err := decodeJSON(src, opts.jsonValueMapper)
	if err != nil {
		return nil, err
	}
	if opts.hasVocabularies() {
		value, err = opts.parseVocabularies(value)
		if err != nil {
			return nil, err
		}
	}
	if opts.isPretty {
		dst.Grow(len(src) * 2)
		if object, ok := value.(orderedObject); ok && len(object.Members) > 0 {
			writeObjectMembers(dst, objectMembers(object, opts.isCanonical), opts.indent, -1, opts.isCanonical)
			dst.WriteByte('\n')
			return dst.Bytes(), nil
		}
		writeValue(dst, value, opts.indent, 0, opts.isCanonical)
		dst.WriteByte('\n')
		return dst.Bytes(), nil
	}

	dst.Grow(len(src))
	writeCompactValue(dst, value, true, opts.isCanonical)
	return dst.Bytes(), nil
}

// FromJSONCompact converts JSON to compact RON.
func FromJSONCompact(src []byte) ([]byte, error) {
	return FromJSON(src, IsPretty(false))
}

// FromJSONCompactInto appends JSON converted to compact RON to dst.
func FromJSONCompactInto(dst *bytes.Buffer, src []byte) ([]byte, error) {
	return FromJSONInto(dst, src, IsPretty(false))
}

// MarshalCompact returns value as compact RON.
func MarshalCompact(value any) ([]byte, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return FromJSONCompact(body)
}

func decodeJSON(src []byte, mapper jsonValueMapper) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()

	value, err := decodeJSONValue(dec, nil, mapper)
	if err != nil {
		return nil, err
	}

	if _, err := dec.Token(); err == nil {
		return nil, newError("unexpected trailing JSON")
	} else if err != io.EOF {
		return nil, err
	}
	return value, nil
}

func decodeJSONValue(dec *json.Decoder, path []JSONPathSegment, mapper jsonValueMapper) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch token := token.(type) {
	case nil, bool, string, json.Number:
		return mapJSONValue(path, token, mapper), nil
	case float64:
		return mapJSONValue(path, token, mapper), nil
	case json.Delim:
		switch token {
		case '{':
			var object orderedObject
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, newError("expected JSON object key")
				}

				value, err := decodeJSONValue(dec, append(path, JSONPathSegment{Key: key}), mapper)
				if err != nil {
					return nil, err
				}
				object.Set(key, value)
			}

			end, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if end != json.Delim('}') {
				return nil, newError("expected JSON object end")
			}
			return mapJSONValue(path, object, mapper), nil
		case '[':
			array := make([]any, 0, 4)
			for dec.More() {
				value, err := decodeJSONValue(dec, append(path, JSONPathSegment{Index: len(array), IsIndex: true}), mapper)
				if err != nil {
					return nil, err
				}
				array = append(array, value)
			}

			end, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if end != json.Delim(']') {
				return nil, newError("expected JSON array end")
			}
			return mapJSONValue(path, array, mapper), nil
		}
	}
	return nil, newError("unexpected JSON token")
}

func mapJSONValue(path []JSONPathSegment, value any, mapper jsonValueMapper) any {
	if mapper == nil {
		return value
	}
	mapped, ok := mapper(append([]JSONPathSegment(nil), path...), value)
	if !ok {
		return value
	}
	return mapped
}
