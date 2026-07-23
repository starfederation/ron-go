package ron

import (
	"bytes"
	"encoding/json"
	"io"
)

// Option configures RON and JSON conversion or stream processing.
type Option func(*optionState)

type optionState struct {
	formatOptions
	jsonValueMapper       jsonValueMapper
	vocabularyMask        vocabularyMask
	vocabularies          map[string]struct{}
	customVocabularies    map[string]CustomVocabulary
	customVocabularyOrder []string
	customTags            map[string]string
	maxRecordSize         int
	maxNestingDepth       int
	ignoreEmptyNdronLines bool
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

// MaxRecordSize limits the encoded RON bytes in each stream record, excluding framing.
// Non-positive values use the default limit.
func MaxRecordSize(size int) Option {
	return func(opts *optionState) {
		opts.maxRecordSize = size
	}
}

// MaxNestingDepth limits nested RON arrays and objects during decoding and stream validation.
// Non-positive values use the stream default or remain unlimited for non-stream conversions.
func MaxNestingDepth(depth int) Option {
	return func(opts *optionState) {
		opts.maxNestingDepth = depth
	}
}

// IgnoreEmptyNdronLines controls whether NDRON decoders skip empty lines.
func IgnoreEmptyNdronLines(ignore bool) Option {
	return func(opts *optionState) {
		opts.ignoreEmptyNdronLines = ignore
	}
}

// MapJSONValues transforms decoded JSON values before JSON-to-RON rendering.
func MapJSONValues(mapper func(path []JSONPathSegment, value any) (any, bool)) Option {
	return func(opts *optionState) {
		opts.jsonValueMapper = mapper
	}
}

// ToJSON converts RON to one JSON value without a trailing newline.
// Output is compact unless pretty output is requested.
func ToJSON(src []byte, options ...Option) ([]byte, error) {
	var buf bytes.Buffer
	return ToJSONInto(&buf, src, options...)
}

// ToJSONInto appends one RON value converted to JSON to dst without a trailing newline.
func ToJSONInto(dst *bytes.Buffer, src []byte, options ...Option) ([]byte, error) {
	if dst == nil {
		return ToJSON(src, options...)
	}

	opts := optionState{
		formatOptions:  formatOptions{isCanonical: true},
		vocabularyMask: defaultVocabularySet,
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
		if err := opts.validateVocabularies(); err != nil {
			return nil, err
		}
		if containsVocabularyMarker(src) {
			value, err := parseWithMaxDepth(src, opts.maxNestingDepth)
			if err != nil {
				return nil, err
			}
			if _, err := opts.parseVocabularyValue(value); err != nil {
				return nil, err
			}
		}
	}

	scratch := jsonScratchPool.Get().(*jsonScratchState)
	p := parser{
		src:               src,
		maxNestingDepth:   opts.maxNestingDepth,
		jsonScratch:       scratch.buffers,
		jsonMemberScratch: scratch.members,
	}
	defer func() {
		scratch.buffers = p.jsonScratch
		scratch.members = p.jsonMemberScratch
		for i := range scratch.buffers {
			if scratch.buffers[i].Cap() > 1<<20 {
				scratch.buffers[i] = bytes.Buffer{}
			} else {
				scratch.buffers[i].Reset()
			}
		}
		for i := range scratch.members {
			if cap(scratch.members[i]) > 1024 {
				scratch.members[i] = nil
			} else {
				scratch.members[i] = scratch.members[i][:0]
			}
		}
		jsonScratchPool.Put(scratch)
	}()
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] != '{' && p.src[p.pos] != '[' {
		p.nestingDepth = 1
		start := p.pos
		bufStart := dst.Len()
		memberScratch, memberValues := p.nextJSONMembers()
		members := jsonMembers{Values: memberValues}
		values := p.nextJSONScratch()
		for {
			p.skipSpace()
			if p.pos == len(p.src) {
				if err := p.writeJSONObjectMembers(dst, members.Values, values.Bytes(), opts.prefix, opts.indent, 0, opts.isCanonical && members.NeedsSort); err != nil {
					p.releaseJSONScratch()
					p.releaseJSONMembers(memberScratch, members.Values)
					return nil, err
				}
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				return dst.Bytes(), nil
			}
			if p.src[p.pos] == '{' || p.src[p.pos] == '[' {
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				break
			}

			key, err := p.parseKeyCurrent()
			if err != nil {
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				break
			}

			valueStart := values.Len()
			if err := p.writeJSONValue(values, opts.prefix, opts.indent, 1, opts.isCanonical); err != nil {
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				if err == ErrNestingTooDeep {
					return nil, err
				}
				break
			}
			members.Set(key, valueStart, values.Len())
		}
		dst.Truncate(bufStart)
		p.pos = start
		p.nestingDepth = 0
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

func containsVocabularyMarker(src []byte) bool {
	return bytes.IndexByte(src, '#') >= 0 || bytes.Contains(src, []byte(`\u0023`))
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

// FromJSON converts JSON to one RON value without a trailing newline.
// Output is pretty unless compact output is requested.
func FromJSON(src []byte, options ...Option) ([]byte, error) {
	var buf bytes.Buffer
	return FromJSONInto(&buf, src, options...)
}

// FromJSONInto appends one JSON value converted to RON to dst without a trailing newline.
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
		vocabularyMask: defaultVocabularySet,
	}
	for _, option := range options {
		option(&opts)
	}
	if opts.isPretty && opts.indent == "" {
		opts.indent = "  "
	}
	var value any
	var err error
	if opts.isCanonical && !opts.isPretty && opts.jsonValueMapper == nil {
		dec := json.NewDecoder(bytes.NewReader(src))
		dec.UseNumber()
		err = dec.Decode(&value)
		if err == nil {
			var trailing any
			if trailingErr := dec.Decode(&trailing); trailingErr == nil {
				err = newError("unexpected trailing JSON")
			} else if trailingErr != io.EOF {
				err = trailingErr
			}
		}
	} else {
		value, err = decodeJSON(src, opts.jsonValueMapper)
	}
	if err != nil {
		return nil, err
	}
	if opts.hasVocabularies() && (opts.jsonValueMapper != nil || containsVocabularyMarker(src)) {
		value, err = opts.parseVocabularies(value)
		if err != nil {
			return nil, err
		}
	}
	if opts.isPretty {
		dst.Grow(len(src) * 2)
		if object, ok := value.(orderedObject); ok && len(object.Members) > 0 {
			writeObjectMembersWithCustom(dst, objectMembers(object, opts.isCanonical), opts.indent, -1, opts.isCanonical, opts.customRenderersList())
			return dst.Bytes(), nil
		}
		writeValueWithCustom(dst, value, opts.indent, 0, opts.isCanonical, opts.customRenderersList())
		return dst.Bytes(), nil
	}

	dst.Grow(len(src))
	writeCompactValueWithCustom(dst, value, true, opts.isCanonical, opts.customRenderersList())
	return dst.Bytes(), nil
}

// FromJSONCompact converts JSON to one compact RON value without a trailing newline.
func FromJSONCompact(src []byte) ([]byte, error) {
	return FromJSON(src, IsPretty(false))
}

// FromJSONCompactInto appends one JSON value converted to compact RON to dst without a trailing newline.
func FromJSONCompactInto(dst *bytes.Buffer, src []byte) ([]byte, error) {
	return FromJSONInto(dst, src, IsPretty(false))
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
			object := orderedObject{Members: make([]objectMember, 0, 8)}
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
