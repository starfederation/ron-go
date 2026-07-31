package ron

import (
	"bytes"
	"testing"
)

func TestFromJSONPrettyRootObjectsElideBraces(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	got, err := FromJSON(readConformanceFile(t, root, "rendering/root_object_elision/input.json"))
	if err != nil {
		t.Fatalf("FromJSON pretty root object: %v", err)
	}
	assertBytesEqual(t, readConformanceFile(t, root, "rendering/root_object_elision/expected.pretty.ron"), got)
}

func TestDefaultModeIsPretty(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	input := readConformanceFile(t, root, "valid/basic/records/input.ron")
	got, err := ToJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, readConformanceFile(t, root, "valid/basic/records/expected.pretty.json"), got)
}

func TestJSONIndentChangesPrettyJSONIndentation(t *testing.T) {
	root, _ := loadConformanceManifest(t)
	input := readConformanceFile(t, root, "valid/basic/records/input.ron")
	defaultJSON, err := ToJSON(input, Mode(Pretty))
	if err != nil {
		t.Fatal(err)
	}
	indentedJSON, err := ToJSON(input, Mode(Pretty), JSONIndent("", "--"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(defaultJSON, indentedJSON) || !bytes.Contains(indentedJSON, []byte("--\"")) {
		t.Fatal("JSONIndent did not change pretty JSON indentation")
	}
}
