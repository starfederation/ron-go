package ron

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

type conformanceFormatOptions struct {
	Mode OutputMode `json:"mode"`
}

type conformanceManifest struct {
	Valid []struct {
		Name                      string   `json:"name"`
		RONInputs                 []string `json:"ronInputs"`
		JSONInput                 string   `json:"jsonInput"`
		ExpectedPrettyJSON        string   `json:"expectedPrettyJSON"`
		ExpectedCompactJSON       string   `json:"expectedCompactJSON"`
		ExpectedCanonicalJSON     string   `json:"expectedCanonicalJSON"`
		ExpectedPrettyRON         string   `json:"expectedPrettyRON"`
		ExpectedCompactRON        string   `json:"expectedCompactRON"`
		ExpectedCanonicalRON      string   `json:"expectedCanonicalRON"`
		ExpectedCanonicalJSONHash string   `json:"expectedCanonicalJSONSHA256"`
		ExpectedCanonicalRONHash  string   `json:"expectedCanonicalRONSHA256"`
	} `json:"valid"`
	InvalidRON         []string `json:"invalidRON"`
	InvalidJSON        []string `json:"invalidJSON"`
	JSONToRONRendering []struct {
		Name            string                   `json:"name"`
		JSONInput       string                   `json:"jsonInput"`
		Options         conformanceFormatOptions `json:"options"`
		TypedValueHooks []struct {
			Path        []any           `json:"path"`
			ReplaceWith json.RawMessage `json:"replaceWith"`
		} `json:"typedValueHooks"`
		ExpectedRON string `json:"expectedRON"`
	} `json:"jsonToRONRendering"`
	CanonicalRON struct {
		ValidRON []struct {
			Name     string `json:"name"`
			Input    string `json:"inputRON"`
			Expected string `json:"expectedCanonicalRON"`
			Hash     string `json:"expectedCanonicalRONSHA256"`
		} `json:"validRON"`
		InvalidRON []string `json:"invalidRON"`
	} `json:"canonicalRON"`
}

type rfc8785Manifest struct {
	Valid []struct {
		Name     string `json:"name"`
		Input    string `json:"inputJSON"`
		JSON     string `json:"expectedCanonicalJSON"`
		RON      string `json:"expectedCanonicalRON"`
		Hex      string `json:"expectedCanonicalUTF8Hex"`
		JSONHash string `json:"expectedCanonicalJSONSHA256"`
		RONHash  string `json:"expectedCanonicalRONSHA256"`
	} `json:"valid"`
	NumberSerialization string `json:"numberSerialization"`
	InvalidIJSON        []struct {
		Name  string `json:"name"`
		Input string `json:"inputJSON"`
	} `json:"invalidIJSON"`
}

type rfc8785NumberCase struct {
	IEEE754Hex           string `json:"ieee754Hex"`
	ExpectedJSON         string `json:"expectedJSON"`
	ExpectedCanonicalRON string `json:"expectedCanonicalRON"`
}

func TestConformanceValid(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, tc := range manifest.Valid {
		t.Run(tc.Name, func(t *testing.T) {
			inputJSON := readConformanceFile(t, root, tc.JSONInput)
			for _, inputRON := range tc.RONInputs {
				ronSource := readConformanceFile(t, root, inputRON)
				for _, check := range []struct {
					mode     OutputMode
					expected string
				}{
					{Pretty, tc.ExpectedPrettyJSON},
					{Compact, tc.ExpectedCompactJSON},
				} {
					got, err := ToJSON(ronSource, Mode(check.mode))
					if err != nil {
						t.Fatalf("ToJSON %s: %v", check.mode, err)
					}
					assertBytesEqual(t, readConformanceFile(t, root, check.expected), got)
				}
				if inputRON == "valid/basic/escapes/input.ron" {
					// This source fixture has duplicate decoded names. Canonical RON rejects it.
					if _, err := ToJSON(ronSource, Mode(Canonical)); err == nil {
						t.Fatal("canonical RON accepted duplicate names")
					}
					continue
				}
				got, err := ToJSON(ronSource, Mode(Canonical))
				if err != nil {
					t.Fatalf("ToJSON canonical: %v", err)
				}
				assertBytesEqual(t, readConformanceFile(t, root, tc.ExpectedCanonicalJSON), got)
				if formatSHA256(got) != tc.ExpectedCanonicalJSONHash {
					t.Fatal("canonical JSON hash mismatch")
				}
			}
			for _, check := range []struct {
				mode           OutputMode
				expected, hash string
			}{
				{Pretty, tc.ExpectedPrettyRON, ""}, {Compact, tc.ExpectedCompactRON, ""}, {Canonical, tc.ExpectedCanonicalRON, tc.ExpectedCanonicalRONHash},
			} {
				got, err := FromJSON(inputJSON, Mode(check.mode))
				if err != nil {
					t.Fatalf("FromJSON %s: %v", check.mode, err)
				}
				assertBytesEqual(t, readConformanceFile(t, root, check.expected), got)
				if check.hash != "" && formatSHA256(got) != check.hash {
					t.Fatalf("canonical RON hash mismatch")
				}
			}
		})
	}
}

func TestConformanceJSONToRONRendering(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, tc := range manifest.JSONToRONRendering {
		t.Run(tc.Name, func(t *testing.T) {
			options := []Option{Mode(tc.Options.Mode)}
			if len(tc.TypedValueHooks) > 0 {
				replacements := make([]any, len(tc.TypedValueHooks))
				for i, hook := range tc.TypedValueHooks {
					value, err := decodeJSON(hook.ReplaceWith, nil)
					if err != nil {
						t.Fatalf("decode hook replacement: %v", err)
					}
					replacements[i] = value
				}
				options = append(options, MapJSONValues(func(path []JSONPathSegment, value any) (any, bool) {
					for i, hook := range tc.TypedValueHooks {
						if len(path) != len(hook.Path) {
							continue
						}
						matches := true
						for index, segment := range path {
							switch expected := hook.Path[index].(type) {
							case string:
								matches = !segment.IsIndex && segment.Key == expected
							case float64:
								matches = segment.IsIndex && segment.Index == int(expected)
							default:
								matches = false
							}
							if !matches {
								break
							}
						}
						if matches {
							return replacements[i], true
						}
					}
					return nil, false
				}))
			}
			got, err := FromJSON(readConformanceFile(t, root, tc.JSONInput), options...)
			if err != nil {
				t.Fatalf("FromJSON rendering: %v", err)
			}
			assertBytesEqual(t, readConformanceFile(t, root, tc.ExpectedRON), got)
		})
	}
}

func TestCanonicalRONBoundaries(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, tc := range manifest.CanonicalRON.ValidRON {
		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.Input)
			canonicalJSON, err := ToJSON(input, Mode(Canonical))
			if err != nil {
				t.Fatal(err)
			}
			got, err := FromJSON(canonicalJSON, Mode(Canonical))
			if err != nil {
				t.Fatalf("canonical RON: %v", err)
			}
			assertBytesEqual(t, readConformanceFile(t, root, tc.Expected), got)
			if formatSHA256(got) != tc.Hash {
				t.Fatal("canonical RON hash mismatch")
			}
		})
	}
	for _, path := range manifest.CanonicalRON.InvalidRON {
		t.Run(path, func(t *testing.T) {
			if _, err := ToJSON(readConformanceFile(t, root, path), Mode(Canonical)); err == nil {
				t.Fatal("canonical RON succeeded")
			}
		})
	}
}

func TestConformanceInvalidInput(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, path := range manifest.InvalidRON {
		if _, err := ToJSON(readConformanceFile(t, root, path), Mode(Pretty)); err == nil {
			t.Fatalf("invalid RON accepted: %s", path)
		}
	}
	for _, path := range manifest.InvalidJSON {
		if _, err := FromJSON(readConformanceFile(t, root, path), Mode(Pretty)); err == nil {
			t.Fatalf("invalid JSON accepted: %s", path)
		}
	}
}

func TestRFC8785Conformance(t *testing.T) {
	root, manifest := loadRFC8785Manifest(t)
	for _, tc := range manifest.Valid {
		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.Input)
			jsonBody, err := canonicalJSON(input)
			if err != nil {
				t.Fatal(err)
			}
			assertBytesEqual(t, readConformanceFile(t, root, tc.JSON), jsonBody)
			assertBytesEqual(t, bytes.TrimSpace(readConformanceFile(t, root, tc.Hex)), []byte(hex.EncodeToString(jsonBody)))
			if formatSHA256(jsonBody) != tc.JSONHash {
				t.Fatal("canonical JSON hash mismatch")
			}
			ronBody, err := FromJSON(input, Mode(Canonical))
			if err != nil {
				t.Fatal(err)
			}
			assertBytesEqual(t, readConformanceFile(t, root, tc.RON), ronBody)
			if formatSHA256(ronBody) != tc.RONHash {
				t.Fatal("canonical RON hash mismatch")
			}
		})
	}
	for _, tc := range manifest.InvalidIJSON {
		if _, err := FromJSON(readConformanceFile(t, root, tc.Input), Mode(Canonical)); err == nil {
			t.Fatalf("invalid I-JSON accepted: %s", tc.Name)
		}
	}
}

func TestRFC8785NumberSerialization(t *testing.T) {
	root, manifest := loadRFC8785Manifest(t)
	var numbers struct {
		Finite               []rfc8785NumberCase `json:"finite"`
		RejectedNativeValues []rfc8785NumberCase `json:"rejectedNativeValues"`
	}
	if err := json.Unmarshal(readConformanceFile(t, root, manifest.NumberSerialization), &numbers); err != nil {
		t.Fatal(err)
	}
	for _, tc := range numbers.Finite {
		value := parseFloat64Hex(t, tc.IEEE754Hex)
		got, err := appendRFC8785Number(nil, value)
		if err != nil {
			t.Fatal(err)
		}
		assertBytesEqual(t, []byte(tc.ExpectedJSON), got)
	}
	for _, tc := range numbers.RejectedNativeValues {
		if _, err := appendRFC8785Number(nil, parseFloat64Hex(t, tc.IEEE754Hex)); err == nil {
			t.Fatal("non-finite number accepted")
		}
	}
}

func loadConformanceManifest(t *testing.T) (string, conformanceManifest) {
	t.Helper()
	root := testdataSubdir(t, "conformance")
	var manifest conformanceManifest
	if err := json.Unmarshal(readFile(t, filepath.Join(root, "manifest.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	return root, manifest
}
func loadRFC8785Manifest(t *testing.T) (string, rfc8785Manifest) {
	t.Helper()
	root := testdataSubdir(t, "rfc8785")
	var manifest rfc8785Manifest
	if err := json.Unmarshal(readFile(t, filepath.Join(root, "manifest.json")), &manifest); err != nil {
		t.Fatal(err)
	}
	return root, manifest
}
func testdataSubdir(t *testing.T, subdir string) string {
	t.Helper()
	root := os.Getenv("RON_TESTDATA_DIR")
	if root == "" {
		root = filepath.Join("..", "ron", "testdata")
	}
	path := filepath.Join(root, subdir)
	manifestPath := filepath.Join(path, "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("RON %s fixtures unavailable: %v", subdir, err)
	}
	if subdir == "conformance" && !bytes.Contains(body, []byte(`"defaultMode"`)) {
		t.Fatal("conformance manifest is stale: update it for the revised Mode API")
	}
	if subdir == "rfc8785" && !bytes.Contains(body, []byte(`"expectedCanonicalRON"`)) {
		t.Fatal("RFC 8785 manifest is stale: add expectedCanonicalRON fixtures")
	}
	return path
}
func readConformanceFile(t *testing.T, root, path string) []byte {
	t.Helper()
	return readFile(t, filepath.Join(root, filepath.FromSlash(path)))
}
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func formatSHA256(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}
func parseFloat64Hex(t *testing.T, value string) float64 {
	t.Helper()
	bits, err := strconv.ParseUint(value, 16, 64)
	if err != nil {
		t.Fatal(err)
	}
	return math.Float64frombits(bits)
}
func readRONValueFixture(t *testing.T, root, path string) []byte {
	t.Helper()
	return readConformanceFile(t, root, path)
}

func assertJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	var wantValue any
	var gotValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Fatalf("JSON differs\nwant: %s\n got: %s", want, got)
	}
}

func assertBytesEqual(t *testing.T, want, got []byte) {
	t.Helper()
	if !bytes.Equal(want, got) {
		t.Fatalf("bytes differ\nwant: %q\n got: %q", want, got)
	}
}
