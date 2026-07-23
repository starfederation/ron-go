package ron

import (
	"testing"
	"time"
)

func TestTimeVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyTimeV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readRONValueFixture(t, root, tc.ExpectedRON)

			got, err := FromJSON(input, EnableVocabularies(VocabularyTimeV1))
			if err != nil {
				t.Fatalf("FromJSON time vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got, EnableVocabularies(VocabularyTimeV1))
			if err != nil {
				t.Fatalf("ToJSON time vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestTimeVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularyTimeV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input, EnableVocabularies(VocabularyTimeV1)); err == nil {
				t.Fatal("FromJSON succeeded for invalid time vocabulary input")
			}
		})
	}
}

func TestTimeVocabularyParsesNativeValues(t *testing.T) {
	value, err := decodeJSON([]byte(`{
		"created":{"#utc":"2026-06-13T00:00:00.123456789Z"},
		"elapsed":{"#dur":"P2DT3H4M5.5S"}
	}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	opts := optionState{}
	EnableVocabularies(VocabularyTimeV1)(&opts)
	parsed, err := opts.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	instant, ok := objectValue(object, "created").(time.Time)
	if !ok || instant.Format(time.RFC3339Nano) != "2026-06-13T00:00:00.123456789Z" {
		t.Fatalf("created type = %T %[1]v", objectValue(object, "created"))
	}
	duration, ok := objectValue(object, "elapsed").(time.Duration)
	if !ok || duration != 51*time.Hour+4*time.Minute+5*time.Second+500*time.Millisecond {
		t.Fatalf("elapsed type = %T %[1]v", objectValue(object, "elapsed"))
	}
}

func TestTimeVocabularyRendersNativeValues(t *testing.T) {
	instant := time.Date(2026, 6, 13, 0, 0, 0, 100, time.UTC)
	assertBytesEqual(t, []byte("{#utc 2026-06-13T00:00:00.0000001Z}"), []byte(renderScalar(instant, true)))
	assertBytesEqual(t, []byte("{#dur P2DT3H4M5.5S}"), []byte(renderScalar(51*time.Hour+4*time.Minute+5*time.Second+500*time.Millisecond, true)))
}
