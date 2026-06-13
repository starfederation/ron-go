package ron

import (
	"bytes"
	"encoding/json"
	"io"
)

// Option configures RON and JSON conversion.
type Option func(*formatOptions)

type formatOptions struct {
	prefix      string
	indent      string
	isPretty    bool
	isCanonical bool
}

// PrettyJSON enables indented JSON output.
func PrettyJSON(prefix, indent string) Option {
	return func(opts *formatOptions) {
		opts.isPretty = true
		opts.prefix = prefix
		opts.indent = indent
	}
}

// IsPretty selects multiline pretty output when true or compact output when false.
func IsPretty(pretty bool) Option {
	return func(opts *formatOptions) {
		opts.isPretty = pretty
	}
}

// IsCanonical selects RFC 8785 UTF-16 object key ordering when true.
// When false, source object order is preserved when available; unordered Go maps use canonical order.
func IsCanonical(canonical bool) Option {
	return func(opts *formatOptions) {
		opts.isCanonical = canonical
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

	opts := formatOptions{isCanonical: true}
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
	return func(opts *formatOptions) {
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

	opts := formatOptions{
		indent:      "  ",
		isPretty:    true,
		isCanonical: true,
	}
	for _, option := range options {
		option(&opts)
	}
	if opts.isPretty && opts.indent == "" {
		opts.indent = "  "
	}
	value, err := decodeJSON(src)
	if err != nil {
		return nil, err
	}
	if opts.isPretty {
		dst.Grow(len(src) * 2)
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

func decodeJSON(src []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()

	value, err := decodeJSONValue(dec)
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

func decodeJSONValue(dec *json.Decoder) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch token := token.(type) {
	case nil, bool, string, json.Number:
		return token, nil
	case float64:
		return token, nil
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

				value, err := decodeJSONValue(dec)
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
			return object, nil
		case '[':
			array := make([]any, 0, 4)
			for dec.More() {
				value, err := decodeJSONValue(dec)
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
			return array, nil
		}
	}
	return nil, newError("unexpected JSON token")
}
