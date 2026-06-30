package ron

import "testing"

func TestSetVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularySetV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readConformanceFile(t, root, tc.ExpectedRON)

			got, err := FromJSON(input)
			if err != nil {
				t.Fatalf("FromJSON set vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got)
			if err != nil {
				t.Fatalf("ToJSON set vocabulary: %v", err)
			}
			roundTrip, err := FromJSON(jsonBody)
			if err != nil {
				t.Fatalf("FromJSON set vocabulary round trip: %v", err)
			}
			assertBytesEqual(t, expected, roundTrip)
		})
	}
}

func TestSetVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularySetV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input); err == nil {
				t.Fatal("FromJSON succeeded for invalid set vocabulary input")
			}
		})
	}
}

func TestSetVocabularyParsesNativeValues(t *testing.T) {
	value, err := decodeJSON([]byte(`{
		"roles":{"#set":["writer","admin","reader","admin"]},
		"bits":{"#bits":[10,[3,5],1,3,4,[5,7],4294967295]}
	}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	parsed, err := optionState{vocabularies: defaultVocabularies()}.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	roles, ok := objectValue(object, "roles").(Set)
	if !ok {
		t.Fatalf("roles type = %T", objectValue(object, "roles"))
	}
	assertBytesEqual(t, []byte("{#set [admin reader writer]}"), []byte(renderScalar(roles, true)))

	bits, ok := objectValue(object, "bits").(Uint32BitSet)
	if !ok {
		t.Fatalf("bits type = %T", objectValue(object, "bits"))
	}
	assertBytesEqual(t, []byte("{#bits [1 [3 7] 10 4294967295]}"), []byte(renderScalar(bits, true)))
}

func TestSetVocabularyRendersNativeValues(t *testing.T) {
	set := Set{"writer", "admin", "reader", "admin", 10}
	assertBytesEqual(t, []byte("{#set [admin reader writer 10]}"), []byte(renderScalar(set, true)))

	bits := Uint32BitSet{
		{First: 10, Last: 10},
		{First: 3, Last: 5},
		{First: 1, Last: 1},
		{First: 5, Last: 7},
	}
	assertBytesEqual(t, []byte("{#bits [1 [3 7] 10]}"), []byte(renderScalar(bits, true)))
}
