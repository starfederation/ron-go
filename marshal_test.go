package ron

import (
	"bytes"
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

func TestMarshalReflectsStructsWithRONTagsThenJSONTags(t *testing.T) {
	got, err := MarshalPretty(marshalSample{
		ID:     1538289,
		Name:   "Ada",
		Alias:  "ron tag wins",
		Hidden: "secret",
		List:   []string{"alpha", "beta"},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := []byte("id 1538289\nname Ada\nronAlias 'ron tag wins'\nlist [alpha beta]\n")
	assertBytesEqual(t, want, got)
}

func TestMarshalReflectsMapsAndSlices(t *testing.T) {
	got, err := Marshal(map[uint64]any{
		1538289: map[string]any{
			"active": true,
			"count":  2,
			"name":   "Ada",
		},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := []byte("1538289{active true count 2 name Ada}")
	assertBytesEqual(t, want, got)
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
	enc := NewEncoder(&buf, Mode(Compact))
	if err := enc.Encode(map[string]int{"b": 2}); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	assertBytesEqual(t, []byte("b 2\n"), buf.Bytes())
}

func TestEncoderSetIndentSelectsPrettyOutput(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf, Mode(Compact))
	enc.SetIndent("    ")
	if err := enc.Encode(map[string]any{"outer": map[string]int{"b": 2, "a": 1}}); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	assertBytesEqual(t, []byte("outer {\n    a 1\n    b 2\n}\n"), buf.Bytes())
}

func TestEncoderReturnsWriterErrors(t *testing.T) {
	enc := NewEncoder(errorWriter{})
	if err := enc.Encode(map[string]int{"b": 2}); !errors.Is(err, errWriteFailed) {
		t.Fatalf("Encode error = %v, want %v", err, errWriteFailed)
	}
}

func TestMarshalUsesJSONMarshalerForCompatibility(t *testing.T) {
	got, err := Marshal(customJSONValue{})
	if err != nil {
		t.Fatalf("Marshal custom JSON value: %v", err)
	}

	assertBytesEqual(t, []byte("name Ada id 1538289"), got)
}

func TestMarshalPreservesByteSliceCompatibility(t *testing.T) {
	got, err := Marshal([]byte{1, 2})
	if err != nil {
		t.Fatalf("Marshal byte slice: %v", err)
	}

	assertBytesEqual(t, []byte("AQI="), got)
}

func TestMarshalPreservesNilSliceAndMapCompatibility(t *testing.T) {
	var array []any
	arrayRON, err := Marshal(array)
	if err != nil {
		t.Fatalf("Marshal nil slice: %v", err)
	}
	assertBytesEqual(t, []byte("null"), arrayRON)

	var object map[string]any
	objectRON, err := Marshal(object)
	if err != nil {
		t.Fatalf("Marshal nil map: %v", err)
	}
	assertBytesEqual(t, []byte("null"), objectRON)
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

func (errorWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}
