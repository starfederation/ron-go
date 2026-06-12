package ron

import (
	"encoding/json"
	"strconv"
)

func renderCompact(value any) []byte {
	return renderCompactCap(value, 0)
}

func renderCompactCap(value any, capacity int) []byte {
	buf := make(jsonBytes, 0, capacity)
	if object, ok := value.(map[string]any); ok {
		writeCompactObject(&buf, object, true)
	} else {
		writeCompactValue(&buf, value)
	}
	return buf
}

func writeCompactValue(buf byteWriter, value any) {
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
		writeCompactArray(buf, value)
	case map[string]any:
		writeCompactObject(buf, value, false)
	default:
		panic("unsupported value")
	}
}

func writeCompactObject(buf byteWriter, object map[string]any, top bool) {
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

	if !top {
		writeByte(buf, '{')
	}
	for i, key := range keys {
		if i > 0 {
			writeByte(buf, ' ')
		}
		writeString(buf, renderString(key, true))
		if compactNeedsKeyValueSpace(object[key]) {
			writeByte(buf, ' ')
		}
		writeCompactValue(buf, object[key])
	}
	if !top {
		writeByte(buf, '}')
	}
}

func writeCompactArray(buf byteWriter, array []any) {
	writeByte(buf, '[')
	for i, value := range array {
		if i > 0 {
			writeByte(buf, ' ')
		}
		writeCompactValue(buf, value)
	}
	writeByte(buf, ']')
}

func compactNeedsKeyValueSpace(value any) bool {
	if value == nil {
		return true
	}
	switch value := value.(type) {
	case bool, ronNumber, json.Number, int64, uint64, float64:
		return true
	case string:
		rendered := renderString(value, false)
		return rendered == "" || (rendered[0] != '\'' && rendered[0] != '"')
	case []any, map[string]any:
		return false
	default:
		return true
	}
}
