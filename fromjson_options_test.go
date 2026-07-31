package ron

import "testing"

func TestFromJSONPrettyRootObjectsElideBraces(t *testing.T) {
	got, err := FromJSON([]byte(`{"status":"ok"}`))
	if err != nil {
		t.Fatalf("FromJSON pretty root object: %v", err)
	}
	assertBytesEqual(t, []byte("status ok\n"), got)
}

func TestModeDefaultsAndJSONIndent(t *testing.T) {
	pretty, err := ToJSON([]byte("a 1"))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("{\n  \"a\": 1\n}"), pretty)

	compact, err := ToJSON([]byte("a 1"), Mode(Compact), JSONIndent("ignored", "--"))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte(`{"a":1}`), compact)

	indented, err := ToJSON([]byte("a 1"), Mode(Pretty), JSONIndent("", "--"))
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, []byte("{\n--\"a\": 1\n}"), indented)
}

func TestFromJSONValueMapperCanRenderTaggedRONValues(t *testing.T) {
	input := []byte(`{"tx":"tx-48830","committed":"2026-06-13T00:00:00Z","reactorId":"reactor-BY"}`)
	got, err := FromJSON(input, Mode(Pretty), MapJSONValues(func(path []JSONPathSegment, value any) (any, bool) {
		if len(path) != 1 || path[0].IsIndex {
			return nil, false
		}
		switch path[0].Key {
		case "tx", "reactorId":
			return Tagged("", value), true
		case "committed":
			return Tagged("time", value), true
		}
		return nil, false
	}))
	if err != nil {
		t.Fatalf("FromJSON tagged values: %v", err)
	}
	assertBytesEqual(t, []byte("tx {# tx-48830}\ncommitted {#time 2026-06-13T00:00:00Z}\nreactorId {# reactor-BY}\n"), got)
}
