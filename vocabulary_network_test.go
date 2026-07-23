package ron

import (
	"net/netip"
	"testing"
)

func TestNetworkVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, VocabularyNetworkV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readRONValueFixture(t, root, tc.ExpectedRON)

			got, err := FromJSON(input)
			if err != nil {
				t.Fatalf("FromJSON network vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got)
			if err != nil {
				t.Fatalf("ToJSON network vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestNetworkVocabularyInvalidFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Invalid {
		if !hasVocabulary(tc.Vocabularies, VocabularyNetworkV1) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			if _, err := FromJSON(input); err == nil {
				t.Fatal("FromJSON succeeded for invalid network vocabulary input")
			}
		})
	}
}

func TestNetworkVocabularyParsesNativeValues(t *testing.T) {
	value, err := decodeJSON([]byte(`{
		"host4":{"#ip4":"192.0.2.1"},
		"host6":{"#ip6":"2001:db8::1"},
		"network":{"#cdr":"192.0.2.0/24"}
	}`), nil)
	if err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}

	parsed, err := optionState{vocabularies: defaultVocabularies()}.parseVocabularies(value)
	if err != nil {
		t.Fatalf("parseVocabularies: %v", err)
	}
	object := parsed.(orderedObject)

	if got, ok := objectValue(object, "host4").(IPv4); !ok || got.Addr.String() != "192.0.2.1" {
		t.Fatalf("host4 type = %T %[1]v", objectValue(object, "host4"))
	}
	if got, ok := objectValue(object, "host6").(IPv6); !ok || got.Addr.String() != "2001:db8::1" {
		t.Fatalf("host6 type = %T %[1]v", objectValue(object, "host6"))
	}
	if got, ok := objectValue(object, "network").(CIDR); !ok || got.Prefix.String() != "192.0.2.0/24" {
		t.Fatalf("network type = %T %[1]v", objectValue(object, "network"))
	}
}

func TestNetworkVocabularyRendersNativeValues(t *testing.T) {
	assertBytesEqual(t, []byte("{#ip4 192.0.2.1}"), []byte(renderScalar(IPv4{Addr: netip.MustParseAddr("192.0.2.1")}, true)))
	assertBytesEqual(t, []byte("{#ip6 2001:db8::1}"), []byte(renderScalar(IPv6{Addr: netip.MustParseAddr("2001:db8::1")}, true)))
	assertBytesEqual(t, []byte("{#cdr 192.0.2.0/24}"), []byte(renderScalar(CIDR{Prefix: netip.MustParsePrefix("192.0.2.0/24")}, true)))
}
