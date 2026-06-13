package ron

import (
	"bytes"
	"encoding/json"
	"strconv"
	"unicode"
	"unicode/utf8"
)

func writeValue(buf *bytes.Buffer, value any, indent string, depth int, canonical bool) {
	switch value := value.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if value {
			buf.WriteString("true")
			return
		}
		buf.WriteString("false")
	case string:
		buf.WriteString(renderString(value, false))
	case ronNumber:
		buf.WriteString(string(value))
	case json.Number:
		buf.WriteString(value.String())
	case int64:
		buf.WriteString(strconv.FormatInt(value, 10))
	case uint64:
		buf.WriteString(strconv.FormatUint(value, 10))
	case float64:
		buf.WriteString(strconv.FormatFloat(value, 'g', -1, 64))
	case []any:
		writeArray(buf, value, indent, depth, canonical)
	case map[string]any, orderedObject:
		writeObject(buf, value, indent, depth, canonical)
	default:
		panic("unsupported value")
	}
}

func writeObject(buf *bytes.Buffer, object any, indent string, depth int, canonical bool) {
	members := objectMembers(object, canonical)
	if len(members) == 0 {
		buf.WriteString("{}")
		return
	}
	if shouldInlineObject(members, canonical) {
		buf.WriteByte('{')
		for _, member := range members {
			buf.WriteString(renderString(member.Key, true))
			buf.WriteByte(' ')
			writeValue(buf, member.Value, indent, depth, canonical)
		}
		buf.WriteByte('}')
		return
	}
	buf.WriteString("{\n")
	writeObjectMembers(buf, members, indent, depth, canonical)
	buf.WriteByte('\n')
	writeIndent(buf, indent, depth)
	buf.WriteByte('}')
}

func writeObjectMembers(buf *bytes.Buffer, members []objectMember, indent string, depth int, canonical bool) {
	for i, member := range members {
		writeIndent(buf, indent, depth+1)
		buf.WriteString(renderString(member.Key, true))
		buf.WriteByte(' ')
		writeValue(buf, member.Value, indent, depth+1, canonical)
		if i+1 < len(members) {
			buf.WriteByte('\n')
		}
	}
}

func writeArray(buf *bytes.Buffer, array []any, indent string, depth int, canonical bool) {
	if len(array) == 0 {
		buf.WriteString("[]")
		return
	}
	if shouldInlineArray(array, canonical) {
		buf.WriteByte('[')
		for i, value := range array {
			if i > 0 {
				buf.WriteByte(' ')
			}
			writeValue(buf, value, indent, depth, canonical)
		}
		buf.WriteByte(']')
		return
	}
	buf.WriteString("[\n")
	for i, value := range array {
		writeIndent(buf, indent, depth+1)
		writeValue(buf, value, indent, depth+1, canonical)
		if i+1 < len(array) {
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')
	writeIndent(buf, indent, depth)
	buf.WriteByte(']')
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
	structural := value == ""
	for i := 0; !structural && i < len(value); {
		b := value[i]
		if b < utf8.RuneSelf {
			switch b {
			case '{', '}', '[', ']', '"', '\'', ',', ' ', '\t', '\n', '\r':
				structural = true
			}
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(value[i:])
		if unicode.IsSpace(r) {
			structural = true
		}
		i += size
	}
	if !structural && (key || (value != "true" && value != "false" && value != "null" && !looksLikeNumberBytes([]byte(value)))) {
		return value
	}

	run := 0
	longest := 0
	for _, r := range value {
		if r == '\'' {
			run++
			if run > longest {
				longest = run
			}
			continue
		}
		run = 0
	}

	quote := make([]byte, longest+1)
	for i := range quote {
		quote[i] = '\''
	}
	return string(quote) + value + string(quote)
}

func writeIndent(buf *bytes.Buffer, indent string, depth int) {
	for i := 0; i < depth; i++ {
		buf.WriteString(indent)
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

func writeCompactValue(buf *bytes.Buffer, value any, top, canonical bool) {
	switch value := value.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if value {
			buf.WriteString("true")
			return
		}
		buf.WriteString("false")
	case string:
		buf.WriteString(renderString(value, false))
	case ronNumber:
		buf.WriteString(string(value))
	case json.Number:
		buf.WriteString(value.String())
	case int64:
		buf.WriteString(strconv.FormatInt(value, 10))
	case uint64:
		buf.WriteString(strconv.FormatUint(value, 10))
	case float64:
		buf.WriteString(strconv.FormatFloat(value, 'g', -1, 64))
	case []any:
		buf.WriteByte('[')
		for i, value := range value {
			if i > 0 {
				buf.WriteByte(' ')
			}
			writeCompactValue(buf, value, false, canonical)
		}
		buf.WriteByte(']')
	case map[string]any, orderedObject:
		members := objectMembers(value, canonical)
		if len(members) == 0 {
			buf.WriteString("{}")
			return
		}

		if !top {
			buf.WriteByte('{')
		}
		for i, member := range members {
			if i > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(renderString(member.Key, true))
			needsSpace := true
			if member.Value != nil {
				switch value := member.Value.(type) {
				case string:
					rendered := renderString(value, false)
					needsSpace = rendered == "" || (rendered[0] != '\'' && rendered[0] != '"')
				case []any, map[string]any, orderedObject:
					needsSpace = false
				}
			}
			if needsSpace {
				buf.WriteByte(' ')
			}
			writeCompactValue(buf, member.Value, false, canonical)
		}
		if !top {
			buf.WriteByte('}')
		}
	default:
		panic("unsupported value")
	}
}
