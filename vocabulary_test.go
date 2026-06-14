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

func hasVocabulary(vocabularies []string, vocabulary string) bool {
	for _, candidate := range vocabularies {
		if candidate == vocabulary {
			return true
		}
	}
	return false
}
