package ron

import (
	"bytes"
	"encoding/json"

	"github.com/paulmach/orb/geojson"
)

const (
	// VocabularyGeoV1 is the RON GeoJSON typed vocabulary URI.
	VocabularyGeoV1 = "https://ron.dev/vocab/geo/v1"
)

// GeoJSON is a geo vocabulary #geo value.
type GeoJSON struct {
	Value any
}

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
	body, err := json.Marshal(jsonCompatibleValue(payload))
	if err != nil {
		return nil, newError("invalid #geo payload")
	}
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &header); err != nil || header.Type == "" {
		return nil, newError("invalid #geo payload")
	}
	if !validGeoJSONShape(jsonCompatibleValue(payload)) {
		return nil, newError("invalid #geo payload")
	}
	switch header.Type {
	case "Feature":
		_, _ = geojson.UnmarshalFeature(body)
	case "FeatureCollection":
		_, _ = geojson.UnmarshalFeatureCollection(body)
	default:
		_, _ = geojson.UnmarshalGeometry(body)
	}
	parsed, err := opts.parseVocabularyValue(payload)
	if err != nil {
		return nil, err
	}
	return GeoJSON{Value: parsed}, nil
}

func geoTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case GeoJSON:
		return objectMember{Key: "#geo", Value: value.Value}, true
	default:
		return objectMember{}, false
	}
}

func validGeoJSONShape(value any) bool {
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	geoType, ok := object["type"].(string)
	if !ok {
		return false
	}
	switch geoType {
	case "Point":
		return validPosition(object["coordinates"])
	case "MultiPoint", "LineString":
		return validPositionArray(object["coordinates"])
	case "MultiLineString", "Polygon":
		return validNestedPositions(object["coordinates"], 2)
	case "MultiPolygon":
		return validNestedPositions(object["coordinates"], 3)
	case "GeometryCollection":
		geometries, ok := object["geometries"].([]any)
		if !ok {
			return false
		}
		for _, geometry := range geometries {
			if !validGeoJSONShape(geometry) {
				return false
			}
		}
		return true
	case "Feature":
		if geometry := object["geometry"]; geometry != nil && !validGeoJSONShape(geometry) {
			return false
		}
		_, ok := object["properties"]
		return ok
	case "FeatureCollection":
		features, ok := object["features"].([]any)
		if !ok {
			return false
		}
		for _, feature := range features {
			if !validGeoJSONShape(feature) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validNestedPositions(value any, depth int) bool {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if depth == 1 {
			if !validPosition(value) {
				return false
			}
		} else if !validNestedPositions(value, depth-1) {
			return false
		}
	}
	return true
}

func validPositionArray(value any) bool {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !validPosition(value) {
			return false
		}
	}
	return true
}

func validPosition(value any) bool {
	values, ok := value.([]any)
	if !ok || len(values) < 2 || len(values) > 3 {
		return false
	}
	for _, value := range values {
		switch value.(type) {
		case json.Number, float64, int64, uint64:
		default:
			return false
		}
	}
	return true
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
