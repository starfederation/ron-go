package ron

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func render(value any, indent string) []byte {
	return renderCap(value, indent, 0)
}

func renderCap(value any, indent string, capacity int) []byte {
	buf := make(jsonBytes, 0, capacity)
	writeValue(&buf, value, indent, 0)
	writeByte(&buf, '\n')
	return buf
}

type byteWriter interface {
	Write([]byte) (int, error)
	WriteByte(byte) error
	WriteString(string) (int, error)
}

func writeBytes(buf byteWriter, value []byte) {
	if _, err := buf.Write(value); err != nil {
		panic(err)
	}
}

func writeByte(buf byteWriter, value byte) {
	if err := buf.WriteByte(value); err != nil {
		panic(err)
	}
}

func writeString(buf byteWriter, value string) {
	if _, err := buf.WriteString(value); err != nil {
		panic(err)
	}
}

func writeValue(buf byteWriter, value any, indent string, depth int) {
	switch value := value.(type) {
	case nil:
		writeString(buf, "null")
	case bool:
		if value {
			writeString(buf, "true")
			return
		}
		writeString(buf, "false")
	case string:
		writeString(buf, renderString(value, false))
	case ronNumber:
		writeString(buf, string(value))
	case json.Number:
		writeString(buf, value.String())
	case int64:
		writeString(buf, strconv.FormatInt(value, 10))
	case uint64:
		writeString(buf, strconv.FormatUint(value, 10))
	case float64:
		writeString(buf, strconv.FormatFloat(value, 'g', -1, 64))
	case []any:
		writeArray(buf, value, indent, depth)
	case map[string]any:
		writeObject(buf, value, indent, depth)
	default:
		panic("unsupported value")
	}
}

func writeObject(buf byteWriter, object map[string]any, indent string, depth int) {
	if len(object) == 0 {
		writeString(buf, "{}")
		return
	}
	var keyStorage [8]string
	keys := keyStorage[:0]
	if len(object) > len(keyStorage) {
		keys = make([]string, 0, len(object))
	}
	keys = appendSortedObjectKeys(keys, object)
	if shouldInlineObject(object, keys) {
		writeByte(buf, '{')
		for _, key := range keys {
			writeString(buf, renderString(key, true))
			writeByte(buf, ' ')
			writeValue(buf, object[key], indent, depth)
		}
		writeByte(buf, '}')
		return
	}
	writeString(buf, "{\n")
	for i, key := range keys {
		writeIndent(buf, indent, depth+1)
		writeString(buf, renderString(key, true))
		writeByte(buf, ' ')
		writeValue(buf, object[key], indent, depth+1)
		if i+1 < len(keys) {
			writeByte(buf, '\n')
		}
	}
	writeByte(buf, '\n')
	writeIndent(buf, indent, depth)
	writeByte(buf, '}')
}

func appendSortedObjectKeys(keys []string, object map[string]any) []string {
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeArray(buf byteWriter, array []any, indent string, depth int) {
	if len(array) == 0 {
		writeString(buf, "[]")
		return
	}
	if shouldInlineArray(array) {
		writeByte(buf, '[')
		for i, value := range array {
			if i > 0 {
				writeByte(buf, ' ')
			}
			writeValue(buf, value, indent, depth)
		}
		writeByte(buf, ']')
		return
	}
	writeString(buf, "[\n")
	for i, value := range array {
		writeIndent(buf, indent, depth+1)
		writeValue(buf, value, indent, depth+1)
		if i+1 < len(array) {
			writeByte(buf, '\n')
		}
	}
	writeByte(buf, '\n')
	writeIndent(buf, indent, depth)
	writeByte(buf, ']')
}

func shouldInlineObject(object map[string]any, keys []string) bool {
	if len(keys) != 1 {
		return false
	}

	size := 2
	for _, key := range keys {
		value := object[key]
		switch value := value.(type) {
		case nil, bool, string, ronNumber, json.Number, int64, uint64, float64:
		case []any:
			if !shouldInlineArray(value) {
				return false
			}
		case map[string]any:
			var childKeyStorage [8]string
			childKeys := childKeyStorage[:0]
			if len(value) > len(childKeyStorage) {
				childKeys = make([]string, 0, len(value))
			}
			childKeys = appendSortedObjectKeys(childKeys, value)
			if !shouldInlineObject(value, childKeys) {
				return false
			}
		default:
			return false
		}

		size += len(renderString(key, true)) + 1 + len(renderScalar(value))
	}
	return size <= 80
}

func shouldInlineArray(array []any) bool {
	size := 2
	for i, value := range array {
		switch value := value.(type) {
		case nil, bool, string, ronNumber, json.Number, int64, uint64, float64:
		case []any:
			if !shouldInlineArray(value) {
				return false
			}
		case map[string]any:
			var keyStorage [8]string
			keys := keyStorage[:0]
			if len(value) > len(keyStorage) {
				keys = make([]string, 0, len(value))
			}
			keys = appendSortedObjectKeys(keys, value)
			if !shouldInlineObject(value, keys) {
				return false
			}
		default:
			return false
		}
		if i > 0 {
			size++
		}
		size += len(renderScalar(value))
	}
	return size <= 80
}

func renderScalar(value any) string {
	switch value := value.(type) {
	case nil:
		return "null"
	case bool:
		if value {
			return "true"
		}
		return "false"
	case string:
		return renderString(value, false)
	case ronNumber:
		return string(value)
	case json.Number:
		return value.String()
	case int64:
		return strconv.FormatInt(value, 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float64:
		return strconv.FormatFloat(value, 'g', -1, 64)
	case []any:
		var buf bytes.Buffer
		writeArray(&buf, value, "", 0)
		return buf.String()
	case map[string]any:
		var buf bytes.Buffer
		var keyStorage [8]string
		keys := keyStorage[:0]
		if len(value) > len(keyStorage) {
			keys = make([]string, 0, len(value))
		}
		keys = appendSortedObjectKeys(keys, value)
		if shouldInlineObject(value, keys) {
			writeByte(&buf, '{')
			for _, key := range keys {
				writeString(&buf, renderString(key, true))
				writeByte(&buf, ' ')
				writeString(&buf, renderScalar(value[key]))
			}
			writeByte(&buf, '}')
			return buf.String()
		}

		writeObject(&buf, value, "", 0)
		return buf.String()
	default:
		panic("unsupported value")
	}
}

func renderString(value string, key bool) string {
	if (key && canBareKey(value)) || (!key && canBareValue(value)) {
		return value
	}
	return quoteString(value)
}

func canBareKey(value string) bool {
	return value != "" && !hasStructuralRune(value)
}

func canBareValue(value string) bool {
	return value != "" && !hasStructuralRune(value) && value != "true" && value != "false" && value != "null" && !looksLikeNumber(value)
}

func hasStructuralRune(value string) bool {
	for i := 0; i < len(value); {
		b := value[i]
		if b < utf8.RuneSelf {
			switch b {
			case '{', '}', '[', ']', '"', '\'', ',', ' ', '\t', '\n', '\r':
				return true
			}
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(value[i:])
		if unicode.IsSpace(r) {
			return true
		}
		i += size
	}
	return false
}

func quoteString(value string) string {
	quote := strings.Repeat("'", longestRun(value, '\'')+1)
	return quote + value + quote
}

func longestRun(value string, quote rune) int {
	run := 0
	best := 0
	for _, r := range value {
		if r == quote {
			run++
			if run > best {
				best = run
			}
			continue
		}
		run = 0
	}
	return best
}

func writeIndent(buf byteWriter, indent string, depth int) {
	for i := 0; i < depth; i++ {
		writeString(buf, indent)
	}
}
