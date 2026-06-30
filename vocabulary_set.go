package ron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
)

const (
	// VocabularySetV1 is the RON set typed vocabulary URI.
	VocabularySetV1 = "https://ron.dev/vocab/set/v1"
)

// Set is a set vocabulary #set value.
type Set []any

// Uint32Range is an inclusive range in a #bits value.
type Uint32Range struct {
	First uint32
	Last  uint32
}

// Uint32BitSet is a set vocabulary #bits value.
type Uint32BitSet []Uint32Range

func (opts optionState) isSetTag(tag string) bool {
	if !opts.vocabularyEnabled(vocabularySet, VocabularySetV1) {
		return false
	}
	switch tag {
	case "#set", "#bits":
		return true
	default:
		return false
	}
}

func (opts optionState) parseSetPayload(tag string, payload any) (any, error) {
	switch tag {
	case "#set":
		values, ok := payload.([]any)
		if !ok {
			return nil, newError("invalid #set payload")
		}
		return canonicalSet(values), nil
	case "#bits":
		values, ok := payload.([]any)
		if !ok {
			return nil, newError("invalid #bits payload")
		}
		return parseBitSet(values)
	default:
		return nil, newError("unsupported set tag")
	}
}

func canonicalSetPayload(value Set) []any {
	set := canonicalSet([]any(value))
	return []any(set)
}

func canonicalSet(values []any) Set {
	entries := make([]setEntry, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := canonicalSetKey(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, setEntry{
			Key:   key,
			Value: canonicalSetValue(value),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	set := make(Set, len(entries))
	for i, entry := range entries {
		set[i] = entry.Value
	}
	return set
}

type setEntry struct {
	Key   string
	Value any
}

func canonicalSetValue(value any) any {
	switch value := value.(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case uint:
		return uint64(value)
	case uint8:
		return uint64(value)
	case uint16:
		return uint64(value)
	case uint32:
		return uint64(value)
	case float32:
		return float64(value)
	case []any:
		canonical := make([]any, len(value))
		for i, child := range value {
			canonical[i] = canonicalSetValue(child)
		}
		return canonical
	case map[string]any:
		var object orderedObject
		for _, member := range objectMembers(value, true) {
			object.Set(member.Key, canonicalSetValue(member.Value))
		}
		return object
	case orderedObject:
		var object orderedObject
		for _, member := range objectMembers(value, true) {
			object.Set(member.Key, canonicalSetValue(member.Value))
		}
		return object
	default:
		return value
	}
}

func canonicalSetKey(value any) string {
	var buf bytes.Buffer
	writeSetCanonicalJSON(&buf, value)
	return buf.String()
}

func writeSetCanonicalJSON(buf *bytes.Buffer, value any) {
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
		writeJSONQuoted(buf, value)
	case ronNumber:
		writeSetCanonicalNumber(buf, string(value))
	case json.Number:
		writeSetCanonicalNumber(buf, value.String())
	case int:
		buf.WriteString(strconv.FormatInt(int64(value), 10))
	case int8:
		buf.WriteString(strconv.FormatInt(int64(value), 10))
	case int16:
		buf.WriteString(strconv.FormatInt(int64(value), 10))
	case int32:
		buf.WriteString(strconv.FormatInt(int64(value), 10))
	case int64:
		buf.WriteString(strconv.FormatInt(value, 10))
	case uint:
		buf.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint8:
		buf.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint16:
		buf.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint32:
		buf.WriteString(strconv.FormatUint(uint64(value), 10))
	case uint64:
		buf.WriteString(strconv.FormatUint(value, 10))
	case float32:
		body, err := appendRFC8785Number(nil, float64(value))
		if err != nil {
			panic(err)
		}
		buf.Write(body)
	case float64:
		body, err := appendRFC8785Number(nil, value)
		if err != nil {
			panic(err)
		}
		buf.Write(body)
	case []any:
		buf.WriteByte('[')
		for i, child := range value {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeSetCanonicalJSON(buf, child)
		}
		buf.WriteByte(']')
	case map[string]any:
		writeSetCanonicalJSONObject(buf, objectMembers(value, true))
	case orderedObject:
		writeSetCanonicalJSONObject(buf, objectMembers(value, true))
	default:
		panic(fmt.Sprintf("unsupported #set value %T", value))
	}
}

func writeSetCanonicalJSONObject(buf *bytes.Buffer, members []objectMember) {
	buf.WriteByte('{')
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeJSONQuoted(buf, member.Key)
		buf.WriteByte(':')
		writeSetCanonicalJSON(buf, member.Value)
	}
	buf.WriteByte('}')
}

func writeSetCanonicalNumber(buf *bytes.Buffer, text string) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		buf.WriteString(text)
		return
	}
	body, err := appendRFC8785Number(nil, value)
	if err != nil {
		panic(err)
	}
	buf.Write(body)
}

func parseBitSet(values []any) (Uint32BitSet, error) {
	ranges := make(Uint32BitSet, 0, len(values))
	for _, value := range values {
		if endpoints, ok := value.([]any); ok {
			if len(endpoints) != 2 {
				return nil, newError("invalid #bits range")
			}
			first, err := parseBitSetEndpoint(endpoints[0])
			if err != nil {
				return nil, err
			}
			last, err := parseBitSetEndpoint(endpoints[1])
			if err != nil {
				return nil, err
			}
			if first > last {
				return nil, newError("invalid #bits range")
			}
			ranges = append(ranges, Uint32Range{First: first, Last: last})
			continue
		}

		index, err := parseBitSetEndpoint(value)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, Uint32Range{First: index, Last: index})
	}
	return canonicalBitSet(ranges), nil
}

func parseBitSetEndpoint(value any) (uint32, error) {
	switch value := value.(type) {
	case ronNumber:
		return parseUint32Text(string(value))
	case json.Number:
		return parseUint32Text(value.String())
	case float64:
		if value < 0 || value > math.MaxUint32 || math.Trunc(value) != value {
			return 0, newError("invalid #bits value")
		}
		return uint32(value), nil
	case int64:
		if value < 0 || value > math.MaxUint32 {
			return 0, newError("invalid #bits value")
		}
		return uint32(value), nil
	case uint64:
		if value > math.MaxUint32 {
			return 0, newError("invalid #bits value")
		}
		return uint32(value), nil
	default:
		return 0, newError("invalid #bits value")
	}
}

func parseUint32Text(text string) (uint32, error) {
	value, err := strconv.ParseUint(text, 10, 32)
	if err != nil || strconv.FormatUint(value, 10) != text {
		return 0, newError("invalid #bits value")
	}
	return uint32(value), nil
}

func canonicalBitSetPayload(value Uint32BitSet) []any {
	ranges := canonicalBitSet(value)
	payload := make([]any, len(ranges))
	for i, bitRange := range ranges {
		if bitRange.First == bitRange.Last {
			payload[i] = uint64(bitRange.First)
		} else {
			payload[i] = []any{
				uint64(bitRange.First),
				uint64(bitRange.Last),
			}
		}
	}
	return payload
}

func canonicalBitSet(value Uint32BitSet) Uint32BitSet {
	if len(value) == 0 {
		return nil
	}
	ranges := append(Uint32BitSet(nil), value...)
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].First == ranges[j].First {
			return ranges[i].Last < ranges[j].Last
		}
		return ranges[i].First < ranges[j].First
	})

	canonical := make(Uint32BitSet, 0, len(ranges))
	for _, bitRange := range ranges {
		if bitRange.Last < bitRange.First {
			bitRange.First, bitRange.Last = bitRange.Last, bitRange.First
		}
		if len(canonical) == 0 {
			canonical = append(canonical, bitRange)
			continue
		}

		last := &canonical[len(canonical)-1]
		if uint64(bitRange.First) <= uint64(last.Last)+1 {
			if bitRange.Last > last.Last {
				last.Last = bitRange.Last
			}
			continue
		}
		canonical = append(canonical, bitRange)
	}
	return canonical
}
