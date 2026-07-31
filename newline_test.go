package ron

import (
	"bytes"
	"testing"
)

func TestPrettyRONByteAPIsEndWithNewline(t *testing.T) {
	multiline := []byte("list [\n  {\n    a 1\n    b 2\n  }\n  {\n    c 3\n    d 4\n  }\n]\nouter {\n  a 1\n  b 2\n}\n")
	cases := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{"scalar", []byte(`true`), []byte("true\n")},
		{"array", []byte(`[1,2]`), []byte("[1 2]\n")},
		{"empty object", []byte(`{}`), []byte("{}\n")},
		{"brace-elided object", []byte(`{"a":1,"b":2}`), []byte("a 1\nb 2\n")},
		{"multiline indentation", []byte(`{"list":[{"a":1,"b":2},{"c":3,"d":4}],"outer":{"a":1,"b":2}}`), multiline},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := FromJSON(test.input, Mode(Pretty))
			if err != nil {
				t.Fatalf("FromJSON: %v", err)
			}
			assertBytesEqual(t, test.want, got)

			var buffer bytes.Buffer
			buffer.WriteString("prefix:")
			got, err = FromJSONInto(&buffer, test.input, Mode(Pretty))
			if err != nil {
				t.Fatalf("FromJSONInto: %v", err)
			}
			assertBytesEqual(t, append([]byte("prefix:"), test.want...), got)
		})
	}
}

func TestCompactAndCanonicalRONByteAPIsOmitTrailingNewline(t *testing.T) {
	input := []byte(`{"b":1,"a":[2,3]}`)
	cases := []struct {
		mode OutputMode
		want []byte
	}{
		{Compact, []byte("b 1 a[2 3]")},
		{Canonical, []byte("a[2 3] b 1")},
	}
	for _, test := range cases {
		got, err := FromJSON(input, Mode(test.mode))
		if err != nil {
			t.Fatalf("FromJSON %s: %v", test.mode, err)
		}
		assertBytesEqual(t, test.want, got)
		if bytes.HasSuffix(got, []byte("\n")) {
			t.Fatalf("FromJSON %s added newline", test.mode)
		}
	}
}

func TestJSONByteAPIsAndConversionBuffer(t *testing.T) {
	input := []byte("a 1\nb [2 3]")
	cases := []struct {
		mode OutputMode
		want []byte
	}{
		{Compact, []byte(`{"a":1,"b":[2,3]}`)},
		{Pretty, []byte("{\n  \"a\": 1,\n  \"b\": [\n    2,\n    3\n  ]\n}")},
		{Canonical, []byte(`{"a":1,"b":[2,3]}`)},
	}
	for _, test := range cases {
		got, err := ToJSON(input, Mode(test.mode))
		if err != nil {
			t.Fatalf("ToJSON %s: %v", test.mode, err)
		}
		assertBytesEqual(t, test.want, got)

		var buffer bytes.Buffer
		buffer.WriteString("prefix:")
		got, err = ToJSONInto(&buffer, input, Mode(test.mode))
		if err != nil {
			t.Fatalf("ToJSONInto %s: %v", test.mode, err)
		}
		assertBytesEqual(t, append([]byte("prefix:"), test.want...), got)
	}
}
