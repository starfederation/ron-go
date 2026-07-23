package ron

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"testing"
)

type sequenceManifest struct {
	Version    int `json:"version"`
	ValidNdron []struct {
		Name            string                   `json:"name"`
		InputValuesJSON string                   `json:"inputValuesJSON"`
		RonInputs       []string                 `json:"ronInputs"`
		ExpectedNdron   string                   `json:"expectedNDRON"`
		Options         conformanceFormatOptions `json:"options"`
	} `json:"validNDRON"`
	InvalidNdron      []string `json:"invalidNDRON"`
	ValidRonSequences []struct {
		Name                string                   `json:"name"`
		InputValuesJSON     string                   `json:"inputValuesJSON"`
		RonInputs           []string                 `json:"ronInputs"`
		ExpectedRonSequence string                   `json:"expectedRONSequence"`
		Options             conformanceFormatOptions `json:"options"`
	} `json:"validRONSequences"`
	RecoveryRonSequences []struct {
		Name                  string `json:"name"`
		InputRonSequence      string `json:"inputRONSequence"`
		ExpectedValuesJSON    string `json:"expectedValuesJSON"`
		ExpectedElementErrors int    `json:"expectedElementErrors"`
	} `json:"recoveryRONSequences"`
}

type streamValueDecoder interface {
	Decode(any) error
}

type streamValueEncoder interface {
	Encode(any) error
}

func TestNdronConformance(t *testing.T) {
	root, manifest := loadSequenceManifest(t)
	for _, tc := range manifest.ValidNdron {
		t.Run(tc.Name+"/encode", func(t *testing.T) {
			values := readSequenceValues(t, root, tc.InputValuesJSON)
			var buf bytes.Buffer
			encoder := NewNdronEncoder(
				&buf,
				IsCanonical(tc.Options.IsCanonical),
				IsPretty(tc.Options.IsPretty),
			)
			for _, value := range values {
				if err := encoder.Encode(value); err != nil {
					t.Fatalf("Encode: %v", err)
				}
			}
			assertBytesEqual(t, readConformanceFile(t, root, tc.ExpectedNdron), buf.Bytes())
		})

		for _, inputPath := range tc.RonInputs {
			t.Run(tc.Name+"/decode/"+filepath.Base(inputPath), func(t *testing.T) {
				decoder := NewNdronDecoder(bytes.NewReader(readConformanceFile(t, root, inputPath)))
				values, errorCount := decodeStreamValues(t, decoder)
				if errorCount != 0 {
					t.Fatalf("Decode errors = %d, want 0", errorCount)
				}
				assertSequenceValues(t, readConformanceFile(t, root, tc.InputValuesJSON), values)
			})
		}
	}
}

func TestNdronEncoderAlwaysWritesSingleLineRecords(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewNdronEncoder(&buf, IsPretty(true))
	if err := encoder.Encode(map[string]any{
		"a": 1,
		"nested": map[string]bool{
			"b": true,
		},
	}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	assertBytesEqual(t, []byte("a 1 nested{b true}\n"), buf.Bytes())
}

func TestNdronRejectsInvalidFixtures(t *testing.T) {
	root, manifest := loadSequenceManifest(t)
	for _, inputPath := range manifest.InvalidNdron {
		t.Run(filepath.Base(inputPath), func(t *testing.T) {
			decoder := NewNdronDecoder(bytes.NewReader(readConformanceFile(t, root, inputPath)))
			_, errorCount := decodeStreamValues(t, decoder)
			if errorCount == 0 {
				t.Fatal("Decode accepted invalid NDRON")
			}
		})
	}
}

func TestNdronEmptyLinePolicyIsConfigurable(t *testing.T) {
	input := []byte("\ntrue\n")
	decoder := NewNdronDecoder(bytes.NewReader(input))
	var value any
	if err := decoder.Decode(&value); !errors.Is(err, ErrEmptyNdronRecord) {
		t.Fatalf("Decode empty line error = %v, want %v", err, ErrEmptyNdronRecord)
	}
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("Decode after empty line: %v", err)
	}
	if value != true {
		t.Fatalf("decoded value = %#v, want true", value)
	}

	decoder = NewNdronDecoder(bytes.NewReader(input), IgnoreEmptyNdronLines(true))
	value = nil
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("Decode ignoring empty line: %v", err)
	}
	if value != true {
		t.Fatalf("decoded value = %#v, want true", value)
	}
}

func TestStreamDecoderDecodesConsecutiveGoValues(t *testing.T) {
	decoder := NewNdronDecoder(bytes.NewReader([]byte("id 1\nid 2\n")))
	var value struct {
		ID int `json:"id"`
	}
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("Decode first value: %v", err)
	}
	if value.ID != 1 {
		t.Fatalf("first ID = %d, want 1", value.ID)
	}
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("Decode second value: %v", err)
	}
	if value.ID != 2 {
		t.Fatalf("second ID = %d, want 2", value.ID)
	}
}

func TestStreamFramingErrors(t *testing.T) {
	ndron := NewNdronDecoder(bytes.NewReader([]byte("true")))
	var value any
	if err := ndron.Decode(&value); !errors.Is(err, ErrUnterminatedNdronRecord) {
		t.Fatalf("Decode unterminated NDRON error = %v, want %v", err, ErrUnterminatedNdronRecord)
	}

	sequence := NewRonSequenceDecoder(bytes.NewReader([]byte("garbage\x1etrue\n")))
	if err := sequence.Decode(&value); !errors.Is(err, ErrRonSequencePreamble) {
		t.Fatalf("Decode sequence preamble error = %v, want %v", err, ErrRonSequencePreamble)
	}
	if err := sequence.Decode(&value); err != nil {
		t.Fatalf("Decode after sequence preamble: %v", err)
	}
	if value != true {
		t.Fatalf("decoded value = %#v, want true", value)
	}

	sequence = NewRonSequenceDecoder(bytes.NewReader([]byte("\x1etrue")))
	if err := sequence.Decode(&value); !errors.Is(err, ErrTruncatedRonSequence) {
		t.Fatalf("Decode truncated sequence error = %v, want %v", err, ErrTruncatedRonSequence)
	}
}

func TestRonSequenceConformance(t *testing.T) {
	root, manifest := loadSequenceManifest(t)
	for _, tc := range manifest.ValidRonSequences {
		t.Run(tc.Name+"/encode", func(t *testing.T) {
			values := readSequenceValues(t, root, tc.InputValuesJSON)
			var buf bytes.Buffer
			encoder := NewRonSequenceEncoder(
				&buf,
				IsCanonical(tc.Options.IsCanonical),
				IsPretty(tc.Options.IsPretty),
			)
			for _, value := range values {
				if err := encoder.Encode(value); err != nil {
					t.Fatalf("Encode: %v", err)
				}
			}
			assertBytesEqual(t, readConformanceFile(t, root, tc.ExpectedRonSequence), buf.Bytes())
		})

		for _, inputPath := range tc.RonInputs {
			t.Run(tc.Name+"/decode/"+filepath.Base(inputPath), func(t *testing.T) {
				decoder := NewRonSequenceDecoder(bytes.NewReader(readConformanceFile(t, root, inputPath)))
				values, errorCount := decodeStreamValues(t, decoder)
				if errorCount != 0 {
					t.Fatalf("Decode errors = %d, want 0", errorCount)
				}
				assertSequenceValues(t, readConformanceFile(t, root, tc.InputValuesJSON), values)
			})
		}
	}
}

func TestRonSequenceRecoveryFixtures(t *testing.T) {
	root, manifest := loadSequenceManifest(t)
	for _, tc := range manifest.RecoveryRonSequences {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := NewRonSequenceDecoder(bytes.NewReader(readConformanceFile(t, root, tc.InputRonSequence)))
			values, errorCount := decodeStreamValues(t, decoder)
			if errorCount != tc.ExpectedElementErrors {
				t.Fatalf("Decode errors = %d, want %d", errorCount, tc.ExpectedElementErrors)
			}
			assertSequenceValues(t, readConformanceFile(t, root, tc.ExpectedValuesJSON), values)
		})
	}
}

func TestStreamRecordSizeLimitRecoversAtNextRecord(t *testing.T) {
	cases := []struct {
		name    string
		decoder streamValueDecoder
	}{
		{
			name:    "NDRON",
			decoder: NewNdronDecoder(bytes.NewReader([]byte("abcde\ntrue\n")), MaxRecordSize(4)),
		},
		{
			name:    "RON sequence",
			decoder: NewRonSequenceDecoder(bytes.NewReader([]byte("\x1eabcde\n\x1etrue\n")), MaxRecordSize(4)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var value any
			if err := tc.decoder.Decode(&value); !errors.Is(err, ErrRecordTooLarge) {
				t.Fatalf("Decode oversized record error = %v, want %v", err, ErrRecordTooLarge)
			}
			if err := tc.decoder.Decode(&value); err != nil {
				t.Fatalf("Decode after oversized record: %v", err)
			}
			if value != true {
				t.Fatalf("decoded value = %#v, want true", value)
			}
		})
	}
}

func TestStreamNestingLimitRecoversAtNextRecord(t *testing.T) {
	cases := []struct {
		name    string
		decoder streamValueDecoder
	}{
		{
			name:    "NDRON",
			decoder: NewNdronDecoder(bytes.NewReader([]byte("a {b true}\ntrue\n")), MaxNestingDepth(1)),
		},
		{
			name:    "RON sequence",
			decoder: NewRonSequenceDecoder(bytes.NewReader([]byte("\x1ea {b true}\n\x1etrue\n")), MaxNestingDepth(1)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var value any
			if err := tc.decoder.Decode(&value); !errors.Is(err, ErrNestingTooDeep) {
				t.Fatalf("Decode nested record error = %v, want %v", err, ErrNestingTooDeep)
			}
			if err := tc.decoder.Decode(&value); err != nil {
				t.Fatalf("Decode after nested record: %v", err)
			}
			if value != true {
				t.Fatalf("decoded value = %#v, want true", value)
			}
		})
	}
}

func TestStreamEncodersEnforceLimits(t *testing.T) {
	cases := []struct {
		name       string
		newEncoder func(io.Writer, ...Option) streamValueEncoder
	}{
		{
			name: "NDRON",
			newEncoder: func(writer io.Writer, options ...Option) streamValueEncoder {
				return NewNdronEncoder(writer, options...)
			},
		},
		{
			name: "RON sequence",
			newEncoder: func(writer io.Writer, options ...Option) streamValueEncoder {
				return NewRonSequenceEncoder(writer, options...)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/record size", func(t *testing.T) {
			var buf bytes.Buffer
			encoder := tc.newEncoder(&buf, MaxRecordSize(3))
			if err := encoder.Encode("abcd"); !errors.Is(err, ErrRecordTooLarge) {
				t.Fatalf("Encode oversized record error = %v, want %v", err, ErrRecordTooLarge)
			}
			if buf.Len() != 0 {
				t.Fatalf("Encode wrote %d bytes after error", buf.Len())
			}
		})

		t.Run(tc.name+"/nesting", func(t *testing.T) {
			var buf bytes.Buffer
			encoder := tc.newEncoder(&buf, MaxNestingDepth(1))
			value := map[string]any{
				"a": map[string]bool{"b": true},
			}
			if err := encoder.Encode(value); !errors.Is(err, ErrNestingTooDeep) {
				t.Fatalf("Encode nested record error = %v, want %v", err, ErrNestingTooDeep)
			}
			if buf.Len() != 0 {
				t.Fatalf("Encode wrote %d bytes after error", buf.Len())
			}
		})

		t.Run(tc.name+"/writer error", func(t *testing.T) {
			encoder := tc.newEncoder(errorWriter{})
			if err := encoder.Encode(true); !errors.Is(err, errWriteFailed) {
				t.Fatalf("Encode writer error = %v, want %v", err, errWriteFailed)
			}
		})
	}
}

func loadSequenceManifest(t *testing.T) (string, sequenceManifest) {
	t.Helper()
	root := testdataSubdir(t, "sequences")
	body := readFile(t, filepath.Join(root, "manifest.json"))

	var manifest sequenceManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("unmarshal sequence manifest: %v", err)
	}
	if manifest.Version != 1 {
		t.Fatalf("sequence manifest version = %d, want 1", manifest.Version)
	}
	return root, manifest
}

func readSequenceValues(t *testing.T, root, path string) []json.RawMessage {
	t.Helper()
	var values []json.RawMessage
	if err := json.Unmarshal(readConformanceFile(t, root, path), &values); err != nil {
		t.Fatalf("unmarshal sequence values: %v", err)
	}
	return values
}

func decodeStreamValues(t *testing.T, decoder streamValueDecoder) ([]json.RawMessage, int) {
	t.Helper()
	values := make([]json.RawMessage, 0)
	errorCount := 0
	for attempts := 0; attempts < 1000; attempts++ {
		var value json.RawMessage
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			return values, errorCount
		}
		if err != nil {
			errorCount++
			continue
		}
		values = append(values, value)
	}
	t.Fatal("Decode did not reach EOF")
	return nil, 0
}

func assertSequenceValues(t *testing.T, want []byte, got []json.RawMessage) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal decoded sequence values: %v", err)
	}
	assertJSONEqual(t, want, gotJSON)
}
