package ron

import "testing"

func TestFromJSONPrettyRootObjectsElideBraces(t *testing.T) {
	got, err := FromJSON([]byte(`{"status":"ok"}`))
	if err != nil {
		t.Fatalf("FromJSON pretty root object: %v", err)
	}
	assertBytesEqual(t, []byte("status ok\n"), got)
}

func TestFromJSONValueMapperCanRenderTaggedRONValues(t *testing.T) {
	input := []byte(`{"tx":"tx-48830","committed":"2026-06-13T00:00:00Z","reactorId":"reactor-BY"}`)
	got, err := FromJSON(
		input,
		IsCanonical(false),
		MapJSONValues(func(path []JSONPathSegment, value any) (any, bool) {
			if len(path) != 1 || path[0].IsIndex {
				return nil, false
			}

			switch path[0].Key {
			case "tx", "reactorId":
				return Tagged("", value), true
			case "committed":
				return Tagged("time", value), true
			default:
				return nil, false
			}
		}),
	)
	if err != nil {
		t.Fatalf("FromJSON tagged values: %v", err)
	}

	want := []byte("tx {# tx-48830}\ncommitted {#time 2026-06-13T00:00:00Z}\nreactorId {# reactor-BY}\n")
	assertBytesEqual(t, want, got)
}
