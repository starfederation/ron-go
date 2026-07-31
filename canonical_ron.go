package ron

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"unicode/utf8"
)

func canonicalRONToJSON(src []byte, maxNestingDepth int) ([]byte, error) {
	p := parser{src: src, maxNestingDepth: maxNestingDepth}
	p.skipSpace()
	if p.pos < len(src) && src[p.pos] != '{' && src[p.pos] != '[' {
		start := p.pos
		p.nestingDepth = 1
		object, ok := p.parseCanonicalElidedObject()
		if ok {
			var buf bytes.Buffer
			if err := writeRFC8785Value(&buf, object); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
		p.pos = start
		p.nestingDepth = 0
	}
	value, err := p.parseCanonicalValue()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos != len(src) {
		return nil, p.errorf("unexpected trailing data")
	}
	var buf bytes.Buffer
	if err := writeRFC8785Value(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *parser) parseCanonicalElidedObject() (orderedObject, bool) {
	object := orderedObject{}
	for {
		p.skipSpace()
		if p.pos == len(p.src) {
			return object, len(object.Members) > 0
		}
		key, err := p.parseKeyCurrent()
		if err != nil {
			return orderedObject{}, false
		}
		if err := validateCanonicalString(key); err != nil {
			return orderedObject{}, false
		}
		p.skipWhitespace()
		value, err := p.parseCanonicalValueCurrent()
		if err != nil {
			return orderedObject{}, false
		}
		if objectHasKey(object, key) {
			return orderedObject{}, false
		}
		object.Members = append(object.Members, objectMember{Key: key, Value: value})
	}
}

func (p *parser) parseCanonicalValue() (any, error) {
	p.skipSpace()
	return p.parseCanonicalValueCurrent()
}

func (p *parser) parseCanonicalValueCurrent() (any, error) {
	if p.pos == len(p.src) {
		return nil, p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{':
		if err := p.enterContainer(); err != nil {
			return nil, err
		}
		defer p.leaveContainer()
		p.pos++
		object := orderedObject{}
		for {
			p.skipWhitespace()
			if p.pos == len(p.src) {
				return nil, p.errorf("expected }")
			}
			if p.src[p.pos] == '}' {
				p.pos++
				return object, nil
			}
			key, err := p.parseKeyCurrent()
			if err != nil {
				return nil, err
			}
			if err := validateCanonicalString(key); err != nil {
				return nil, err
			}
			if objectHasKey(object, key) {
				return nil, p.errorf("duplicate object key")
			}
			p.skipWhitespace()
			value, err := p.parseCanonicalValueCurrent()
			if err != nil {
				return nil, err
			}
			object.Members = append(object.Members, objectMember{Key: key, Value: value})
			p.skipSeparators()
		}
	case '[':
		if err := p.enterContainer(); err != nil {
			return nil, err
		}
		defer p.leaveContainer()
		p.pos++
		array := make([]any, 0, 4)
		for {
			p.skipWhitespace()
			if p.pos == len(p.src) {
				return nil, p.errorf("expected ]")
			}
			if p.src[p.pos] == ']' {
				p.pos++
				return array, nil
			}
			value, err := p.parseCanonicalValueCurrent()
			if err != nil {
				return nil, err
			}
			array = append(array, value)
			p.skipSeparators()
		}
	case ',', '\'', '"':
		value, err := p.parseValueCurrent()
		if err != nil {
			return nil, err
		}
		if text, ok := value.(string); ok {
			return text, validateCanonicalString(text)
		}
		return value, nil
	}
	start, end, err := p.parseTokenSpan()
	if err != nil {
		return nil, err
	}
	token := p.src[start:end]
	if bytes.Equal(token, []byte("true")) {
		return true, nil
	}
	if bytes.Equal(token, []byte("false")) {
		return false, nil
	}
	if bytes.Equal(token, []byte("null")) {
		return nil, nil
	}
	if looksLikeNumberBytes(token) {
		value, err := strconv.ParseFloat(string(token), 64)
		if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
			return nil, p.errorf("non-finite JSON number")
		}
		return value, nil
	}
	value, err := p.decodeStringSpan(start, end)
	if err != nil {
		return nil, err
	}
	return value, validateCanonicalString(value)
}

func objectHasKey(object orderedObject, key string) bool {
	for _, member := range object.Members {
		if member.Key == key {
			return true
		}
	}
	return false
}

func canonicalizeMarshaledValue(value any) (any, error) {
	switch value := value.(type) {
	case nil, bool:
		return value, nil
	case string:
		return value, validateCanonicalString(value)
	case ronNumber:
		parsed, err := strconv.ParseFloat(string(value), 64)
		if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
			return nil, newError("non-finite JSON number")
		}
		return parsed, nil
	case json.Number:
		parsed, err := strconv.ParseFloat(value.String(), 64)
		if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
			return nil, newError("non-finite JSON number")
		}
		return parsed, nil
	case int64:
		return float64(value), nil
	case uint64:
		return float64(value), nil
	case float64:
		if math.IsInf(value, 0) || math.IsNaN(value) {
			return nil, newError("non-finite JSON number")
		}
		return value, nil
	case []any:
		result := make([]any, len(value))
		for i, child := range value {
			var err error
			result[i], err = canonicalizeMarshaledValue(child)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	case multilineArray:
		return canonicalizeMarshaledValue([]any(value))
	case map[string]any:
		object := orderedObject{Members: make([]objectMember, 0, len(value))}
		for key, child := range value {
			if err := validateCanonicalString(key); err != nil {
				return nil, err
			}
			normalized, err := canonicalizeMarshaledValue(child)
			if err != nil {
				return nil, err
			}
			object.Members = append(object.Members, objectMember{Key: key, Value: normalized})
		}
		return object, nil
	case orderedObject:
		object := orderedObject{Members: make([]objectMember, 0, len(value.Members))}
		for _, member := range value.Members {
			if err := validateCanonicalString(member.Key); err != nil {
				return nil, err
			}
			if objectHasKey(object, member.Key) {
				return nil, newError("duplicate object key")
			}
			normalized, err := canonicalizeMarshaledValue(member.Value)
			if err != nil {
				return nil, err
			}
			object.Members = append(object.Members, objectMember{Key: member.Key, Value: normalized})
		}
		return object, nil
	default:
		return nil, newError("unsupported canonical value")
	}
}

func validateCanonicalString(value string) error {
	if !utf8.ValidString(value) {
		return newError("invalid Unicode")
	}
	for _, r := range value {
		if r >= 0xd800 && r <= 0xdfff {
			return newError("invalid Unicode")
		}
		if (r >= 0xfdd0 && r <= 0xfdef) || r&0xffff == 0xfffe || r&0xffff == 0xffff {
			return newError("Unicode noncharacter")
		}
	}
	return nil
}
