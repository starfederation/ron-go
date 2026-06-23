package ron

import (
	"bytes"
	"encoding/json"

	rongeo "github.com/starfederation/ron-go/components/geo"
)

const (
	// VocabularyGeoV1 is the RON GeoJSON typed vocabulary URI.
	VocabularyGeoV1 = "https://ron.dev/vocab/geo/v1"
)

// GeoJSON is a geo vocabulary #geo value.
type GeoJSON = rongeo.Value

func (opts optionState) isGeoTag(tag string) bool {
	if _, ok := opts.vocabularies[VocabularyGeoV1]; !ok {
		return false
	}
	return tag == "#geo"
}

func (opts optionState) parseGeoPayload(tag string, payload any) (any, error) {
	if tag != "#geo" {
		return nil, newError("unsupported geo tag")
	}
	if !rongeo.Valid(jsonCompatibleValue(payload)) {
		return nil, newError("invalid #geo payload")
	}
	parsed, err := opts.parseVocabularyValue(payload)
	if err != nil {
		return nil, err
	}
	return GeoJSON{Data: parsed}, nil
}

func jsonCompatibleValue(value any) any {
	switch value := value.(type) {
	case orderedObject:
		object := make(map[string]any, len(value.Members))
		for _, member := range value.Members {
			object[member.Key] = jsonCompatibleValue(member.Value)
		}
		return object
	case map[string]any:
		object := make(map[string]any, len(value))
		for key, child := range value {
			object[key] = jsonCompatibleValue(child)
		}
		return object
	case []any:
		array := make([]any, len(value))
		for i, child := range value {
			array[i] = jsonCompatibleValue(child)
		}
		return array
	case ronNumber:
		var decoded any
		dec := json.NewDecoder(bytes.NewBufferString(string(value)))
		dec.UseNumber()
		if err := dec.Decode(&decoded); err != nil {
			return string(value)
		}
		return decoded
	default:
		return value
	}
}
