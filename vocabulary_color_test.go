package ron

import (
	"slices"
	"testing"

	colorlib "github.com/SCKelemen/color"
)

func TestColorVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyColorV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readRONValueFixture(t, root, tc.ExpectedRON)

			got, err := FromJSON(input, EnableVocabularies(VocabularyColorV1))
			if err != nil {
				t.Fatalf("FromJSON color vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got, EnableVocabularies(VocabularyColorV1))
			if err != nil {
				t.Fatalf("ToJSON color vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestColorVocabularyParsesNativeValues(t *testing.T) {
	value, err := decodeJSON([]byte(`{"accent":{"#clr":["oklch",0.7,0.15,230]}}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	opts := optionState{}
	EnableVocabularies(VocabularyColorV1)(&opts)
	parsed, err := opts.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	colorValue, ok := objectValue(object, "accent").(Color)
	if !ok {
		t.Fatalf("accent type = %T %[1]v", objectValue(object, "accent"))
	}
	if colorValue.Space != ColorSpaceOKLCH || !slices.Equal(colorValue.Channels, []float64{0.7, 0.15, 230}) {
		t.Fatalf("accent = %#v", colorValue)
	}
	if colorValue.Value == nil {
		t.Fatal("accent library color is nil")
	}
}

func TestColorVocabularyRendersNativeValues(t *testing.T) {
	assertBytesEqual(t, []byte("{#clr [oklch 0.7 0.15 230]}"), []byte(renderScalar(NewOKLCHColor(0.7, 0.15, 230), true)))
	assertBytesEqual(t, []byte("{#clr [oklch 0.7 0.15 230]}"), []byte(renderScalar(colorlib.NewOKLCH(0.7, 0.15, 230, 1), true)))
}

func TestColorVocabularyRejectsInvalidPayloads(t *testing.T) {
	for _, input := range [][]byte{
		[]byte(`{"accent":{"#clr":["oklch",0.7,0.15]}}`),
		[]byte(`{"accent":{"#clr":["cmyk",0.7,0.15,230]}}`),
		[]byte(`{"accent":{"#clr":["oklch",0.7,0.15,"230"]}}`),
		[]byte(`{"accent":{"#clr":["rgba",0.7,0.15,0.2,2]}}`),
	} {
		if _, err := FromJSON(input, EnableVocabularies(VocabularyColorV1)); err == nil {
			t.Fatalf("FromJSON succeeded for invalid color vocabulary input: %s", input)
		}
	}
}
