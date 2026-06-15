package ron

import "testing"

func TestMathVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyMathV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readConformanceFile(t, root, tc.ExpectedRON)

			got, err := FromJSON(input)
			if err != nil {
				t.Fatalf("FromJSON math vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got)
			if err != nil {
				t.Fatalf("ToJSON math vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestMathVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularyMathV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input); err == nil {
				t.Fatal("FromJSON succeeded for invalid math vocabulary input")
			}
		})
	}
}

func TestMathVocabularyParsesNativeValues(t *testing.T) {
	value, err := decodeJSON([]byte(`{
		"answer":{"#i64":"-42"},
		"count":{"#u64":"300"},
		"score":{"#f64":42.5},
		"cell":{"#iv3":[1,2,3]},
		"position":{"#f3v":[1.5,2.5,3.5]},
		"rotation":{"#qat":[0,0,0,1]},
		"euler":{"#eul":[0,0,0,"XYZ"]},
		"identity4":{"#m4x":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}
	}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	parsed, err := optionState{vocabularies: defaultVocabularies()}.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	if got, ok := objectValue(object, "answer").(Int64); !ok || got != -42 {
		t.Fatalf("answer type = %T %[1]v", objectValue(object, "answer"))
	}
	if got, ok := objectValue(object, "count").(Uint64); !ok || got != 300 {
		t.Fatalf("count type = %T %[1]v", objectValue(object, "count"))
	}
	if got, ok := objectValue(object, "score").(Float64); !ok || got != 42.5 {
		t.Fatalf("score type = %T %[1]v", objectValue(object, "score"))
	}
	if got, ok := objectValue(object, "cell").(IntVector3); !ok || got != (IntVector3{1, 2, 3}) {
		t.Fatalf("cell type = %T %[1]v", objectValue(object, "cell"))
	}
	if got, ok := objectValue(object, "position").(Vector3); !ok || got.X != 1.5 || got.Y != 2.5 || got.Z != 3.5 {
		t.Fatalf("position type = %T %[1]v", objectValue(object, "position"))
	}
	if got, ok := objectValue(object, "rotation").(Quaternion); !ok || got.W != 1 {
		t.Fatalf("rotation type = %T %[1]v", objectValue(object, "rotation"))
	}
	if got, ok := objectValue(object, "euler").(Euler); !ok || got.Order != EulerOrderXYZ {
		t.Fatalf("euler type = %T %[1]v", objectValue(object, "euler"))
	}
	if _, ok := objectValue(object, "identity4").(Matrix4); !ok {
		t.Fatalf("identity4 type = %T %[1]v", objectValue(object, "identity4"))
	}
}

func TestMathVocabularyRendersNativeValues(t *testing.T) {
	assertBytesEqual(t, []byte("{#i64 '-42'}"), []byte(renderScalar(Int64(-42), true)))
	assertBytesEqual(t, []byte("{#u64 '300'}"), []byte(renderScalar(Uint64(300), true)))
	assertBytesEqual(t, []byte("{#f64 42.5}"), []byte(renderScalar(Float64(42.5), true)))
	assertBytesEqual(t, []byte("{#iv3 [1 2 3]}"), []byte(renderScalar(IntVector3{1, 2, 3}, true)))
	assertBytesEqual(t, []byte("{#f3v [1.5 2.5 3.5]}"), []byte(renderScalar(Vector3{X: 1.5, Y: 2.5, Z: 3.5}, true)))
	assertBytesEqual(t, []byte("{#qat [0 0 0 1]}"), []byte(renderScalar(Quaternion{W: 1}, true)))
	assertBytesEqual(t, []byte("{#eul [0 0 0 XYZ]}"), []byte(renderScalar(Euler{Order: EulerOrderXYZ}, true)))
}
