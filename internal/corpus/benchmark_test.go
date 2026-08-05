package corpus

import (
	"encoding/json"
	"os"
	"testing"
)

func BenchmarkReadManifest(b *testing.B) {
	tmpFile := createBenchmarkManifest(b, 10000)
	defer os.Remove(tmpFile)

	reader := NewReader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.ReadManifest(tmpFile)
	}
}

func createBenchmarkManifest(b *testing.B, count int) string {
	b.Helper()
	tmpFile, err := os.CreateTemp("", "benchmark-manifest-*.jsonl")
	if err != nil {
		b.Fatalf("Ошибка создания файла: %v", err)
	}
	defer tmpFile.Close()

	encoder := json.NewEncoder(tmpFile)
	for i := 0; i < count; i++ {
		rec := Record{
			ID:         "id-" + string(rune(i)),
			Audio:      "audio/1.wav",
			Text:       "текст",
			Language:   "ru",
			Duration:   1000,
			SampleRate: 16000,
			Channels:   1,
			Tags:       []string{"crowd"},
		}
		if err := encoder.Encode(rec); err != nil {
			b.Fatalf("Ошибка записи: %v", err)
		}
	}
	return tmpFile.Name()
}
func BenchmarkReadHypotheses(b *testing.B) {
	tmpFile := createBenchmarkHypotheses(b, 10000)
	defer os.Remove(tmpFile)

	reader := NewReader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.ReadHypotheses(tmpFile)
	}
}

func createBenchmarkHypotheses(b *testing.B, count int) string {
	b.Helper()
	tmpFile, err := os.CreateTemp("", "benchmark-hyp-*.jsonl")
	if err != nil {
		b.Fatalf("Ошибка создания файла: %v", err)
	}
	defer tmpFile.Close()

	encoder := json.NewEncoder(tmpFile)
	for i := 0; i < count; i++ {
		rec := Hypothesis{
			ID:   "id-" + string(rune(i)),
			Text: "текст",
		}
		if err := encoder.Encode(rec); err != nil {
			b.Fatalf("Ошибка записи: %v", err)
		}
	}
	return tmpFile.Name()
}

func BenchmarkParseManifest(b *testing.B) {
	
}
func createBenchmarkData(count int)