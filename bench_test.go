package ron

import (
	"bytes"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var benchmarkResult []byte

var (
	benchmarkRON = []byte(func() string {
		const records = 256

		var b strings.Builder
		b.Grow(records*96 + 128)
		b.WriteString("meta {count ")
		b.WriteString(strconv.Itoa(records))
		b.WriteString(" label benchmark active true} people [")
		for i := range records {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString("{active ")
			if i%3 == 0 {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
			b.WriteString(" id ")
			b.WriteString(strconv.Itoa(i))
			b.WriteString(" name person_")
			b.WriteString(strconv.Itoa(i))
			b.WriteString(" score ")
			b.WriteString(strconv.FormatFloat(float64(i)*1.25, 'f', 2, 64))
			b.WriteString(" tags [alpha beta group_")
			b.WriteString(strconv.Itoa(i % 10))
			b.WriteString("]}")
		}
		b.WriteByte(']')
		return b.String()
	}())
	benchmarkJSON = []byte(func() string {
		const records = 256

		var b strings.Builder
		b.Grow(records*112 + 128)
		b.WriteString(`{"meta":{"active":true,"count":`)
		b.WriteString(strconv.Itoa(records))
		b.WriteString(`,"label":"benchmark"},"people":[`)
		for i := range records {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(`{"active":`)
			if i%3 == 0 {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
			b.WriteString(`,"id":`)
			b.WriteString(strconv.Itoa(i))
			b.WriteString(`,"name":"person_`)
			b.WriteString(strconv.Itoa(i))
			b.WriteString(`","score":`)
			b.WriteString(strconv.FormatFloat(float64(i)*1.25, 'f', 2, 64))
			b.WriteString(`,"tags":["alpha","beta","group_`)
			b.WriteString(strconv.Itoa(i % 10))
			b.WriteString(`"]}`)
		}
		b.WriteString(`]}`)
		return b.String()
	}())
	benchmarkValue = func() any {
		value, err := parse(benchmarkRON)
		if err != nil {
			panic(err)
		}
		return value
	}()
)

func BenchmarkToJSON(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		result, err := ToJSON(benchmarkRON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkToJSONUntaggedQuery(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		result, err := ToJSON(benchmarkRON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkToJSONTaggedValues(b *testing.B) {
	body := []byte(`{created {#utc 2026-06-13T00:00:00Z} digest {#sha256 e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855} entity {# 123}}`)

	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		result, err := ToJSON(body)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkToJSONBuffer(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		buf.Reset()
		result, err := ToJSONInto(&buf, benchmarkRON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkToJSONPretty(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		result, err := ToJSON(benchmarkRON, PrettyJSON("", "  "))
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkFromJSONCompact(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkJSON)))
	for b.Loop() {
		result, err := FromJSONCompact(benchmarkJSON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkFromJSONCompactBuffer(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkJSON)))
	for b.Loop() {
		buf.Reset()
		result, err := FromJSONCompactInto(&buf, benchmarkJSON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkFromJSONPretty(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkJSON)))
	for b.Loop() {
		result, err := FromJSON(benchmarkJSON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkFromJSONPrettyBuffer(b *testing.B) {
	var buf bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkJSON)))
	for b.Loop() {
		buf.Reset()
		result, err := FromJSONInto(&buf, benchmarkJSON)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkResult = result
	}
}

func BenchmarkParseRON(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		result, err := parse(benchmarkRON)
		if err != nil {
			b.Fatal(err)
		}
		runtime.KeepAlive(result)
	}
}

func BenchmarkRenderRONCompact(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		var buf bytes.Buffer
		buf.Grow(len(benchmarkRON))
		writeCompactValueWithCustom(&buf, benchmarkValue, true, true, nil)
		benchmarkResult = buf.Bytes()
	}
}

func BenchmarkRenderRONPretty(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		var buf bytes.Buffer
		buf.Grow(len(benchmarkRON) * 2)
		writeValueWithCustom(&buf, benchmarkValue, "  ", 0, true, nil)
		buf.WriteByte('\n')
		benchmarkResult = buf.Bytes()
	}
}
