package ron

import (
	"bytes"
	"encoding/json"
	"testing"
)

type statefulJSONValue struct {
	calls int
}

func (value *statefulJSONValue) MarshalJSON() ([]byte, error) {
	value.calls++
	if value.calls == 1 {
		return []byte(`{"value":"first"}`), nil
	}
	return []byte(`{"value":"second"}`), nil
}

type duplicateJSONMembers struct{}

func (duplicateJSONMembers) MarshalJSON() ([]byte, error) {
	return []byte(`{"name":"first","name":"second"}`), nil
}

func TestCanonicalStringValidation(t *testing.T) {
	for _, value := range []string{string([]byte{0xff}), "\ufdd0", "\U0001fffe"} {
		if _, err := MarshalCanonical(value); err == nil {
			t.Fatalf("MarshalCanonical accepted %q", value)
		}
	}
	got, err := MarshalCanonical("\ufffd")
	if err != nil {
		t.Fatalf("MarshalCanonical replacement character: %v", err)
	}
	jsonBody, err := ToJSON(got, Mode(Canonical))
	if err != nil {
		t.Fatalf("ToJSON canonical replacement character: %v", err)
	}
	var value string
	if err := json.Unmarshal(jsonBody, &value); err != nil || value != "\ufffd" {
		t.Fatalf("canonical replacement character = %q, %v", jsonBody, err)
	}
}

func TestMarshalCanonicalRejectsDuplicateJSONMarshalerNames(t *testing.T) {
	if _, err := MarshalCanonical(duplicateJSONMembers{}); err == nil {
		t.Fatal("MarshalCanonical accepted duplicate JSON names")
	}
}

func TestMarshalCanonicalCallsJSONMarshalerOnce(t *testing.T) {
	value := &statefulJSONValue{}
	got, err := MarshalCanonical(value)
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	if value.calls != 1 {
		t.Fatalf("MarshalJSON calls = %d, want 1", value.calls)
	}
	jsonBody, err := ToJSON(got, Mode(Canonical))
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(jsonBody, &decoded); err != nil || decoded["value"] != "first" {
		t.Fatalf("canonical JSON Marshaler value = %q, %v", jsonBody, err)
	}
}

func TestCanonicalToJSONRejectsDuplicatesBeforeVocabularyParsing(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	source := readConformanceFile(t, root, "canonical/invalid/duplicate_key.ron")
	source = append(bytes.TrimSuffix(source, []byte("}")), []byte(" id {#uid 00112233-4455-6677-8899-aabbccddeeff}}")...)
	if _, err := ToJSON(source, Mode(Canonical)); err == nil {
		t.Fatal("canonical ToJSON accepted duplicate names with a vocabulary value")
	}
}

func TestCanonicalFromJSONRejectsDuplicatesBeforeMapping(t *testing.T) {
	root, _ := loadRFC8785Manifest(t)
	source := readConformanceFile(t, root, "invalid/duplicate_property_names.json")
	mapper := MapJSONValues(func(_ []JSONPathSegment, _ any) (any, bool) { return nil, false })
	if _, err := FromJSON(source, Mode(Canonical), mapper); err == nil {
		t.Fatal("canonical FromJSON accepted duplicate names with a value mapper")
	}
}

func TestCanonicalMaxNestingDepth(t *testing.T) {
	if _, err := ToJSON([]byte("[[1]]"), Mode(Canonical), MaxNestingDepth(1)); err == nil {
		t.Fatal("canonical ToJSON accepted excessive nesting")
	}
	if _, err := FromJSON([]byte("[[1]]"), Mode(Canonical), MaxNestingDepth(1)); err == nil {
		t.Fatal("canonical FromJSON accepted excessive nesting")
	}
}
