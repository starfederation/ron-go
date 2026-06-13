package ron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/zeebo/xxh3"
)

type conformanceManifest struct {
	Version     int                    `json:"version"`
	Formatting  conformanceFormatting  `json:"formatting"`
	Valid       []conformanceValidCase `json:"valid"`
	InvalidRON  []string               `json:"invalidRON"`
	InvalidJSON []string               `json:"invalidJSON"`
}

type conformanceFormatting struct {
	JSONPrefix                string                   `json:"jsonPrefix"`
	JSONIndent                string                   `json:"jsonIndent"`
	RONIndent                 string                   `json:"ronIndent"`
	PrettyRONTrailingNewline  bool                     `json:"prettyRONTrailingNewline"`
	ObjectKeyOrder            string                   `json:"objectKeyOrder"`
	CanonicalRON              string                   `json:"canonicalRON"`
	CanonicalRONHashAlgorithm string                   `json:"canonicalRONHashAlgorithm"`
	ExpectedPrettyOptions     conformanceFormatOptions `json:"expectedPrettyOptions"`
	ExpectedCompactOptions    conformanceFormatOptions `json:"expectedCompactOptions"`
}

type conformanceFormatOptions struct {
	IsPretty    bool `json:"isPretty"`
	IsCanonical bool `json:"isCanonical"`
}

type conformanceValidCase struct {
	Name                     string   `json:"name"`
	RONInputs                []string `json:"ronInputs"`
	JSONInput                string   `json:"jsonInput"`
	ExpectedPrettyJSON       string   `json:"expectedPrettyJSON"`
	ExpectedCompactJSON      string   `json:"expectedCompactJSON"`
	ExpectedPrettyRON        string   `json:"expectedPrettyRON"`
	ExpectedCompactRON       string   `json:"expectedCompactRON"`
	ExpectedCanonicalRONXXH3 string   `json:"expectedCanonicalRONXXH3"`
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
			if tc.ExpectedCanonicalRONXXH3 != "" {
				assertCanonicalRONHash(t, tc.ExpectedCanonicalRONXXH3, compactRON)
			}
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

func TestToJSONDuplicateKeysUseLastValue(t *testing.T) {
	got, err := ToJSON([]byte("item {name first name second count 1}"))
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	want := []byte(`{"item":{"count":1,"name":"second"}}`)
	assertBytesEqual(t, want, got)
}

func TestFromJSONCompactDuplicateKeysUseLastValue(t *testing.T) {
	got, err := FromJSONCompact([]byte(`{"item":{"name":"first","name":"second","count":1}}`))
	if err != nil {
		t.Fatalf("FromJSONCompact: %v", err)
	}

	want := []byte("item{count 1 name second}")
	assertBytesEqual(t, want, got)
}

func TestRONBuilderReuse(t *testing.T) {
	var builder RONBuilder
	pretty, err := FromJSONInto(&builder, []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("FromJSONInto: %v", err)
	}
	assertBytesEqual(t, []byte("{a 1}\n"), pretty)

	builder.Reset()
	compact, err := FromJSONCompactInto(&builder, []byte(`{"b":2}`))
	if err != nil {
		t.Fatalf("FromJSONCompactInto: %v", err)
	}
	assertBytesEqual(t, []byte("b 2"), compact)
}

func assertCanonicalRONHash(t *testing.T, want string, body []byte) {
	t.Helper()
	got := fmt.Sprintf("%016x", xxh3.Hash(body))
	if got != want {
		t.Fatalf("canonical RON XXH3 mismatch\nwant: %s\n got: %s", want, got)
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
	assertBytesEqual(t, []byte("{\n  b 1\n  a 2\n}\n"), nonCanonicalPretty)
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
