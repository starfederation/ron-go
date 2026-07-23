package ron

import (
	"bytes"
	"testing"
)

func TestPrettyRONByteAPIsOmitTrailingNewline(t *testing.T) {
	multilineWant := []byte("list [\n" +
		"  {\n" +
		"    a 1\n" +
		"    b 2\n" +
		"  }\n" +
		"  {\n" +
		"    c 3\n" +
		"    d 4\n" +
		"  }\n" +
		"]\n" +
		"outer {\n" +
		"  a 1\n" +
		"  b 2\n" +
		"}")
	cases := []struct {
		name      string
		jsonInput []byte
		goValue   any
		want      []byte
	}{
		{
			name:      "scalar",
			jsonInput: []byte(`true`),
			goValue:   true,
			want:      []byte("true"),
		},
		{
			name:      "list",
			jsonInput: []byte(`[1,2]`),
			goValue:   []int{1, 2},
			want:      []byte("[1 2]"),
		},
		{
			name:      "object",
			jsonInput: []byte(`{}`),
			goValue:   map[string]any{},
			want:      []byte("{}"),
		},
		{
			name:      "brace-elided object members",
			jsonInput: []byte(`{"a":1,"b":2}`),
			goValue:   map[string]int{"a": 1, "b": 2},
			want:      []byte("a 1\nb 2"),
		},
		{
			name:      "multiline indentation",
			jsonInput: []byte(`{"list":[{"b":2,"a":1},{"d":4,"c":3}],"outer":{"b":2,"a":1}}`),
			goValue: map[string]any{
				"list": []any{
					map[string]int{"b": 2, "a": 1},
					map[string]int{"d": 4, "c": 3},
				},
				"outer": map[string]int{"b": 2, "a": 1},
			},
			want: multilineWant,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fromJSON, err := FromJSON(tc.jsonInput)
			if err != nil {
				t.Fatalf("FromJSON: %v", err)
			}
			assertBytesEqual(t, tc.want, fromJSON)

			var fromJSONBuf bytes.Buffer
			fromJSONBuf.WriteString("prefix:")
			fromJSONInto, err := FromJSONInto(&fromJSONBuf, tc.jsonInput)
			if err != nil {
				t.Fatalf("FromJSONInto: %v", err)
			}
			assertBytesEqual(t, append([]byte("prefix:"), tc.want...), fromJSONInto)

			marshaled, err := Marshal(tc.goValue)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			assertBytesEqual(t, tc.want, marshaled)

			var marshalBuf bytes.Buffer
			marshalBuf.WriteString("prefix:")
			marshaledInto, err := MarshalInto(&marshalBuf, tc.goValue)
			if err != nil {
				t.Fatalf("MarshalInto: %v", err)
			}
			assertBytesEqual(t, append([]byte("prefix:"), tc.want...), marshaledInto)
		})
	}
}

func TestCompactRONByteAPIsOmitTrailingNewline(t *testing.T) {
	jsonInput := []byte(`{"a":1,"b":[2,3]}`)
	value := map[string]any{
		"a": 1,
		"b": []int{2, 3},
	}
	want := []byte("a 1 b[2 3]")

	fromJSON, err := FromJSONCompact(jsonInput)
	if err != nil {
		t.Fatalf("FromJSONCompact: %v", err)
	}
	assertBytesEqual(t, want, fromJSON)

	var fromJSONBuf bytes.Buffer
	fromJSONBuf.WriteString("prefix:")
	fromJSONInto, err := FromJSONCompactInto(&fromJSONBuf, jsonInput)
	if err != nil {
		t.Fatalf("FromJSONCompactInto: %v", err)
	}
	assertBytesEqual(t, append([]byte("prefix:"), want...), fromJSONInto)

	marshaled, err := MarshalCompact(value)
	if err != nil {
		t.Fatalf("MarshalCompact: %v", err)
	}
	assertBytesEqual(t, want, marshaled)

	var marshalBuf bytes.Buffer
	marshalBuf.WriteString("prefix:")
	marshaledInto, err := MarshalCompactInto(&marshalBuf, value)
	if err != nil {
		t.Fatalf("MarshalCompactInto: %v", err)
	}
	assertBytesEqual(t, append([]byte("prefix:"), want...), marshaledInto)
}

func TestJSONByteAPIsOmitTrailingNewline(t *testing.T) {
	input := []byte("a 1\nb [2 3]")
	cases := []struct {
		name    string
		options []Option
		want    []byte
	}{
		{
			name: "compact",
			want: []byte(`{"a":1,"b":[2,3]}`),
		},
		{
			name:    "pretty",
			options: []Option{PrettyJSON("", "  ")},
			want: []byte("{\n" +
				"  \"a\": 1,\n" +
				"  \"b\": [\n" +
				"    2,\n" +
				"    3\n" +
				"  ]\n" +
				"}"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToJSON(input, tc.options...)
			if err != nil {
				t.Fatalf("ToJSON: %v", err)
			}
			assertBytesEqual(t, tc.want, got)

			var buf bytes.Buffer
			buf.WriteString("prefix:")
			got, err = ToJSONInto(&buf, input, tc.options...)
			if err != nil {
				t.Fatalf("ToJSONInto: %v", err)
			}
			assertBytesEqual(t, append([]byte("prefix:"), tc.want...), got)
		})
	}
}
