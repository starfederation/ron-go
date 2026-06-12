package ron

import (
	"strconv"
	"strings"
	"testing"
)

var (
	benchmarkResult []byte
	benchmarkAny    any
)

var (
	benchmarkRON   = []byte(makeBenchmarkRON(256))
	benchmarkJSON  = []byte(makeBenchmarkJSON(256))
	benchmarkValue = mustParseBenchmarkRON(benchmarkRON)
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

func BenchmarkToJSONBuilder(b *testing.B) {
	var builder JSONBuilder
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		builder.Reset()
		result, err := ToJSONInto(&builder, benchmarkRON)
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

func BenchmarkFromJSONCompactBuilder(b *testing.B) {
	var builder RONBuilder
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkJSON)))
	for b.Loop() {
		builder.Reset()
		result, err := FromJSONCompactInto(&builder, benchmarkJSON)
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

func BenchmarkFromJSONPrettyBuilder(b *testing.B) {
	var builder RONBuilder
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkJSON)))
	for b.Loop() {
		builder.Reset()
		result, err := FromJSONInto(&builder, benchmarkJSON)
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
		benchmarkAny = result
	}
}

func BenchmarkRenderRONCompact(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		benchmarkResult = renderCompactCap(benchmarkValue, len(benchmarkRON))
	}
}

func BenchmarkRenderRONPretty(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkRON)))
	for b.Loop() {
		benchmarkResult = renderCap(benchmarkValue, "  ", len(benchmarkRON)*2)
	}
}

func mustParseBenchmarkRON(src []byte) any {
	value, err := parse(src)
	if err != nil {
		panic(err)
	}
	return value
}

func makeBenchmarkRON(records int) string {
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
}

func makeBenchmarkJSON(records int) string {
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
}
