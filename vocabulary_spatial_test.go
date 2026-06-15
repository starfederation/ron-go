package ron

import (
	"testing"

	"github.com/paulmach/orb"
)

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
		"home":{"#lla":[-73.9857,40.7484,381]},
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

	if got, ok := objectValue(object, "home").(LngLatAlt); !ok || got.Point != (orb.Point{-73.9857, 40.7484}) || got.Altitude != 381 {
		t.Fatalf("home type = %T %[1]v", objectValue(object, "home"))
	}
	if got, ok := objectValue(object, "bounds").(Box3); !ok || got.Max.X != 10 || got.Max.Y != 10 || got.Max.Z != 10 {
		t.Fatalf("bounds type = %T %[1]v", objectValue(object, "bounds"))
	}
	if got, ok := objectValue(object, "voxels").(VoxelSet); !ok || got.Dimensions != 3 || len(got.Cells) != 0 {
		t.Fatalf("voxels type = %T %[1]v", objectValue(object, "voxels"))
	}
}

func TestSpatialVocabularyRendersNativeValues(t *testing.T) {
	assertBytesEqual(t, []byte("{#lla [-73.9857 40.7484 381]}"), []byte(renderScalar(LngLatAlt{Point: orb.Point{-73.9857, 40.7484}, Altitude: 381}, true)))
	assertBytesEqual(t, []byte("{#bx2 [[0 0] [10 10]]}"), []byte(renderScalar(Box2{Min: orb.Point{0, 0}, Max: orb.Point{10, 10}}, true)))
	assertBytesEqual(t, []byte("{#ln2 [[0 0] [1 1]]}"), []byte(renderScalar(Line2{Line: orb.LineString{{0, 0}, {1, 1}}}, true)))
}
