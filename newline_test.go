package ron

import (
	"bytes"
	"testing"
)

func TestRONByteAPIsUseReferenceGoldens(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	cases := []struct {
		name string
		path string
	}{
		{"scalar", "valid/ported/valid/y_root_scalar_false"},
		{"array", "valid/ported/valid/y_structure_array_empty"},
		{"empty object", "valid/ported/valid/y_structure_object_empty"},
		{"root object", "valid/ron_only/valid/top_level_elided_map"},
		{"multiline", "valid/basic/records"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.path+"/input.json")
			for _, check := range []struct {
				mode OutputMode
				path string
			}{
				{Pretty, "expected.pretty.ron"},
				{Compact, "expected.compact.ron"},
				{Canonical, "expected.canonical.ron"},
			} {
				expected := readConformanceFile(t, root, tc.path+"/"+check.path)
				got, err := FromJSON(input, Mode(check.mode))
				if err != nil {
					t.Fatalf("FromJSON %s: %v", check.mode, err)
				}
				assertBytesEqual(t, expected, got)

				var buffer bytes.Buffer
				buffer.WriteString("prefix:")
				got, err = FromJSONInto(&buffer, input, Mode(check.mode))
				if err != nil {
					t.Fatalf("FromJSONInto %s: %v", check.mode, err)
				}
				prefixed := append(append([]byte{}, "prefix:"...), expected...)
				assertBytesEqual(t, prefixed, got)
				assertBytesEqual(t, prefixed, buffer.Bytes())
			}
		})
	}
}

func TestJSONByteAPIsUseReferenceGoldens(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	for _, tc := range []struct {
		name string
		path string
	}{
		{"records", "valid/basic/records"},
		{"negative zero", "valid/ported/valid/y_number_negative_zero"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.path+"/input.ron")
			for _, check := range []struct {
				mode OutputMode
				path string
			}{
				{Pretty, "expected.pretty.json"},
				{Compact, "expected.compact.json"},
				{Canonical, "expected.canonical.json"},
			} {
				expected := readConformanceFile(t, root, tc.path+"/"+check.path)
				got, err := ToJSON(input, Mode(check.mode))
				if err != nil {
					t.Fatalf("ToJSON %s: %v", check.mode, err)
				}
				assertBytesEqual(t, expected, got)

				var buffer bytes.Buffer
				buffer.WriteString("prefix:")
				got, err = ToJSONInto(&buffer, input, Mode(check.mode))
				if err != nil {
					t.Fatalf("ToJSONInto %s: %v", check.mode, err)
				}
				prefixed := append(append([]byte{}, "prefix:"...), expected...)
				assertBytesEqual(t, prefixed, got)
				assertBytesEqual(t, prefixed, buffer.Bytes())
			}
		})
	}
}
