package ron

import "testing"

func TestGeoVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyGeoV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readRONValueFixture(t, root, tc.ExpectedRON)

			got, err := FromJSON(input)
			if err != nil {
				t.Fatalf("FromJSON geo vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got)
			if err != nil {
				t.Fatalf("ToJSON geo vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestGeoVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularyGeoV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input); err == nil {
				t.Fatal("FromJSON succeeded for invalid geo vocabulary input")
			}
		})
	}
}

func TestGeoVocabularyPointPositionsRoundTrip(t *testing.T) {
	for _, input := range []string{
		`{"#geo":{"type":"Point","coordinates":[-73.9857,40.7484]}}`,
		`{"#geo":{"type":"Point","coordinates":[-73.9857,40.7484,381]}}`,
	} {
		ronBody, err := FromJSON([]byte(input))
		if err != nil {
			t.Fatalf("FromJSON(%s): %v", input, err)
		}

		jsonBody, err := ToJSON(ronBody)
		if err != nil {
			t.Fatalf("ToJSON(%s): %v", ronBody, err)
		}
		assertJSONEqual(t, []byte(input), jsonBody)
	}
}

func TestGeoVocabularyParsesNativeValue(t *testing.T) {
	value, err := decodeJSON([]byte(`{"point":{"#geo":{"type":"Point","coordinates":[-73.9857,40.7484]}}}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	parsed, err := optionState{vocabularies: defaultVocabularies()}.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)
	if got, ok := objectValue(object, "point").(GeoJSON); !ok || got.Data == nil {
		t.Fatalf("point type = %T %[1]v", objectValue(object, "point"))
	}
}
