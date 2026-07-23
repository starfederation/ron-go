package ron

import "testing"

func TestSpatialVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularySpatialV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readConformanceFile(t, root, tc.ExpectedRON)

			got, err := FromJSON(input)
			if err != nil {
				t.Fatalf("FromJSON spatial vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got)
			if err != nil {
				t.Fatalf("ToJSON spatial vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestSpatialVocabularyParsesNativeValues(t *testing.T) {
	value, err := decodeJSON([]byte(`{
		"bounds":{"#bx3":[[0,0,0],[10,10,10]]},
		"voxels":{"#vox":{"dimensions":3,"origin":{"#vN":[0,0,0]},"cellSize":{"#vN":[1,1,1]},"cells":[]}}
	}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	parsed, err := optionState{vocabularies: defaultVocabularies()}.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	if got, ok := objectValue(object, "bounds").(Box3); !ok || got.Max.X != 10 || got.Max.Y != 10 || got.Max.Z != 10 {
		t.Fatalf("bounds type = %T %[1]v", objectValue(object, "bounds"))
	}
	if got, ok := objectValue(object, "voxels").(VoxelSet); !ok || got.Dimensions != 3 || len(got.Cells) != 0 {
		t.Fatalf("voxels type = %T %[1]v", objectValue(object, "voxels"))
	}
}

func TestSpatialVocabularyRendersNativeValues(t *testing.T) {
	assertBytesEqual(t, []byte("{#bx2 [[0 0] [10 10]]}"), []byte(renderScalar(Box2{Min: Vector2{X: 0, Y: 0}, Max: Vector2{X: 10, Y: 10}}, true)))
	assertBytesEqual(t, []byte("{#ln2 [[0 0] [1 1]]}"), []byte(renderScalar(Line2{Start: Vector2{X: 0, Y: 0}, End: Vector2{X: 1, Y: 1}}, true)))
}
