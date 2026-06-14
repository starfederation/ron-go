package ron

import "testing"

func TestCoreVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyCoreV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readConformanceFile(t, root, tc.ExpectedRON)

			got, err := FromJSON(input, EnableVocabularies(VocabularyCoreV1))
			if err != nil {
				t.Fatalf("FromJSON core vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got, EnableVocabularies(VocabularyCoreV1))
			if err != nil {
				t.Fatalf("ToJSON core vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestCoreVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularyCoreV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input, EnableVocabularies(VocabularyCoreV1)); err == nil {
				t.Fatal("FromJSON succeeded for invalid core vocabulary input")
			}
		})
	}
}

func TestCoreVocabularyRejectsInvalidRON(t *testing.T) {
	if _, err := ToJSON([]byte(`bad {#uid not-a-uuid}`), EnableVocabularies(VocabularyCoreV1)); err == nil {
		t.Fatal("ToJSON succeeded for invalid core vocabulary input")
	}
}
