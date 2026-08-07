package report

import (
	"encoding/json"
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

func TestGroupByDuration(t *testing.T) {
    results := []corpus.Result{
        {
            ID:             "1",
            DurationMS:     2000,
            ReferenceWords: 3,
            Substitutions:  1,
            WER:            1.0 / 3.0,
        },
        {
            ID:             "2",
            DurationMS:     5000,
            ReferenceWords: 10,
            Substitutions:  2,
            WER:            2.0 / 10.0,
        },
        {
            ID:             "3",
            DurationMS:     15000, 
            ReferenceWords: 20,
            Substitutions:  5,
            WER:            5.0 / 20.0,
        },
    }

    builder := NewBuilder()
    reportData := builder.Build(results)
    if _, ok := reportData.Groups.ByDuration["short"]; !ok {
        t.Error("группа 'short' не найдена")
    }
    if _, ok := reportData.Groups.ByDuration["medium"]; !ok {
        t.Error("группа 'medium' не найдена")
    }
    if _, ok := reportData.Groups.ByDuration["long"]; !ok {
        t.Error("группа 'long' не найдена")
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

func TestRecordsSorted(t *testing.T) {
    results := []corpus.Result{
        {ID: "b-2", Reference: "текст2"},
        {ID: "a-1", Reference: "текст1"},
        {ID: "c-3", Reference: "текст3"},
    }

    builder := NewBuilder()
    reportData := builder.Build(results)

    expectedIDs := []string{"a-1", "b-2", "c-3"}
    for i, record := range reportData.Records {
        if record.ID != expectedIDs[i] {
            t.Errorf("Records[%d].ID = %v, want %v", i, record.ID, expectedIDs[i])
        }
    }
}

func TestStableJSON(t *testing.T) {
    results := []corpus.Result{
        {ID: "1", Reference: "текст", Hypothesis: "текст", WER: 0.0, CER: 0.0},
    }

    builder := NewBuilder()
    reportData1 := builder.Build(results)
    reportData2 := builder.Build(results)

    json1, _ := json.Marshal(reportData1)
    json2, _ := json.Marshal(reportData2)

    if string(json1) != string(json2) {
        t.Error("JSON не стабильный — разный при одинаковых данных")
    }
}