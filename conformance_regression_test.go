package ron

import (
	"bytes"
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

func TestBaseDuplicateKeysUseLastValueAndMoveItLast(t *testing.T) {
	got, err := ToJSON([]byte("{a 1 b 2 a 3}"), Mode(Compact))
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	assertBytesEqual(t, []byte(`{"b":2,"a":3}`), got)

	ron, err := FromJSON([]byte(`{"a":1,"b":2,"a":3}`), Mode(Compact))
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}
	assertBytesEqual(t, []byte("b 2 a 3"), ron)
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

func TestCanonicalModesValidateVocabularyPayloads(t *testing.T) {
	invalidRON := []byte("id {#uid not-a-uuid}")
	if _, err := ToJSON(invalidRON, Mode(Canonical)); err == nil {
		t.Fatal("canonical ToJSON accepted invalid vocabulary payload")
	}
	invalidJSON := []byte(`{"id":{"#uid":"not-a-uuid"}}`)
	if _, err := FromJSON(invalidJSON, Mode(Canonical)); err == nil {
		t.Fatal("canonical FromJSON accepted invalid vocabulary payload")
	}
}

func TestConversionBufferReuse(t *testing.T) {
	var buffer bytes.Buffer
	pretty, err := FromJSONInto(&buffer, []byte(`{"a":1}`), Mode(Pretty))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("a 1\n"), pretty)

	buffer.Reset()
	compact, err := FromJSONInto(&buffer, []byte(`{"b":2}`), Mode(Compact))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("b 2"), compact)
}

func TestModesSelectDistinctOutputs(t *testing.T) {
	source := []byte(`{"b":1,"a":2}`)
	pretty, err := FromJSON(source, Mode(Pretty))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("b 1\na 2\n"), pretty)
	compact, err := FromJSON(source, Mode(Compact))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("b 1 a 2"), compact)
	canonical, err := FromJSON(source, Mode(Canonical))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("a 2 b 1"), canonical)
}
