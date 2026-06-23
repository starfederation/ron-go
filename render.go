package ron

import (
	"bytes"
	"encoding/json"
	"strconv"
	"unicode"
	"unicode/utf8"
)

func writeValueWithCustom(buf *bytes.Buffer, value any, indent string, depth int, canonical bool, renderers []CustomRenderFunc) {
	if member, ok := typedTaggedMemberWithCustom(value, renderers); ok {
		writeTaggedObjectWithCustom(buf, member, indent, depth, canonical, renderers)
		return
	}

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
		writeArrayWithCustom(buf, value, indent, depth, canonical, renderers)
	case multilineArray:
		writeMultilineArrayWithCustom(buf, []any(value), indent, depth, canonical, renderers)
	case map[string]any, orderedObject:
		writeObjectWithCustom(buf, value, indent, depth, canonical, renderers)
	default:
		panic("unsupported value")
	}
}

func writeObjectWithCustom(buf *bytes.Buffer, object any, indent string, depth int, canonical bool, renderers []CustomRenderFunc) {
	members := objectMembers(object, canonical)
	if len(members) == 0 {
		buf.WriteString("{}")
		return
	}
	if len(members) == 1 && len(members[0].Key) > 0 && members[0].Key[0] == '#' {
		writeTaggedObjectWithCustom(buf, members[0], indent, depth, canonical, renderers)
		return
	}
	if shouldInlineObjectWithCustom(members, canonical, renderers) {
		buf.WriteByte('{')
		for _, member := range members {
			buf.WriteString(renderString(member.Key, true))
			buf.WriteByte(' ')
			writeValueWithCustom(buf, member.Value, indent, depth, canonical, renderers)
		}
		buf.WriteByte('}')
		return
	}
	buf.WriteString("{\n")
	writeObjectMembersWithCustom(buf, members, indent, depth, canonical, renderers)
	buf.WriteByte('\n')
	writeIndent(buf, indent, depth)
	buf.WriteByte('}')
}

func writeTaggedObjectWithCustom(buf *bytes.Buffer, member objectMember, indent string, depth int, canonical bool, renderers []CustomRenderFunc) {
	buf.WriteByte('{')
	buf.WriteString(renderString(member.Key, true))
	buf.WriteByte(' ')
	if inline, ok := renderInlineValueWithCustom(member.Value, canonical, renderers); ok && len(inline)+len(member.Key)+3 <= 80 {
		buf.WriteString(inline)
	} else {
		writeValueWithCustom(buf, member.Value, indent, depth, canonical, renderers)
	}
	buf.WriteByte('}')
}

func writeObjectMembersWithCustom(buf *bytes.Buffer, members []objectMember, indent string, depth int, canonical bool, renderers []CustomRenderFunc) {
	for i, member := range members {
		writeIndent(buf, indent, depth+1)
		buf.WriteString(renderString(member.Key, true))
		buf.WriteByte(' ')
		writeValueWithCustom(buf, member.Value, indent, depth+1, canonical, renderers)
		if i+1 < len(members) {
			buf.WriteByte('\n')
		}
	}
}

type multilineArray []any

func writeArrayWithCustom(buf *bytes.Buffer, array []any, indent string, depth int, canonical bool, renderers []CustomRenderFunc) {
	if len(array) == 0 {
		buf.WriteString("[]")
		return
	}
	if shouldInlineArrayWithCustom(array, canonical, renderers) {
		buf.WriteByte('[')
		for i, value := range array {
			if i > 0 {
				buf.WriteByte(' ')
			}
			writeValueWithCustom(buf, value, indent, depth, canonical, renderers)
		}
		buf.WriteByte(']')
		return
	}
	buf.WriteString("[\n")
	for i, value := range array {
		writeIndent(buf, indent, depth+1)
		writeValueWithCustom(buf, value, indent, depth+1, canonical, renderers)
		if i+1 < len(array) {
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')
	writeIndent(buf, indent, depth)
	buf.WriteByte(']')
}

func writeMultilineArrayWithCustom(buf *bytes.Buffer, array []any, indent string, depth int, canonical bool, renderers []CustomRenderFunc) {
	if len(array) == 0 {
		buf.WriteString("[]")
		return
	}
	buf.WriteString("[\n")
	for i, value := range array {
		writeIndent(buf, indent, depth+1)
		writeValueWithCustom(buf, value, indent, depth+1, canonical, renderers)
		if i+1 < len(array) {
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')
	writeIndent(buf, indent, depth)
	buf.WriteByte(']')
}

func shouldInlineObjectWithCustom(members []objectMember, canonical bool, renderers []CustomRenderFunc) bool {
	if len(members) != 1 {
		return false
	}

	size := 2
	for _, member := range members {
		if !canInlineValueWithCustom(member.Value, canonical, renderers) {
			return false
		}
		size += len(renderString(member.Key, true)) + 1 + len(renderScalarWithCustom(member.Value, canonical, renderers))
	}
	return size <= 80
}

func renderTypedTaggedValueWithCustom(value any, canonical bool, renderers []CustomRenderFunc) (string, bool) {
	member, ok := typedTaggedMemberWithCustom(value, renderers)
	if !ok {
		return "", false
	}
	payload, ok := renderInlineValueWithCustom(member.Value, canonical, renderers)
	if !ok {
		return "", false
	}
	return "{" + renderString(member.Key, true) + " " + payload + "}", true
}

func shouldInlineArrayWithCustom(array []any, canonical bool, renderers []CustomRenderFunc) bool {
	size := 2
	for i, value := range array {
		if !canInlineValueWithCustom(value, canonical, renderers) {
			return false
		}
		if i > 0 {
			size++
		}
		size += len(renderScalarWithCustom(value, canonical, renderers))
	}
	return size <= 80
}

func canInlineValueWithCustom(value any, canonical bool, renderers []CustomRenderFunc) bool {
	if _, ok := typedTaggedMemberWithCustom(value, renderers); ok {
		return true
	}

	switch value := value.(type) {
	case nil, bool, string, ronNumber, json.Number, int64, uint64, float64:
		return true
	case []any:
		return shouldInlineArrayWithCustom(value, canonical, renderers)
	case map[string]any, orderedObject:
		return shouldInlineObjectWithCustom(objectMembers(value, canonical), canonical, renderers)
	default:
		return false
	}
}

func renderInlineValueWithCustom(value any, canonical bool, renderers []CustomRenderFunc) (string, bool) {
	if rendered, ok := renderTypedTaggedValueWithCustom(value, canonical, renderers); ok {
		return rendered, true
	}

	switch value := value.(type) {
	case nil, bool, string, ronNumber, json.Number, int64, uint64, float64:
		return renderScalarWithCustom(value, canonical, renderers), true
	case []any:
		parts := make([]string, len(value))
		size := 2
		for i, child := range value {
			part, ok := renderInlineValueWithCustom(child, canonical, renderers)
			if !ok {
				return "", false
			}
			parts[i] = part
			size += len(part)
			if i > 0 {
				size++
			}
			if size > 80 {
				return "", false
			}
		}
		var buf bytes.Buffer
		buf.Grow(size)
		buf.WriteByte('[')
		for i, part := range parts {
			if i > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(part)
		}
		buf.WriteByte(']')
		return buf.String(), true
	case map[string]any, orderedObject:
		members := objectMembers(value, canonical)
		parts := make([]string, len(members))
		size := 2
		for i, member := range members {
			part, ok := renderInlineValueWithCustom(member.Value, canonical, renderers)
			if !ok {
				return "", false
			}
			key := renderString(member.Key, true)
			parts[i] = key + " " + part
			size += len(parts[i])
			if i > 0 {
				size++
			}
			if size > 80 {
				return "", false
			}
		}
		var buf bytes.Buffer
		buf.Grow(size)
		buf.WriteByte('{')
		for i, part := range parts {
			if i > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(part)
		}
		buf.WriteByte('}')
		return buf.String(), true
	default:
		return "", false
	}
}

func renderScalar(value any, canonical bool) string {
	return renderScalarWithCustom(value, canonical, nil)
}

func renderScalarWithCustom(value any, canonical bool, renderers []CustomRenderFunc) string {
	if rendered, ok := renderTypedTaggedValueWithCustom(value, canonical, renderers); ok {
		return rendered
	}

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
		writeArrayWithCustom(&buf, value, "", 0, canonical, renderers)
		return buf.String()
	case multilineArray:
		var buf bytes.Buffer
		writeMultilineArrayWithCustom(&buf, []any(value), "", 0, canonical, renderers)
		return buf.String()
	case map[string]any, orderedObject:
		var buf bytes.Buffer
		writeObjectWithCustom(&buf, value, "", 0, canonical, renderers)
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

func writeCompactValueWithCustom(buf *bytes.Buffer, value any, top, canonical bool, renderers []CustomRenderFunc) {
	if member, ok := typedTaggedMemberWithCustom(value, renderers); ok {
		buf.WriteByte('{')
		buf.WriteString(renderString(member.Key, true))
		if inline, ok := renderInlineValueWithCustom(member.Value, canonical, renderers); ok {
			buf.WriteByte(' ')
			buf.WriteString(inline)
		} else {
			writeCompactValueWithCustom(buf, member.Value, false, canonical, renderers)
		}
		buf.WriteByte('}')
		return
	}

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
	case multilineArray:
		buf.WriteByte('[')
		for i, value := range value {
			if i > 0 {
				buf.WriteByte(' ')
			}
			writeCompactValueWithCustom(buf, value, false, canonical, renderers)
		}
		buf.WriteByte(']')
	case []any:
		buf.WriteByte('[')
		for i, value := range value {
			if i > 0 {
				buf.WriteByte(' ')
			}
			writeCompactValueWithCustom(buf, value, false, canonical, renderers)
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
			if _, ok := typedTaggedMemberWithCustom(member.Value, renderers); ok {
				needsSpace = false
			} else if member.Value != nil {
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
			writeCompactValueWithCustom(buf, member.Value, false, canonical, renderers)
		}
		if !top {
			buf.WriteByte('}')
		}
	default:
		panic("unsupported value")
	}
}
