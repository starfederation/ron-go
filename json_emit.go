package ron

import (
	"bytes"
	"encoding/json"
	"strconv"
)

type jsonBytes []byte

func (b *jsonBytes) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

func (b *jsonBytes) WriteByte(value byte) error {
	*b = append(*b, value)
	return nil
}

func (b *jsonBytes) WriteString(value string) (int, error) {
	*b = append(*b, value...)
	return len(value), nil
}

func emitJSON(value any, prefix, indent string) []byte {
	var buf bytes.Buffer
	emitJSONInto(&buf, value, prefix, indent)
	return buf.Bytes()
}

func emitJSONInto(buf byteWriter, value any, prefix, indent string) {
	writeJSONValue(buf, value, prefix, indent, 0)
}

func writeJSONValue(buf byteWriter, value any, prefix, indent string, depth int) {
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
		writeString(buf, strconv.Quote(value))
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
		writeJSONArray(buf, value, prefix, indent, depth)
	case map[string]any, orderedObject:
		writeJSONObject(buf, value, prefix, indent, depth)
	default:
		panic("unsupported value")
	}
}

func writeJSONObject(buf byteWriter, object any, prefix, indent string, depth int) {
	members := objectMembers(object, true)
	if len(members) == 0 {
		writeString(buf, "{}")
		return
	}
	writeByte(buf, '{')
	if indent == "" {
		for i, member := range members {
			if i > 0 {
				writeByte(buf, ',')
			}
			writeString(buf, strconv.Quote(member.Key))
			writeByte(buf, ':')
			writeJSONValue(buf, member.Value, prefix, indent, depth+1)
		}
		writeByte(buf, '}')
		return
	}
	writeByte(buf, '\n')
	for i, member := range members {
		writeJSONIndent(buf, prefix, indent, depth+1)
		writeString(buf, strconv.Quote(member.Key))
		writeString(buf, ": ")
		writeJSONValue(buf, member.Value, prefix, indent, depth+1)
		if i+1 < len(members) {
			writeByte(buf, ',')
		}
		writeByte(buf, '\n')
	}
	writeJSONIndent(buf, prefix, indent, depth)
	writeByte(buf, '}')
}

func writeJSONArray(buf byteWriter, array []any, prefix, indent string, depth int) {
	if len(array) == 0 {
		writeString(buf, "[]")
		return
	}
	writeByte(buf, '[')
	if indent == "" {
		for i, value := range array {
			if i > 0 {
				writeByte(buf, ',')
			}
			writeJSONValue(buf, value, prefix, indent, depth+1)
		}
		writeByte(buf, ']')
		return
	}
	writeByte(buf, '\n')
	for i, value := range array {
		writeJSONIndent(buf, prefix, indent, depth+1)
		writeJSONValue(buf, value, prefix, indent, depth+1)
		if i+1 < len(array) {
			writeByte(buf, ',')
		}
		writeByte(buf, '\n')
	}
	writeJSONIndent(buf, prefix, indent, depth)
	writeByte(buf, ']')
}

func writeJSONIndent(buf byteWriter, prefix, indent string, depth int) {
	writeString(buf, prefix)
	for i := 0; i < depth; i++ {
		writeString(buf, indent)
	}
}
