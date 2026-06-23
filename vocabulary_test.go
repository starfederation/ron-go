package ron

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

type vocabularyManifest struct {
	Valid           []vocabularyCase `json:"valid"`
	Invalid         []vocabularyCase `json:"invalid"`
	InvalidProfiles []struct {
		Name    string `json:"name"`
		Profile string `json:"profile"`
		Reason  string `json:"reason"`
	} `json:"invalidProfiles"`
	Registry string `json:"registry"`
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

func TestVocabularyRegistryFixtureLoads(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	if manifest.Registry == "" {
		t.Fatal("manifest registry is empty")
	}
	var registry struct {
		Version      int `json:"version"`
		Vocabularies []struct {
			URI string `json:"uri"`
		} `json:"vocabularies"`
	}
	if err := json.Unmarshal(readConformanceFile(t, root, manifest.Registry), &registry); err != nil {
		t.Fatalf("unmarshal vocabulary registry: %v", err)
	}
	if registry.Version == 0 || len(registry.Vocabularies) == 0 {
		t.Fatalf("registry not loaded: %#v", registry)
	}
}

func TestVocabularyInvalidProfileFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	if len(manifest.InvalidProfiles) == 0 {
		t.Fatal("manifest invalidProfiles is empty")
	}
	for _, tc := range manifest.InvalidProfiles {
		t.Run(tc.Name, func(t *testing.T) {
			profile := readConformanceFile(t, root, tc.Profile)
			if err := ValidateVocabularyProfile(profile); err == nil {
				t.Fatal("ValidateVocabularyProfile accepted invalid profile")
			}
		})
	}
}

func TestVocabularyProfileAllowsOptionalUnknown(t *testing.T) {
	if err := ValidateVocabularyProfile([]byte(`{"vocabularies":{"https://example.com/unknown/v1":false}}`)); err != nil {
		t.Fatalf("ValidateVocabularyProfile rejected optional unknown vocabulary: %v", err)
	}
}

func TestVocabularyProfileAllowsRegisteredCustomRequired(t *testing.T) {
	profile := []byte(`{"vocabularies":{"https://example.com/vocab/invoice/v1":true}}`)
	if err := ValidateVocabularyProfile(profile, invoiceVocabularyOption()); err != nil {
		t.Fatalf("ValidateVocabularyProfile rejected custom vocabulary: %v", err)
	}
}

func TestSupportedVocabulariesAreEnabledByDefault(t *testing.T) {
	got, err := FromJSON([]byte(`{
		"id":{"#uid":"00112233-4455-6677-8899-aabbccddeeff"},
		"created":{"#utc":"2026-06-13T00:00:00Z"},
		"accent":{"#clr":["oklch",0.7,0.15,230]}
	}`))
	if err != nil {
		t.Fatalf("FromJSON default vocabularies: %v", err)
	}
	assertBytesEqual(t, []byte("accent {#clr [oklch 0.7 0.15 230]}\ncreated {#utc 2026-06-13T00:00:00Z}\nid {#uid 00112233-4455-6677-8899-aabbccddeeff}\n"), got)

	escaped, err := FromJSON([]byte(`{"created":{"\u0023utc":"2026-06-13T00:00:00Z"}}`))
	if err != nil {
		t.Fatalf("FromJSON escaped vocabulary marker: %v", err)
	}
	assertBytesEqual(t, []byte("created {#utc 2026-06-13T00:00:00Z}\n"), escaped)

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
