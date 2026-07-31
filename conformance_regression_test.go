package ron

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRFC8785StringLessUsesUTF16Order(t *testing.T) {
	if !rfc8785StringLess("\U0001f600", "\ue000") {
		t.Fatal("rfc8785StringLess did not use UTF-16 order")
	}
	if rfc8785StringLess("\ue000", "\U0001f600") {
		t.Fatal("rfc8785StringLess reversed UTF-16 order")
	}
}

func TestRFC8785StringLessDoesNotAllocate(t *testing.T) {
	allocations := testing.AllocsPerRun(1000, func() {
		_ = rfc8785StringLess("created", "digest")
		_ = rfc8785StringLess("\U0001f600", "\ue000")
	})
	if allocations != 0 {
		t.Fatalf("rfc8785StringLess allocated %v times", allocations)
	}
}

func TestBaseDuplicateKeysUseLastValue(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	got, err := ToJSON(readConformanceFile(t, root, "valid/basic/escapes/input.ron"), Mode(Compact))
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	assertBytesEqual(t, readConformanceFile(t, root, "valid/basic/escapes/expected.compact.json"), got)

	rfcRoot, _ := loadRFC8785Manifest(t)
	ronBody, err := FromJSON(readConformanceFile(t, rfcRoot, "invalid/duplicate_property_names.json"), Mode(Compact))
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}
	jsonBody, err := ToJSON(ronBody, Mode(Compact))
	if err != nil {
		t.Fatalf("ToJSON generated RON: %v", err)
	}
	var value map[string]float64
	if err := json.Unmarshal(jsonBody, &value); err != nil || len(value) != 1 || value["a"] != 2 {
		t.Fatalf("base duplicate JSON = %s, %v", jsonBody, err)
	}
}

func TestRONContainsVocabularyMarker(t *testing.T) {
	cases := []struct {
		source []byte
		want   bool
	}{
		{[]byte("query {filter {status active}}"), false},
		{[]byte("entity {# 123}"), true},
		{[]byte("created {#utc 2026-06-13T00:00:00Z}"), true},
		{[]byte(`entity {"\u0023" 123}`), true},
	}
	for _, test := range cases {
		if got := containsVocabularyMarker(test.source); got != test.want {
			t.Fatalf("containsVocabularyMarker(%q) = %v, want %v", test.source, got, test.want)
		}
	}
}

func TestUnsupportedVocabularyIsRejected(t *testing.T) {
	option := EnableVocabularies("https://example.com/vocab/unknown/v1")
	if _, err := ToJSON([]byte("item {name alpha}"), option); err == nil {
		t.Fatal("ToJSON accepted unsupported vocabulary")
	}
	if _, err := FromJSON([]byte(`{"item":"alpha"}`), option); err == nil {
		t.Fatal("FromJSON accepted unsupported vocabulary")
	}
}

func TestCanonicalModeValidatesVocabularyPayloads(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		input := readConformanceFile(t, root, tc.InputJSON)
		if _, err := FromJSON(input, Mode(Canonical)); err == nil {
			t.Fatalf("canonical FromJSON accepted invalid vocabulary payload: %s", tc.Name)
		}
		value, err := decodeJSON(input, nil)
		if err != nil {
			t.Fatalf("decode invalid vocabulary fixture: %v", err)
		}
		var ronBody bytes.Buffer
		writeCompactValueWithCustom(&ronBody, value, true, false, nil)
		if _, err := ToJSON(ronBody.Bytes(), Mode(Canonical)); err == nil {
			t.Fatalf("canonical ToJSON accepted invalid vocabulary payload: %s", tc.Name)
		}
	}
}

func TestConversionBufferReuse(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	input := readConformanceFile(t, root, "valid/basic/records/input.json")
	var buffer bytes.Buffer
	pretty, err := FromJSONInto(&buffer, input, Mode(Pretty))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, readConformanceFile(t, root, "valid/basic/records/expected.pretty.ron"), pretty)

	buffer.Reset()
	compact, err := FromJSONInto(&buffer, input, Mode(Compact))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, readConformanceFile(t, root, "valid/basic/records/expected.compact.ron"), compact)
}
