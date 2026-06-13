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
	writeValue(&buf, value, indent, 0, true)
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

func writeValue(buf byteWriter, value any, indent string, depth int, canonical bool) {
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
		writeArray(buf, value, indent, depth, canonical)
	case map[string]any, orderedObject:
		writeObject(buf, value, indent, depth, canonical)
	default:
		panic("unsupported value")
	}
}

func writeObject(buf byteWriter, object any, indent string, depth int, canonical bool) {
	members := objectMembers(object, canonical)
	if len(members) == 0 {
		writeString(buf, "{}")
		return
	}
	if shouldInlineObject(members, canonical) {
		writeByte(buf, '{')
		for _, member := range members {
			writeString(buf, renderString(member.Key, true))
			writeByte(buf, ' ')
			writeValue(buf, member.Value, indent, depth, canonical)
		}
		writeByte(buf, '}')
		return
	}
	writeString(buf, "{\n")
	for i, member := range members {
		writeIndent(buf, indent, depth+1)
		writeString(buf, renderString(member.Key, true))
		writeByte(buf, ' ')
		writeValue(buf, member.Value, indent, depth+1, canonical)
		if i+1 < len(members) {
			writeByte(buf, '\n')
		}
	}
	writeByte(buf, '\n')
	writeIndent(buf, indent, depth)
	writeByte(buf, '}')
}

func writeArray(buf byteWriter, array []any, indent string, depth int, canonical bool) {
	if len(array) == 0 {
		writeString(buf, "[]")
		return
	}
	if shouldInlineArray(array, canonical) {
		writeByte(buf, '[')
		for i, value := range array {
			if i > 0 {
				writeByte(buf, ' ')
			}
			writeValue(buf, value, indent, depth, canonical)
		}
		writeByte(buf, ']')
		return
	}
	writeString(buf, "[\n")
	for i, value := range array {
		writeIndent(buf, indent, depth+1)
		writeValue(buf, value, indent, depth+1, canonical)
		if i+1 < len(array) {
			writeByte(buf, '\n')
		}
	}
	writeByte(buf, '\n')
	writeIndent(buf, indent, depth)
	writeByte(buf, ']')
}

func shouldInlineObject(members []objectMember, canonical bool) bool {
	if len(members) != 1 {
		return false
	}

	size := 2
	for _, member := range members {
		if !canInlineValue(member.Value, canonical) {
			return false
		}
		size += len(renderString(member.Key, true)) + 1 + len(renderScalar(member.Value, canonical))
	}
	return size <= 80
}

func shouldInlineArray(array []any, canonical bool) bool {
	size := 2
	for i, value := range array {
		if !canInlineValue(value, canonical) {
			return false
		}
		if i > 0 {
			size++
		}
		size += len(renderScalar(value, canonical))
	}
	return size <= 80
}

func canInlineValue(value any, canonical bool) bool {
	switch value := value.(type) {
	case nil, bool, string, ronNumber, json.Number, int64, uint64, float64:
		return true
	case []any:
		return shouldInlineArray(value, canonical)
	case map[string]any, orderedObject:
		return shouldInlineObject(objectMembers(value, canonical), canonical)
	default:
		return false
	}
}

func renderScalar(value any, canonical bool) string {
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
		writeArray(&buf, value, "", 0, canonical)
		return buf.String()
	case map[string]any, orderedObject:
		var buf bytes.Buffer
		writeObject(&buf, value, "", 0, canonical)
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

type objectMember struct {
	Key   string
	Value any
}

type orderedObject struct {
	Members []objectMember
	Index   map[string]int
}

func (o *orderedObject) Set(key string, value any) {
	if o.Index != nil {
		if idx, ok := o.Index[key]; ok {
			copy(o.Members[idx:], o.Members[idx+1:])
			o.Members = o.Members[:len(o.Members)-1]
			delete(o.Index, key)
			for i := idx; i < len(o.Members); i++ {
				o.Index[o.Members[i].Key] = i
			}
		}
	} else if len(o.Members) > 0 {
		for idx, member := range o.Members {
			if member.Key == key {
				copy(o.Members[idx:], o.Members[idx+1:])
				o.Members = o.Members[:len(o.Members)-1]
				break
			}
		}
		if len(o.Members) == 8 {
			o.Index = make(map[string]int, len(o.Members)+1)
			for i, member := range o.Members {
				o.Index[member.Key] = i
			}
		}
	}

	if o.Index != nil {
		o.Index[key] = len(o.Members)
	}
	o.Members = append(o.Members, objectMember{
		Key:   key,
		Value: value,
	})
}

func objectMembers(value any, canonical bool) []objectMember {
	switch value := value.(type) {
	case map[string]any:
		members := make([]objectMember, 0, len(value))
		for key, child := range value {
			members = append(members, objectMember{
				Key:   key,
				Value: child,
			})
		}
		sortObjectMembers(members)
		return members
	case orderedObject:
		members := append([]objectMember(nil), value.Members...)
		if canonical {
			sortObjectMembers(members)
		}
		return members
	default:
		panic("unsupported object")
	}
}

func sortObjectMembers(members []objectMember) {
	sort.Slice(members, func(i, j int) bool {
		return members[i].Key < members[j].Key
	})
}
