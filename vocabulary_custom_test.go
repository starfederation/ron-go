package ron

import "testing"

type invoiceTotal struct {
	Currency string
	Amount   string
}

func TestCustomVocabularyFixtures(t *testing.T) {
	root, manifest := loadVocabularyManifest(t)
	for _, tc := range manifest.Valid {
		if !hasVocabulary(tc.Vocabularies, invoiceVocabularyURI) {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			input := readConformanceFile(t, root, tc.InputJSON)
			expected := readRONValueFixture(t, root, tc.ExpectedRON)

			got, err := FromJSON(input, invoiceVocabularyOption())
			if err != nil {
				t.Fatalf("FromJSON custom vocabulary: %v", err)
			}
			assertBytesEqual(t, expected, got)

			jsonBody, err := ToJSON(got, invoiceVocabularyOption())
			if err != nil {
				t.Fatalf("ToJSON custom vocabulary: %v", err)
			}
			assertJSONEqual(t, input, jsonBody)
		})
	}
}

func TestCustomVocabularyRejectsUnknownWithoutOption(t *testing.T) {
	_, err := FromJSON(
		[]byte(`{"amount":{"#com.example/money":["USD","123.45"]}}`),
		EnableVocabularies(invoiceVocabularyURI),
	)
	if err == nil {
		t.Fatal("FromJSON accepted custom vocabulary without custom option")
	}
}

func TestCustomVocabularyRendersNativeValues(t *testing.T) {
	got, err := FromJSON(
		[]byte(`{"amount":{"currency":"USD","amount":"123.45"}}`),
		MapJSONValues(func(path []JSONPathSegment, value any) (any, bool) {
			if len(path) == 1 && !path[0].IsIndex && path[0].Key == "amount" {
				return invoiceTotal{Currency: "USD", Amount: "123.45"}, true
			}
			return nil, false
		}),
		invoiceVocabularyOption(),
	)
	if err != nil {
		t.Fatalf("FromJSON custom native value: %v", err)
	}
	assertBytesEqual(t, []byte("amount {#com.example/money [USD '123.45']}"), got)
}

const invoiceVocabularyURI = "https://example.com/vocab/invoice/v1"

func invoiceVocabularyOption() Option {
	return UseCustomVocabulary(CustomVocabulary{
		URI: invoiceVocabularyURI,
		Tags: []string{
			"#com.example/money",
			"#com.example/rating",
			"#com.example/tags",
		},
		Parse: func(tag string, payload any) (any, error) {
			switch tag {
			case "#com.example/money":
				switch payload := payload.(type) {
				case []any:
					if len(payload) != 2 {
						return nil, newError("invalid custom money payload")
					}
					if _, ok := payload[0].(string); !ok {
						return nil, newError("invalid custom money payload")
					}
					if _, ok := payload[1].(string); !ok {
						return nil, newError("invalid custom money payload")
					}
				case orderedObject, map[string]any:
				default:
					return nil, newError("invalid custom money payload")
				}
			case "#com.example/rating":
				if _, ok := numberAsInt64(payload); !ok {
					return nil, newError("invalid custom rating payload")
				}
			case "#com.example/tags":
				values, ok := payload.([]any)
				if !ok {
					return nil, newError("invalid custom tags payload")
				}
				for _, value := range values {
					if _, ok := value.(string); !ok {
						return nil, newError("invalid custom tags payload")
					}
				}
			default:
				return nil, newError("unsupported custom tag")
			}
			return Custom(tag, payload), nil
		},
		Render: func(value any) (string, any, bool) {
			total, ok := value.(invoiceTotal)
			if !ok {
				return "", nil, false
			}
			return "#com.example/money", []any{total.Currency, total.Amount}, true
		},
	})
}
