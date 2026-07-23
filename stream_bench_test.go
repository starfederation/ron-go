package ron

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

func BenchmarkEncoderCompact(b *testing.B) {
	value := benchmarkStreamValue()
	encoder := NewEncoder(io.Discard, IsPretty(false))
	b.ReportAllocs()
	for b.Loop() {
		if err := encoder.Encode(value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNdronEncode(b *testing.B) {
	value := benchmarkStreamValue()
	encoder := NewNdronEncoder(io.Discard)
	b.ReportAllocs()
	for b.Loop() {
		if err := encoder.Encode(value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRonSequenceEncode(b *testing.B) {
	value := benchmarkStreamValue()
	encoder := NewRonSequenceEncoder(io.Discard, IsPretty(false))
	b.ReportAllocs()
	for b.Loop() {
		if err := encoder.Encode(value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNdronDecode(b *testing.B) {
	stream := benchmarkNdronStream(b, 1024)
	decoder := NewNdronDecoder(bytes.NewReader(stream))
	b.SetBytes(int64(len(stream) / 1024))
	b.ReportAllocs()
	for b.Loop() {
		var value json.RawMessage
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			decoder = NewNdronDecoder(bytes.NewReader(stream))
			err = decoder.Decode(&value)
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNdronDecodeStruct(b *testing.B) {
	stream := benchmarkNdronStream(b, 1024)
	decoder := NewNdronDecoder(bytes.NewReader(stream))
	b.SetBytes(int64(len(stream) / 1024))
	b.ReportAllocs()
	for b.Loop() {
		var value struct {
			Active  bool   `json:"active"`
			ID      int    `json:"id"`
			Message string `json:"message"`
			Nested  struct {
				Count int      `json:"count"`
				Tags  []string `json:"tags"`
			} `json:"nested"`
		}
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			decoder = NewNdronDecoder(bytes.NewReader(stream))
			err = decoder.Decode(&value)
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRonSequenceDecode(b *testing.B) {
	stream := benchmarkRonSequenceStream(b, 1024)
	decoder := NewRonSequenceDecoder(bytes.NewReader(stream))
	b.SetBytes(int64(len(stream) / 1024))
	b.ReportAllocs()
	for b.Loop() {
		var value json.RawMessage
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			decoder = NewRonSequenceDecoder(bytes.NewReader(stream))
			err = decoder.Decode(&value)
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkStreamValue() map[string]any {
	return map[string]any{
		"active":  true,
		"id":      1538289,
		"message": "first\nsecond",
		"nested": map[string]any{
			"count": 3,
			"tags":  []string{"alpha", "beta", "gamma"},
		},
	}
}

func benchmarkNdronStream(b *testing.B, records int) []byte {
	b.Helper()
	var buf bytes.Buffer
	encoder := NewNdronEncoder(&buf)
	value := benchmarkStreamValue()
	for range records {
		if err := encoder.Encode(value); err != nil {
			b.Fatal(err)
		}
	}
	return buf.Bytes()
}

func benchmarkRonSequenceStream(b *testing.B, records int) []byte {
	b.Helper()
	var buf bytes.Buffer
	encoder := NewRonSequenceEncoder(&buf, IsPretty(false))
	value := benchmarkStreamValue()
	for range records {
		if err := encoder.Encode(value); err != nil {
			b.Fatal(err)
		}
	}
	return buf.Bytes()
}
