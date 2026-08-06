package report

import (
	"encoding/json"
	"sttq/internal/corpus"
	"testing"
)

func BenchmarkReport(b *testing.B) {
	results := make([]corpus.Result, 10000)

	for i := 0; i < 10000; i++ {
		results[i] = corpus.Result{
			ID:             "id-" + string(rune(i%1000)),
			ReferenceWords: 5,
			Hits:           4,
			Substitutions:  1,
			Deletions:      0,
			Insertions:     0,
			WER:            0.2,
			CER:            0.1,
			ExactMatch:     false,
			Tags:           []string{"crowd"},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewBuilder()
		report := builder.Build(results)
		_ = report
	}

}

func TestReportMemory(t *testing.T) {
	results := make([]corpus.Result, 10000)

	for i := 0; i < 10000; i++ {
		results[i] = corpus.Result{
			ID:             "id-" + string(rune(i%1000)),
			ReferenceWords: 5,
			Hits:           4,
			Substitutions:  1,
			Deletions:      0,
			Insertions:     0,
			WER:            0.2,
			CER:            0.1,
			ExactMatch:     false,
			Tags:           []string{"crowd"},
		}
	}

	builder := NewBuilder()
	report := builder.Build(results)
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Ошибка маршалинга: %v", err)
	}

	sizeMB := float64(len(data)) / (1024 * 1024)
	t.Logf("Размер отчета: %.2f MB", sizeMB)

}
