package ron

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

func canonicalJSON(src []byte) ([]byte, error) {
	for i := 0; i < len(src); i++ {
		if src[i] != '"' {
			continue
		}
		i++
		for i < len(src) && src[i] != '"' {
			if src[i] != '\\' {
				i++
				continue
			}
			i++
			if i == len(src) {
				break
			}
			if src[i] != 'u' {
				i++
				continue
			}

			code, ok := parseHex4(src[i+1:])
			if !ok {
				break
			}
			switch {
			case code >= 0xd800 && code <= 0xdbff:
				if i+11 >= len(src) || src[i+5] != '\\' || src[i+6] != 'u' {
					return nil, newError("invalid lone surrogate")
				}
				low, ok := parseHex4(src[i+7:])
				if !ok || low < 0xdc00 || low > 0xdfff {
					return nil, newError("invalid lone surrogate")
				}
				i += 11
			case code >= 0xdc00 && code <= 0xdfff:
				return nil, newError("invalid lone surrogate")
			default:
				i += 5
			}
		}
	}

	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()

	value, err := decodeRFC8785Value(dec)
	if err != nil {
		return nil, err
	}

	if _, err := dec.Token(); err == nil {
		return nil, newError("unexpected trailing JSON")
	} else if err != io.EOF {
		return nil, err
	}

	var buf bytes.Buffer
	if err := writeRFC8785Value(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeRFC8785Value(dec *json.Decoder) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}

	switch token := token.(type) {
	case nil, bool, string:
		return token, nil
	case json.Number:
		text := token.String()
		float, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsInf(float, 0) || math.IsNaN(float) {
			return nil, newError("invalid JSON number")
		}

		isInteger := true
		for i := 0; i < len(text); i++ {
			switch text[i] {
			case '.', 'e', 'E':
				isInteger = false
			}
		}
		if isInteger {
			integer := new(big.Int)
			if _, ok := integer.SetString(text, 10); !ok {
				return nil, newError("JSON integer is not exactly representable as float64")
			}

			rational := new(big.Rat)
			rational.SetFloat64(float)
			if !rational.IsInt() || rational.Num().Cmp(integer) != 0 {
				return nil, newError("JSON integer is not exactly representable as float64")
			}
		}
		return float, nil
	case json.Delim:
		switch token {
		case '{':
			var object orderedObject
			seen := make(map[string]struct{})
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, newError("expected JSON object key")
				}
				if _, ok := seen[key]; ok {
					return nil, newError("duplicate JSON object key")
				}
				seen[key] = struct{}{}

				value, err := decodeRFC8785Value(dec)
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
				value, err := decodeRFC8785Value(dec)
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

func writeRFC8785Value(buf *bytes.Buffer, value any) error {
	switch value := value.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if value {
			buf.WriteString("true")
			return nil
		}
		buf.WriteString("false")
	case string:
		writeJSONQuoted(buf, value)
	case float64:
		body, err := appendRFC8785Number(nil, value)
		if err != nil {
			return err
		}
		buf.Write(body)
	case []any:
		buf.WriteByte('[')
		for i, value := range value {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeRFC8785Value(buf, value); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case orderedObject:
		members := objectMembers(value, true)
		buf.WriteByte('{')
		for i, member := range members {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONQuoted(buf, member.Key)
			buf.WriteByte(':')
			if err := writeRFC8785Value(buf, member.Value); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		panic("unsupported value")
	}
	return nil
}

func appendRFC8785Number(dst []byte, value float64) ([]byte, error) {
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return nil, newError("non-finite JSON number")
	}
	if value == 0 {
		return append(dst, '0'), nil
	}

	if value < 0 {
		dst = append(dst, '-')
		value = -value
	}

	text := strconv.FormatFloat(value, 'g', -1, 64)
	exponent := 0
	if idx := bytes.IndexAny([]byte(text), "eE"); idx >= 0 {
		parsed, _ := strconv.Atoi(text[idx+1:])
		exponent = parsed
		text = text[:idx]
	}

	integerDigits := len(text)
	digits := text
	if idx := bytes.IndexByte([]byte(text), '.'); idx >= 0 {
		integerDigits = idx
		digits = text[:idx] + text[idx+1:]
	}

	trimmed := 0
	for trimmed < len(digits) && digits[trimmed] == '0' {
		trimmed++
	}
	if trimmed == len(digits) {
		digits = "0"
		integerDigits = 1
		trimmed = 0
	}
	digits = digits[trimmed:]
	decimalExp := integerDigits + exponent - trimmed

	switch {
	case decimalExp > 0 && decimalExp <= 21:
		if len(digits) <= decimalExp {
			dst = append(dst, digits...)
			return appendRepeatedByte(dst, '0', decimalExp-len(digits)), nil
		}
		dst = append(dst, digits[:decimalExp]...)
		dst = append(dst, '.')
		dst = append(dst, digits[decimalExp:]...)
		return dst, nil
	case decimalExp > -6 && decimalExp <= 0:
		dst = append(dst, '0', '.')
		dst = appendRepeatedByte(dst, '0', -decimalExp)
		dst = append(dst, digits...)
		return dst, nil
	default:
		dst = append(dst, digits[0])
		if len(digits) > 1 {
			dst = append(dst, '.')
			dst = append(dst, digits[1:]...)
		}
		dst = append(dst, 'e')
		exponent := decimalExp - 1
		if exponent >= 0 {
			dst = append(dst, '+')
		}
		dst = strconv.AppendInt(dst, int64(exponent), 10)
		return dst, nil
	}
}

func appendRepeatedByte(dst []byte, value byte, count int) []byte {
	for i := 0; i < count; i++ {
		dst = append(dst, value)
	}
	return dst
}

func sortObjectMembers(members []objectMember) {
	sort.Slice(members, func(i, j int) bool {
		return rfc8785StringLess(members[i].Key, members[j].Key)
	})
}

func rfc8785StringLess(left, right string) bool {
	if asciiString(left) && asciiString(right) {
		return asciiStringLess(left, right)
	}

	leftIter := utf16StringIterator{text: left}
	rightIter := utf16StringIterator{text: right}
	for {
		leftCode, leftOK := leftIter.next()
		rightCode, rightOK := rightIter.next()
		if !leftOK || !rightOK {
			return !leftOK && rightOK
		}
		if leftCode != rightCode {
			return leftCode < rightCode
		}
	}
}

func asciiStringLess(left, right string) bool {
	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return len(left) < len(right)
}

func asciiString(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

type utf16StringIterator struct {
	text    string
	pos     int
	pending uint16
}

func (iter *utf16StringIterator) next() (uint16, bool) {
	if iter.pending != 0 {
		code := iter.pending
		iter.pending = 0
		return code, true
	}
	if iter.pos >= len(iter.text) {
		return 0, false
	}

	r, size := utf8.DecodeRuneInString(iter.text[iter.pos:])
	iter.pos += size
	if r <= 0xffff {
		return uint16(r), true
	}

	high, low := utf16.EncodeRune(r)
	iter.pending = uint16(low)
	return uint16(high), true
}

func parseHex4(src []byte) (uint16, bool) {
	if len(src) < 4 {
		return 0, false
	}

	var value uint16
	for i := 0; i < 4; i++ {
		var hex byte
		switch value := src[i]; {
		case value >= '0' && value <= '9':
			hex = value - '0'
		case value >= 'a' && value <= 'f':
			hex = value - 'a' + 10
		case value >= 'A' && value <= 'F':
			hex = value - 'A' + 10
		default:
			return 0, false
		}
		value = value<<4 | uint16(hex)
	}
	return value, true
}
