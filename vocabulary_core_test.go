package ron

import (
	"net/url"
	"testing"

	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
)

func TestCoreVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyCoreV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readRONValueFixture(t, root, tc.ExpectedRON)

			got, err := FromJSON(input, EnableVocabularies(VocabularyCoreV1))
			if err != nil {
				t.Fatalf("FromJSON core vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got, EnableVocabularies(VocabularyCoreV1))
			if err != nil {
				t.Fatalf("ToJSON core vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestCoreVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularyCoreV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input, EnableVocabularies(VocabularyCoreV1)); err == nil {
				t.Fatal("FromJSON succeeded for invalid core vocabulary input")
			}
		})
	}
}

func TestCoreVocabularyRejectsInvalidRON(t *testing.T) {
	if _, err := ToJSON([]byte(`bad {#uid not-a-uuid}`), EnableVocabularies(VocabularyCoreV1)); err == nil {
		t.Fatal("ToJSON succeeded for invalid core vocabulary input")
	}
}

func TestCoreVocabularyParsesNativeValuesFromJSON(t *testing.T) {
	value, err := decodeJSON([]byte(`{
		"id":{"#uid":"00112233-4455-6677-8899-aabbccddeeff"},
		"homepage":{"#url":"https://example.com/docs?q=ron#intro"},
		"regex":{"#rx":["^foo\\d+$","i"]},
		"price":{"#dec":"123.45"},
		"payload":{"#b64":"3q2-7w"},
		"emptyHash":{"#sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		"parent":{"#":300},
		"opaque":{"#tag":[127,{"mode":"raw","value":[1,2,3]}]}
	}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	parsed, err := coreVocabularyOptions().parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	if got, ok := objectValue(object, "id").(uuid.UUID); !ok || got.String() != "00112233-4455-6677-8899-aabbccddeeff" {
		t.Fatalf("id type = %T %[1]v", objectValue(object, "id"))
	}
	if got, ok := objectValue(object, "homepage").(*url.URL); !ok || got.String() != "https://example.com/docs?q=ron#intro" {
		t.Fatalf("homepage type = %T %[1]v", objectValue(object, "homepage"))
	}
	if got, ok := objectValue(object, "regex").(RegExp); !ok || got.Source != "^foo\\d+$" || got.Flags != "i" || !got.MatchString("FOO123") {
		t.Fatalf("regex type = %T %[1]v", objectValue(object, "regex"))
	}
	if got, ok := objectValue(object, "price").(*apd.Decimal); !ok || got.Text('f') != "123.45" {
		t.Fatalf("price type = %T %[1]v", objectValue(object, "price"))
	}
	if got, ok := objectValue(object, "payload").(Bytes); !ok || string(got) != string([]byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("payload type = %T %[1]v", objectValue(object, "payload"))
	}
	if _, ok := objectValue(object, "emptyHash").(SHA256); !ok {
		t.Fatalf("emptyHash type = %T %[1]v", objectValue(object, "emptyHash"))
	}
	if got, ok := objectValue(object, "parent").(EntityRef); !ok || got.Value == nil {
		t.Fatalf("parent type = %T %[1]v", objectValue(object, "parent"))
	}
	if got, ok := objectValue(object, "opaque").(OpaqueTag); !ok || got.Payload == nil {
		t.Fatalf("opaque type = %T %[1]v", objectValue(object, "opaque"))
	}
}

func TestCoreVocabularyRendersAPDDecimal(t *testing.T) {
	var value apd.Decimal
	if _, _, err := value.SetString("1.230"); err != nil {
		t.Fatalf("SetString: %v", err)
	}

	got, err := FromJSON([]byte(`{"price":{"#dec":"1.23"}}`), EnableVocabularies(VocabularyCoreV1))
	if err != nil {
		t.Fatalf("FromJSON #dec: %v", err)
	}
	assertBytesEqual(t, []byte("price {#dec '1.23'}"), got)

	var buf []byte
	buf = append(buf, renderScalar(&value, true)...)
	assertBytesEqual(t, []byte("{#dec '1.23'}"), buf)
}

func TestCoreVocabularyParsesNativeValuesFromRON(t *testing.T) {
	value, err := parse([]byte(`id {#uid 00112233-4455-6677-8899-aabbccddeeff} regex {#rx [(?<word>\\w+) m]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	parsed, err := coreVocabularyOptions().parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(map[string]any)
	if got, ok := object["id"].(uuid.UUID); !ok || got.String() != "00112233-4455-6677-8899-aabbccddeeff" {
		t.Fatalf("id type = %T %[1]v", object["id"])
	}
	if got, ok := object["regex"].(RegExp); !ok || got.Source != "(?<word>\\w+)" || got.Flags != "m" || !got.MatchString("word") {
		t.Fatalf("regex type = %T %[1]v", object["regex"])
	}
}

func TestCoreVocabularyRegExpConvertsJSSyntax(t *testing.T) {
	value, err := ParseRegExp(`^\u0041[\b]\cJ$`, "is")
	if err != nil {
		t.Fatalf("ParseRegExp: %v", err)
	}

	pattern, err := value.GoPattern()
	if err != nil {
		t.Fatalf("GoPattern: %v", err)
	}
	if pattern != `(?is:^\x{0041}[\x{08}]\x{0a}$)` {
		t.Fatalf("GoPattern = %q", pattern)
	}
	if !value.MatchString("a\b\n") {
		t.Fatal("converted regexp did not match")
	}
}

func coreVocabularyOptions() optionState {
	opts := optionState{}
	EnableVocabularies(VocabularyCoreV1)(&opts)
	return opts
}

func objectValue(object orderedObject, key string) any {
	for _, member := range object.Members {
		if member.Key == key {
			return member.Value
		}
	}
	return nil
}
