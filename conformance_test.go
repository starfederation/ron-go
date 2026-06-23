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

type conformanceManifest struct {
	Version    int `json:"version"`
	Formatting struct {
		JSONPrefix                string                   `json:"jsonPrefix"`
		JSONIndent                string                   `json:"jsonIndent"`
		RONIndent                 string                   `json:"ronIndent"`
		PrettyRONTrailingNewline  bool                     `json:"prettyRONTrailingNewline"`
		ObjectKeyOrder            string                   `json:"objectKeyOrder"`
		CanonicalRON              string                   `json:"canonicalRON"`
		CanonicalRONHashAlgorithm string                   `json:"canonicalRONHashAlgorithm"`
		ExpectedPrettyOptions     conformanceFormatOptions `json:"expectedPrettyOptions"`
		ExpectedCompactOptions    conformanceFormatOptions `json:"expectedCompactOptions"`
	} `json:"formatting"`
	Valid []struct {
		Name                       string   `json:"name"`
		RONInputs                  []string `json:"ronInputs"`
		JSONInput                  string   `json:"jsonInput"`
		ExpectedPrettyJSON         string   `json:"expectedPrettyJSON"`
		ExpectedCompactJSON        string   `json:"expectedCompactJSON"`
		ExpectedPrettyRON          string   `json:"expectedPrettyRON"`
		ExpectedCompactRON         string   `json:"expectedCompactRON"`
		ExpectedCanonicalRONSHA256 string   `json:"expectedCanonicalRONSHA256"`
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
}

type conformanceFormatOptions struct {
	IsPretty    bool `json:"isPretty"`
	IsCanonical bool `json:"isCanonical"`
}

type rfc8785Manifest struct {
	Valid []struct {
		Name                        string `json:"name"`
		InputJSON                   string `json:"inputJSON"`
		ExpectedCanonicalJSON       string `json:"expectedCanonicalJSON"`
		ExpectedCanonicalUTF8Hex    string `json:"expectedCanonicalUTF8Hex"`
		ExpectedCanonicalJSONSHA256 string `json:"expectedCanonicalJSONSHA256"`
	} `json:"valid"`
	NumberSerialization string `json:"numberSerialization"`
	InvalidIJSON        []struct {
		Name      string `json:"name"`
		InputJSON string `json:"inputJSON"`
	} `json:"invalidIJSON"`
}

type rfc8785NumberCase struct {
	IEEE754Hex   string `json:"ieee754Hex"`
	ExpectedJSON string `json:"expectedJSON"`
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

			compactRON, err := FromJSON(jsonInput, IsPretty(false), IsCanonical(true))
			if err != nil {
				t.Fatalf("FromJSON compact canonical: %v", err)
			}
			assertBytesEqual(t, expectedCompactRON, compactRON)
			if tc.ExpectedCanonicalRONSHA256 != "" {
				gotHash := formatSHA256(compactRON)
				if gotHash != tc.ExpectedCanonicalRONSHA256 {
					t.Fatalf("canonical RON SHA-256 mismatch\nwant: %s\n got: %s", tc.ExpectedCanonicalRONSHA256, gotHash)
				}
			}
			compactRONJSON, err := ToJSON(compactRON)
			if err != nil {
				t.Fatalf("ToJSON generated compact RON: %v", err)
			}
			assertJSONEqual(t, jsonInput, compactRONJSON)
		})
	}
}

func TestConformanceJSONToRONRendering(t *testing.T) {
	root, manifest := loadConformanceManifest(t)
	for _, tc := range manifest.JSONToRONRendering {
		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.JSONInput)
			expected := readConformanceFile(t, root, tc.ExpectedRON)

			options := []Option{
				IsPretty(tc.Options.IsPretty),
				IsCanonical(tc.Options.IsCanonical),
			}
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
						for i, segment := range path {
							switch value := hook.Path[i].(type) {
							case string:
								matches = !segment.IsIndex && segment.Key == value
							case float64:
								matches = segment.IsIndex && segment.Index == int(value)
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

			got, err := FromJSON(input, options...)
			if err != nil {
				t.Fatalf("FromJSON rendering: %v", err)
			}
			assertBytesEqual(t, expected, got)
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

func TestRFC8785CanonicalJSONValid(t *testing.T) {
	root, manifest := loadRFC8785Manifest(t)
	for _, tc := range manifest.Valid {
		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expectedJSON := readConformanceFile(t, root, tc.ExpectedCanonicalJSON)
			expectedHex := readConformanceFile(t, root, tc.ExpectedCanonicalUTF8Hex)

			got, err := canonicalJSON(input)
			if err != nil {
				t.Fatalf("canonicalJSON: %v", err)
			}
			assertBytesEqual(t, expectedJSON, got)
			assertBytesEqual(t, bytes.TrimSpace(expectedHex), []byte(hex.EncodeToString(got)))

			gotHash := formatSHA256(got)
			if gotHash != tc.ExpectedCanonicalJSONSHA256 {
				t.Fatalf("canonical JSON SHA-256 mismatch\nwant: %s\n got: %s", tc.ExpectedCanonicalJSONSHA256, gotHash)
			}
		})
	}
}

func TestRFC8785NumberSerialization(t *testing.T) {
	root, manifest := loadRFC8785Manifest(t)
	body := readConformanceFile(t, root, manifest.NumberSerialization)

	var numbers struct {
		Finite               []rfc8785NumberCase `json:"finite"`
		RejectedNativeValues []rfc8785NumberCase `json:"rejectedNativeValues"`
	}
	if err := json.Unmarshal(body, &numbers); err != nil {
		t.Fatalf("unmarshal number serialization: %v", err)
	}

	for _, tc := range numbers.Finite {
		t.Run(tc.IEEE754Hex, func(t *testing.T) {
			value := parseFloat64Hex(t, tc.IEEE754Hex)
			got, err := appendRFC8785Number(nil, value)
			if err != nil {
				t.Fatalf("appendRFC8785Number: %v", err)
			}
			assertBytesEqual(t, []byte(tc.ExpectedJSON), got)
		})
	}

	for _, tc := range numbers.RejectedNativeValues {
		t.Run("reject/"+tc.IEEE754Hex, func(t *testing.T) {
			value := parseFloat64Hex(t, tc.IEEE754Hex)
			if _, err := appendRFC8785Number(nil, value); err == nil {
				t.Fatal("appendRFC8785Number accepted non-finite value")
			}
		})
	}
}

func TestRFC8785InvalidIJSON(t *testing.T) {
	root, manifest := loadRFC8785Manifest(t)
	for _, tc := range manifest.InvalidIJSON {
		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := canonicalJSON(input); err == nil {
				t.Fatal("canonicalJSON succeeded for invalid I-JSON")
			}
		})
	}
}

func TestRFC8785StringLessUsesUTF16Order(t *testing.T) {
	if !rfc8785StringLess("\U0001f600", "\ue000") {
		t.Fatal("rfc8785StringLess did not use UTF-16 code unit order")
	}
	if rfc8785StringLess("\ue000", "\U0001f600") {
		t.Fatal("rfc8785StringLess reversed UTF-16 code unit order")
	}
}

func TestRFC8785StringLessDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = rfc8785StringLess("created", "digest")
		_ = rfc8785StringLess("\U0001f600", "\ue000")
	})
	if allocs != 0 {
		t.Fatalf("rfc8785StringLess allocated %v times", allocs)
	}
}

func TestToJSONDuplicateKeysUseLastValue(t *testing.T) {
	got, err := ToJSON([]byte("item {name first name second count 1}"))
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	want := []byte(`{"item":{"count":1,"name":"second"}}`)
	assertBytesEqual(t, want, got)
}

func TestRONContainsVocabularyMarker(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
		want bool
	}{
		{
			name: "untagged query object",
			src:  []byte("query {filter {status active limit 20} sort [created desc]}"),
		},
		{
			name: "empty core tag",
			src:  []byte("entity {# 123}"),
			want: true,
		},
		{
			name: "utc tag",
			src:  []byte("created {#utc 2026-06-13T00:00:00Z}"),
			want: true,
		},
		{
			name: "sha256 tag",
			src:  []byte("digest {#sha256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855}"),
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ronContainsVocabularyMarker(tc.src); got != tc.want {
				t.Fatalf("ronContainsVocabularyMarker() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestToJSONUntaggedStillRejectsUnsupportedVocabulary(t *testing.T) {
	if _, err := ToJSON([]byte("item {name alpha count 1}"), EnableVocabularies("https://example.com/vocab/unknown/v1")); err == nil {
		t.Fatal("ToJSON accepted unsupported vocabulary")
	}
}

func TestFromJSONCompactDuplicateKeysUseLastValue(t *testing.T) {
	got, err := FromJSONCompact([]byte(`{"item":{"name":"first","name":"second","count":1}}`))
	if err != nil {
		t.Fatalf("FromJSONCompact: %v", err)
	}

	want := []byte("item{count 1 name second}")
	assertBytesEqual(t, want, got)
}

func TestRONBufferReuse(t *testing.T) {
	var buf bytes.Buffer
	pretty, err := FromJSONInto(&buf, []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("FromJSONInto: %v", err)
	}
	assertBytesEqual(t, []byte("a 1\n"), pretty)

	buf.Reset()
	compact, err := FromJSONCompactInto(&buf, []byte(`{"b":2}`))
	if err != nil {
		t.Fatalf("FromJSONCompactInto: %v", err)
	}
	assertBytesEqual(t, []byte("b 2"), compact)
}

func loadConformanceManifest(t *testing.T) (string, conformanceManifest) {
	t.Helper()
	root := testdataSubdir(t, "conformance")
	body := readFile(t, filepath.Join(root, "manifest.json"))

	var manifest conformanceManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return root, manifest
}

func loadRFC8785Manifest(t *testing.T) (string, rfc8785Manifest) {
	t.Helper()
	root := testdataSubdir(t, "rfc8785")
	body := readFile(t, filepath.Join(root, "manifest.json"))

	var manifest rfc8785Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("unmarshal RFC 8785 manifest: %v", err)
	}
	return root, manifest
}

func testdataSubdir(t *testing.T, subdir string) string {
	t.Helper()
	if testdataRoot := os.Getenv("RON_TESTDATA_DIR"); testdataRoot != "" {
		return filepath.Join(testdataRoot, subdir)
	}

	root := filepath.Join("testdata", subdir)
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err != nil {
		t.Skip("RON testdata unavailable; set RON_TESTDATA_DIR or run nix flake check")
	}
	return root
}

func formatSHA256(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

func parseFloat64Hex(t *testing.T, value string) float64 {
	t.Helper()
	bits, err := strconv.ParseUint(value, 16, 64)
	if err != nil {
		t.Fatalf("parse float64 hex %q: %v", value, err)
	}
	return math.Float64frombits(bits)
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
	value, err := decodeJSON(body, nil)
	if err != nil {
		t.Fatalf("decode JSON %q: %v", body, err)
	}
	return normalizeJSONForTest(value)
}

func normalizeJSONForTest(value any) any {
	switch value := value.(type) {
	case orderedObject:
		object := make(map[string]any, len(value.Members))
		for _, member := range value.Members {
			object[member.Key] = normalizeJSONForTest(member.Value)
		}
		return object
	case []any:
		array := make([]any, len(value))
		for i, item := range value {
			array[i] = normalizeJSONForTest(item)
		}
		return array
	default:
		return value
	}
}

func TestToJSONCanonicalOptionControlsObjectOrder(t *testing.T) {
	nonCanonical, err := ToJSON([]byte("{b 1 a 2}"), IsCanonical(false))
	if err != nil {
		t.Fatalf("ToJSON non-canonical: %v", err)
	}
	assertBytesEqual(t, []byte(`{"b":1,"a":2}`), nonCanonical)

	canonical, err := ToJSON([]byte("{b 1 a 2}"), IsCanonical(true))
	if err != nil {
		t.Fatalf("ToJSON canonical: %v", err)
	}
	assertBytesEqual(t, []byte(`{"a":2,"b":1}`), canonical)
}

func TestFromJSONOptionsControlPrettyAndCanonicalRON(t *testing.T) {
	nonCanonicalCompact, err := FromJSON([]byte(`{"b":1,"a":2}`), IsPretty(false), IsCanonical(false))
	if err != nil {
		t.Fatalf("FromJSON compact non-canonical: %v", err)
	}
	assertBytesEqual(t, []byte("b 1 a 2"), nonCanonicalCompact)

	canonicalCompact, err := FromJSON([]byte(`{"b":1,"a":2}`), IsPretty(false), IsCanonical(true))
	if err != nil {
		t.Fatalf("FromJSON compact canonical: %v", err)
	}
	assertBytesEqual(t, []byte("a 2 b 1"), canonicalCompact)

	nonCanonicalPretty, err := FromJSON([]byte(`{"b":1,"a":2}`), IsPretty(true), IsCanonical(false))
	if err != nil {
		t.Fatalf("FromJSON pretty non-canonical: %v", err)
	}
	assertBytesEqual(t, []byte("b 1\na 2\n"), nonCanonicalPretty)
}

func TestNonCanonicalDuplicateKeysMoveSurvivorToLastPosition(t *testing.T) {
	jsonBody, err := ToJSON([]byte("{a 1 b 2 a 3}"), IsCanonical(false))
	if err != nil {
		t.Fatalf("ToJSON non-canonical duplicates: %v", err)
	}
	assertBytesEqual(t, []byte(`{"b":2,"a":3}`), jsonBody)

	ronBody, err := FromJSON([]byte(`{"a":1,"b":2,"a":3}`), IsPretty(false), IsCanonical(false))
	if err != nil {
		t.Fatalf("FromJSON non-canonical duplicates: %v", err)
	}
	assertBytesEqual(t, []byte("b 2 a 3"), ronBody)
}
