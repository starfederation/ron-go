package ron

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type conformanceManifest struct {
	Version     int                    `json:"version"`
	Formatting  conformanceFormatting  `json:"formatting"`
	Valid       []conformanceValidCase `json:"valid"`
	InvalidRON  []string               `json:"invalidRON"`
	InvalidJSON []string               `json:"invalidJSON"`
}

type conformanceFormatting struct {
	JSONPrefix               string `json:"jsonPrefix"`
	JSONIndent               string `json:"jsonIndent"`
	RONIndent                string `json:"ronIndent"`
	PrettyRONTrailingNewline bool   `json:"prettyRONTrailingNewline"`
	ObjectKeyOrder           string `json:"objectKeyOrder"`
}

type conformanceValidCase struct {
	Name                string   `json:"name"`
	RONInputs           []string `json:"ronInputs"`
	JSONInput           string   `json:"jsonInput"`
	ExpectedPrettyJSON  string   `json:"expectedPrettyJSON"`
	ExpectedCompactJSON string   `json:"expectedCompactJSON"`
	ExpectedPrettyRON   string   `json:"expectedPrettyRON"`
	ExpectedCompactRON  string   `json:"expectedCompactRON"`
}

func TestConformanceValid(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	if manifest.Formatting.JSONPrefix != "" || manifest.Formatting.JSONIndent == "" || manifest.Formatting.RONIndent == "" {
		t.Fatalf("unsupported conformance formatting: %+v", manifest.Formatting)
	}

	for _, tc := range manifest.Valid {
		t.Run(tc.Name, func(t *testing.T) {
			jsonInput := readConformanceFile(t, root, tc.JSONInput)
			expectedCompactJSON := readConformanceFile(t, root, tc.ExpectedCompactJSON)
			expectedPrettyJSON := readConformanceFile(t, root, tc.ExpectedPrettyJSON)
			expectedPrettyRON := readConformanceFile(t, root, tc.ExpectedPrettyRON)
			expectedCompactRON := readConformanceFile(t, root, tc.ExpectedCompactRON)

			for _, ronInput := range tc.RONInputs {
				t.Run("ron_to_json/"+filepath.Base(ronInput), func(t *testing.T) {
					ronSource := readConformanceFile(t, root, ronInput)

					compactJSON, err := ToJSON(ronSource)
					if err != nil {
						t.Fatalf("ToJSON compact: %v", err)
					}
					assertBytesEqual(t, expectedCompactJSON, compactJSON)
					assertJSONEqual(t, jsonInput, compactJSON)

					prettyJSON, err := ToJSON(
						ronSource,
						PrettyJSON(manifest.Formatting.JSONPrefix, manifest.Formatting.JSONIndent),
					)
					if err != nil {
						t.Fatalf("ToJSON pretty: %v", err)
					}
					assertBytesEqual(t, expectedPrettyJSON, prettyJSON)
					assertJSONEqual(t, jsonInput, prettyJSON)
				})
			}

			prettyRON, err := FromJSON(jsonInput, Indent(manifest.Formatting.RONIndent))
			if err != nil {
				t.Fatalf("FromJSON pretty: %v", err)
			}
			assertBytesEqual(t, expectedPrettyRON, prettyRON)
			prettyRONJSON, err := ToJSON(prettyRON)
			if err != nil {
				t.Fatalf("ToJSON generated pretty RON: %v", err)
			}
			assertJSONEqual(t, jsonInput, prettyRONJSON)

			compactRON, err := FromJSONCompact(jsonInput)
			if err != nil {
				t.Fatalf("FromJSONCompact: %v", err)
			}
			assertBytesEqual(t, expectedCompactRON, compactRON)
			compactRONJSON, err := ToJSON(compactRON)
			if err != nil {
				t.Fatalf("ToJSON generated compact RON: %v", err)
			}
			assertJSONEqual(t, jsonInput, compactRONJSON)
		})
	}
}

func TestConformanceInvalidRON(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, path := range manifest.InvalidRON {
		t.Run(path, func(t *testing.T) {
			if _, err := ToJSON(readConformanceFile(t, root, path)); err == nil {
				t.Fatal("ToJSON succeeded for invalid RON")
			}
		})
	}
}

func TestConformanceInvalidJSON(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, path := range manifest.InvalidJSON {
		t.Run(path, func(t *testing.T) {
			body := readConformanceFile(t, root, path)
			if _, err := FromJSON(body); err == nil {
				t.Fatal("FromJSON succeeded for invalid JSON")
			}
			if _, err := FromJSONCompact(body); err == nil {
				t.Fatal("FromJSONCompact succeeded for invalid JSON")
			}
		})
	}
}

func loadConformanceManifest(t *testing.T) (string, conformanceManifest) {
	t.Helper()
	root := conformanceRoot(t)
	body := readFile(t, filepath.Join(root, "manifest.json"))

	var manifest conformanceManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return root, manifest
}

func conformanceRoot(t *testing.T) string {
	t.Helper()
	if root := os.Getenv("RON_TESTDATA_DIR"); root != "" {
		return filepath.Join(root, "conformance")
	}

	root := filepath.Join("testdata", "conformance")
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err == nil {
		return root
	}

	t.Skip("RON conformance corpus unavailable; set RON_TESTDATA_DIR or run nix flake check")
	return ""
}

func readConformanceFile(t *testing.T, root, path string) []byte {
	t.Helper()
	return readFile(t, filepath.Join(root, filepath.FromSlash(path)))
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return body
}

func assertBytesEqual(t *testing.T, want, got []byte) {
	t.Helper()
	if !bytes.Equal(want, got) {
		t.Fatalf("bytes differ\nwant: %q\n got: %q", want, got)
	}
}

func assertJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	wantValue := decodeJSONForTest(t, want)
	gotValue := decodeJSONForTest(t, got)
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Fatalf("JSON differs\nwant: %s\n got: %s", want, got)
	}
}

func decodeJSONForTest(t *testing.T, body []byte) any {
	t.Helper()
	value, err := decodeJSON(body)
	if err != nil {
		t.Fatalf("decode JSON %q: %v", body, err)
	}
	return value
}
