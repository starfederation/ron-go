package ron

import "testing"

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
	for _, value := range []string{
		string([]byte{0xff}),
		"\ufdd0",
		"\U0001fffe",
	} {
		if _, err := MarshalCanonical(value); err == nil {
			t.Fatalf("MarshalCanonical accepted %q", value)
		}
	}
	got, err := MarshalCanonical("\ufffd")
	if err != nil {
		t.Fatalf("MarshalCanonical replacement character: %v", err)
	}
	assertBytesEqual(t, []byte("�"), got)
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
	assertBytesEqual(t, []byte("value first"), got)
}

func TestCanonicalToJSONRejectsDuplicatesBeforeVocabularyParsing(t *testing.T) {
	source := []byte("a 1 a 2 id {#uid 00112233-4455-6677-8899-aabbccddeeff}")
	if _, err := ToJSON(source, Mode(Canonical)); err == nil {
		t.Fatal("canonical ToJSON accepted duplicate names with a vocabulary value")
	}
}

func TestCanonicalFromJSONRejectsDuplicatesBeforeMapping(t *testing.T) {
	mapper := MapJSONValues(func(_ []JSONPathSegment, _ any) (any, bool) {
		return nil, false
	})
	if _, err := FromJSON([]byte(`{"a":1,"a":2}`), Mode(Canonical), mapper); err == nil {
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
