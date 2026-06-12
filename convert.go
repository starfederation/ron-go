package ron

import (
	"bytes"
	"encoding/json"
	"io"
)

// ToJSONOption configures RON to JSON conversion.
type ToJSONOption func(*toJSONOptions)

type toJSONOptions struct {
	prefix string
	indent string
}

// PrettyJSON enables indented JSON output.
func PrettyJSON(prefix, indent string) ToJSONOption {
	return func(opts *toJSONOptions) {
		opts.prefix = prefix
		opts.indent = indent
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
	opts := toJSONOptions{}
	for _, option := range options {
		option(&opts)
	}
	b.buf = b.buf[:0]
	if err := writeRONJSONInto(&b.buf, src, opts.prefix, opts.indent); err != nil {
		return nil, err
	}
	return b.buf, nil
}

// ToJSON converts RON to compact JSON unless PrettyJSON is provided.
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
type FromJSONOption func(*fromJSONOptions)

type fromJSONOptions struct {
	indent string
}

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
	return func(opts *fromJSONOptions) {
		if indent == "" {
			opts.indent = "  "
			return
		}
		opts.indent = indent
	}
}

// FromJSON converts JSON to pretty RON using this builder.
func (b *RONBuilder) FromJSON(src []byte, options ...FromJSONOption) ([]byte, error) {
	opts := fromJSONOptions{indent: "  "}
	for _, option := range options {
		option(&opts)
	}
	value, err := decodeJSON(src)
	if err != nil {
		return nil, err
	}
	b.resetCap(len(src) * 2)
	writeValue(&b.buf, value, opts.indent, 0)
	writeByte(&b.buf, '\n')
	return b.buf, nil
}

// FromJSONCompact converts JSON to compact RON using this builder.
func (b *RONBuilder) FromJSONCompact(src []byte) ([]byte, error) {
	value, err := decodeJSON(src)
	if err != nil {
		return nil, err
	}
	b.resetCap(len(src))
	if object, ok := value.(map[string]any); ok {
		writeCompactObject(&b.buf, object, true)
	} else {
		writeCompactValue(&b.buf, value)
	}
	return b.buf, nil
}

// FromJSON converts JSON to pretty RON.
func FromJSON(src []byte, options ...FromJSONOption) ([]byte, error) {
	var builder RONBuilder
	return builder.FromJSON(src, options...)
}

// FromJSONInto converts JSON to pretty RON using dst when non-nil.
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
	var value any
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}

	var trailing any
	if err := dec.Decode(&trailing); err == nil {
		return nil, newError("unexpected trailing JSON")
	} else if err != io.EOF {
		return nil, err
	}
	return value, nil
}
