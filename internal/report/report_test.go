package report

import (
	"math"
	"sttq/internal/corpus"
	"testing"
)

func TestCalculateWER(t *testing.T) {
	// Создаем тестовые результаты
	results := []corpus.Result{
		{
			ID:             "1",
			ReferenceWords: 3,
			Hits:           2,
			Substitutions:  1,
			Deletions:      0,
			Insertions:     0,
			WER:            1.0 / 3.0,
		},
		{
			ID:             "2",
			ReferenceWords: 2,
			Hits:           1,
			Substitutions:  1,
			Deletions:      0,
			Insertions:     0,
			WER:            1.0 / 2.0,
		},
	}

	builder := NewBuilder()
	reportData := builder.Build(results)
	expectedWER := 0.4
	if reportData.Summary.WER != expectedWER {
		t.Errorf("WER = %v, want %v", reportData.Summary.WER, expectedWER)
	}
}

func TestGroupByTag(t *testing.T) {
	results := []corpus.Result{
		{
			ID:             "1",
			Tags:           []string{"crowd"},
			ReferenceWords: 3,
			Substitutions:  1,
			Deletions:      0,
			Insertions:     0,
			WER:            1.0 / 3.0,
		},
		{
			ID:             "2",
			Tags:           []string{"farfield"},
			ReferenceWords: 2,
			Substitutions:  1,
			Deletions:      0,
			Insertions:     0,
			WER:            1.0 / 2.0,
		},
		{
			ID:             "3",
			Tags:           []string{"crowd"},
			ReferenceWords: 4,
			Substitutions:  2,
			Deletions:      0,
			Insertions:     0,
			WER:            2.0 / 4.0,
		},
	}

	builder := NewBuilder()
	reportData := builder.Build(results)

	crowdWER := reportData.Groups.ByTag["crowd"].WER
	expectedCrowdWER := math.Round((3.0/7.0)*100000) / 100000
	if crowdWER != expectedCrowdWER {
		t.Errorf("crowd WER = %v, want %v", crowdWER, expectedCrowdWER)
	}
}

func TestCoverage(t *testing.T) {
	results := []corpus.Result{
		{ID: "1", Error: "", Hypothesis: "text"},          // успешная
		{ID: "2", Error: "no hypothesis", Hypothesis: ""}, // ошибка
		{ID: "3", Error: "", Hypothesis: "text"},          // успешная
	}

	builder := NewBuilder()
	reportData := builder.Build(results)

	// Coverage = 2 / 3 = 0.6667
	expectedCoverage := math.Round((2.0/3.0)*100000) / 100000
	if reportData.Summary.Coverage != expectedCoverage {
		t.Errorf("Coverage = %v, want %v", reportData.Summary.Coverage, expectedCoverage)
	}
}
