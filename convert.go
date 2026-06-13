package ron

import (
	"bytes"
	"encoding/json"
	"io"
)

// Option configures RON and JSON conversion.
type Option func(*formatOptions)

// ToJSONOption configures RON to JSON conversion.
type ToJSONOption = Option

type formatOptions struct {
	prefix      string
	indent      string
	isPretty    bool
	isCanonical bool
}

// PrettyJSON enables indented JSON output.
func PrettyJSON(prefix, indent string) ToJSONOption {
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

// IsCanonical selects lexicographic object key ordering when true.
// When false, source object order is preserved when available; unordered Go maps use sorted order.
func IsCanonical(canonical bool) Option {
	return func(opts *formatOptions) {
		opts.isCanonical = canonical
	}
}

// JSONBuilder reuses memory across RON to JSON conversions.
type JSONBuilder struct {
	buf jsonBytes
}

// Reset releases the currently buffered JSON bytes while retaining capacity.
func (b *JSONBuilder) Reset() {
	b.buf = b.buf[:0]
}

// ToJSON converts RON to JSON using this builder.
func (b *JSONBuilder) ToJSON(src []byte, options ...ToJSONOption) ([]byte, error) {
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
	b.buf = b.buf[:0]
	if err := writeRONJSONInto(&b.buf, src, opts.prefix, opts.indent, opts.isCanonical); err != nil {
		return nil, err
	}
	return b.buf, nil
}

// ToJSON converts RON to compact JSON unless pretty output is requested.
func ToJSON(src []byte, options ...ToJSONOption) ([]byte, error) {
	var builder JSONBuilder
	return builder.ToJSON(src, options...)
}

// ToJSONInto converts RON to JSON using dst when non-nil.
func ToJSONInto(dst *JSONBuilder, src []byte, options ...ToJSONOption) ([]byte, error) {
	if dst == nil {
		return ToJSON(src, options...)
	}
	return dst.ToJSON(src, options...)
}

// FromJSONOption configures JSON to RON conversion.
type FromJSONOption = Option

// RONBuilder reuses memory across JSON to RON conversions.
type RONBuilder struct {
	buf jsonBytes
}

// Reset releases the currently buffered RON bytes while retaining capacity.
func (b *RONBuilder) Reset() {
	b.buf = b.buf[:0]
}

func (b *RONBuilder) resetCap(capacity int) {
	if cap(b.buf) < capacity {
		b.buf = make(jsonBytes, 0, capacity)
		return
	}
	b.buf = b.buf[:0]
}

// Indent sets the pretty RON indentation string.
func Indent(indent string) FromJSONOption {
	return func(opts *formatOptions) {
		if indent == "" {
			opts.indent = "  "
			return
		}
		opts.indent = indent
	}
}

// FromJSON converts JSON to RON using this builder.
func (b *RONBuilder) FromJSON(src []byte, options ...FromJSONOption) ([]byte, error) {
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
		b.resetCap(len(src) * 2)
		writeValue(&b.buf, value, opts.indent, 0, opts.isCanonical)
		writeByte(&b.buf, '\n')
		return b.buf, nil
	}

	b.resetCap(len(src))
	writeCompactValue(&b.buf, value, true, opts.isCanonical)
	return b.buf, nil
}

// FromJSONCompact converts JSON to compact RON using this builder.
func (b *RONBuilder) FromJSONCompact(src []byte) ([]byte, error) {
	return b.FromJSON(src, IsPretty(false))
}

// FromJSON converts JSON to pretty RON unless compact output is requested.
func FromJSON(src []byte, options ...FromJSONOption) ([]byte, error) {
	var builder RONBuilder
	return builder.FromJSON(src, options...)
}

// FromJSONInto converts JSON to RON using dst when non-nil.
func FromJSONInto(dst *RONBuilder, src []byte, options ...FromJSONOption) ([]byte, error) {
	if dst == nil {
		return FromJSON(src, options...)
	}
	return dst.FromJSON(src, options...)
}

// FromJSONCompact converts JSON to compact RON.
func FromJSONCompact(src []byte) ([]byte, error) {
	var builder RONBuilder
	return builder.FromJSONCompact(src)
}

// FromJSONCompactInto converts JSON to compact RON using dst when non-nil.
func FromJSONCompactInto(dst *RONBuilder, src []byte) ([]byte, error) {
	if dst == nil {
		return FromJSONCompact(src)
	}
	return dst.FromJSONCompact(src)
}

// MarshalCompact returns value as compact RON.
func MarshalCompact(value any) ([]byte, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var builder RONBuilder
	return builder.FromJSONCompact(body)
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
