package ron

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

type vocabularyManifest struct {
	Valid   []vocabularyCase `json:"valid"`
	Invalid []vocabularyCase `json:"invalid"`
}

type vocabularyCase struct {
	Name         string   `json:"name"`
	Vocabularies []string `json:"vocabularies"`
	InputJSON    string   `json:"inputJSON"`
	ExpectedRON  string   `json:"expectedRON"`
}

func loadVocabularyManifest(t *testing.T) (string, vocabularyManifest) {
	t.Helper()
	root := testdataSubdir(t, "vocabularies")
	body := readFile(t, filepath.Join(root, "manifest.json"))

	var manifest vocabularyManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("unmarshal vocabulary manifest: %v", err)
	}
	return root, manifest
}

func TestSupportedVocabulariesAreEnabledByDefault(t *testing.T) {
	got, err := FromJSON([]byte(`{
		"id":{"#uid":"00112233-4455-6677-8899-aabbccddeeff"},
		"created":{"#utc":"2026-06-13T00:00:00Z"}
	}`))
	if err != nil {
		t.Fatalf("FromJSON default vocabularies: %v", err)
	}
	assertBytesEqual(t, []byte("created {#utc 2026-06-13T00:00:00Z}\nid {#uid 00112233-4455-6677-8899-aabbccddeeff}\n"), got)

	if _, err := FromJSON([]byte(`{"bad":{"#dur":"P1M"}}`)); err == nil {
		t.Fatal("FromJSON accepted invalid default vocabulary payload")
	}
}

func hasVocabulary(vocabularies []string, vocabulary string) bool {
	for _, candidate := range vocabularies {
		if candidate == vocabulary {
			return true
		}
	}
	return false
}
