package ron

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

type marshalSample struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Alias    string   `ron:"ronAlias" json:"jsonAlias"`
	Empty    string   `json:"empty,omitempty"`
	RONEmpty string   `ron:"ronEmpty,omitempty" json:"jsonEmpty"`
	Hidden   string   `json:"-"`
	List     []string `json:"list"`
}

func TestMarshalAPIsUseReferenceGoldens(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	input := readConformanceFile(t, root, "valid/basic/records/input.json")
	value, err := decodeJSON(input, nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}
	for _, check := range []struct {
		name string
		call func(any) ([]byte, error)
		path string
	}{
		{"Marshal", Marshal, "expected.compact.ron"},
		{"MarshalPretty", MarshalPretty, "expected.pretty.ron"},
		{"MarshalCanonical", MarshalCanonical, "expected.canonical.ron"},
	} {
		t.Run(check.name, func(t *testing.T) {
			got, err := check.call(value)
			if err != nil {
				t.Fatalf("%s: %v", check.name, err)
			}
			assertBytesEqual(t, readConformanceFile(t, root, "valid/basic/records/"+check.path), got)
		})
	}
}

func TestMarshalReflectsStructsWithRONTagsThenJSONTags(t *testing.T) {
	got, err := MarshalPretty(marshalSample{
		ID: 1538289, Name: "Ada", Alias: "ron tag wins", Hidden: "secret", List: []string{"alpha", "beta"},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	jsonBody, err := ToJSON(got)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(jsonBody, &value); err != nil || len(value) != 4 || value["id"] != float64(1538289) || value["name"] != "Ada" || value["ronAlias"] != "ron tag wins" {
		t.Fatalf("Marshal struct JSON = %q, %v", jsonBody, err)
	}
	list, ok := value["list"].([]any)
	if !ok || len(list) != 2 || list[0] != "alpha" || list[1] != "beta" {
		t.Fatalf("Marshal struct list = %#v", value["list"])
	}
}

func TestMarshalReflectsMapsAndSlices(t *testing.T) {
	got, err := Marshal(map[uint64]any{1538289: map[string]any{"active": true, "count": 2, "name": "Ada"}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	jsonBody, err := ToJSON(got)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var value map[string]map[string]any
	if err := json.Unmarshal(jsonBody, &value); err != nil || len(value) != 1 || value["1538289"]["active"] != true || value["1538289"]["count"] != float64(2) || value["1538289"]["name"] != "Ada" {
		t.Fatalf("Marshal map JSON = %q, %v", jsonBody, err)
	}
}

func TestMarshalPrettyHandlesDeepUnaryNesting(t *testing.T) {
	value := deeplyNestedUnaryValue(64)
	got, err := MarshalPretty(value)
	if err != nil {
		t.Fatalf("MarshalPretty: %v", err)
	}
	gotJSON, err := ToJSON(got, Mode(Canonical))
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	wantJSON, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatal("deeply nested pretty output changed its value")
	}
}

func TestMaxNestingDepthLimitsSingleDocuments(t *testing.T) {
	if _, err := ToJSON([]byte("item {value true}"), MaxNestingDepth(1)); !errors.Is(err, ErrNestingTooDeep) {
		t.Fatalf("ToJSON nesting error = %v, want %v", err, ErrNestingTooDeep)
	}
	var buf bytes.Buffer
	encoder := NewEncoder(&buf, Mode(Compact), MaxNestingDepth(1))
	if err := encoder.Encode(map[string]any{"item": map[string]bool{"value": true}}); !errors.Is(err, ErrNestingTooDeep) {
		t.Fatalf("Encoder nesting error = %v, want %v", err, ErrNestingTooDeep)
	}
}

func TestEncoderEncodeWritesRONValueWithTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf, Mode(Compact))
	if err := encoder.Encode(map[string]int{"b": 2}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
		t.Fatal("Encode did not add a trailing newline")
	}
	if _, err := ToJSON(bytes.TrimSuffix(buf.Bytes(), []byte("\n"))); err != nil {
		t.Fatalf("Encode output is not RON: %v", err)
	}
}

func TestEncoderSetIndentSelectsPrettyOutput(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf, Mode(Compact))
	encoder.SetIndent("    ")
	if err := encoder.Encode(map[string]any{"outer": map[string]int{"b": 2, "a": 1}}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\n    ")) {
		t.Fatal("SetIndent did not set pretty indentation")
	}
}

func TestEncoderReturnsWriterErrors(t *testing.T) {
	encoder := NewEncoder(errorWriter{})
	if err := encoder.Encode(map[string]int{"b": 2}); !errors.Is(err, errWriteFailed) {
		t.Fatalf("Encode error = %v, want %v", err, errWriteFailed)
	}
}

func TestMarshalUsesJSONMarshalerForCompatibility(t *testing.T) {
	got, err := Marshal(customJSONValue{})
	if err != nil {
		t.Fatalf("Marshal custom JSON value: %v", err)
	}
	jsonBody, err := ToJSON(got)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(jsonBody, &value); err != nil || len(value) != 2 || value["name"] != "Ada" || value["id"] != float64(1538289) {
		t.Fatalf("Marshal JSON Marshaler value = %q, %v", jsonBody, err)
	}
}

func TestMarshalPreservesByteSliceCompatibility(t *testing.T) {
	got, err := Marshal([]byte{1, 2})
	if err != nil {
		t.Fatalf("Marshal byte slice: %v", err)
	}
	jsonBody, err := ToJSON(got)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var value string
	if err := json.Unmarshal(jsonBody, &value); err != nil || value != "AQI=" {
		t.Fatalf("Marshal byte slice JSON = %q, %v", jsonBody, err)
	}
}

func TestMarshalPreservesNilSliceAndMapCompatibility(t *testing.T) {
	var array []any
	arrayRON, err := Marshal(array)
	if err != nil {
		t.Fatalf("Marshal nil slice: %v", err)
	}
	var object map[string]any
	objectRON, err := Marshal(object)
	if err != nil {
		t.Fatalf("Marshal nil map: %v", err)
	}
	for _, got := range [][]byte{arrayRON, objectRON} {
		jsonBody, err := ToJSON(got)
		if err != nil {
			t.Fatalf("ToJSON: %v", err)
		}
		var value any
		if err := json.Unmarshal(jsonBody, &value); err != nil || value != nil {
			t.Fatalf("Marshal nil JSON = %q, %v", jsonBody, err)
		}
	}
}

func TestMarshalRejectsUnsupportedValues(t *testing.T) {
	if _, err := Marshal(func() {}); err == nil {
		t.Fatal("Marshal accepted function")
	}
}

var errWriteFailed = errors.New("write failed")

type customJSONValue struct{}

func (customJSONValue) MarshalJSON() ([]byte, error) {
	return []byte(`{"name":"Ada","id":1538289}`), nil
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errWriteFailed }
