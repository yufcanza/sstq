package report

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

//Превышение порога -> FAIL
func TestCompare_FAIL_1(t *testing.T) {
	baseline := Report{
		Summary: Summary{WER: 0.10, CER: 0.05, Coverage: 1.0},
		Records: []RecordEntry{},
		Errors:  []ErrorEntry{},
	}
	current := Report{
		Summary: Summary{WER: 0.15, CER: 0.05, Coverage: 1.0},
		Records: []RecordEntry{},
		Errors:  []ErrorEntry{},
	}
	baselinePath := createTempReport(t, baseline)
	currentPath := createTempReport(t, current)

	result, err, errcode := Compare(baselinePath, currentPath, 0.02, 0.02)
	if err != nil {
		t.Fatalf("Ошибка сравнения: %v", err)
	}
	if errcode == 1 {
		t.Errorf("Status = %s, want FAIL", result.Status)
	}
	werDelta := math.Round(result.Summary.WERdelta*1000) / 1000
	if werDelta != 0.05 {
		t.Errorf("WERDelta = %v, want 0.05", werDelta)
	}
}
//улуучшение сверх порогов -> PASS
func TestCompare_PASS_0(t *testing.T) {
	baseline := Report{
		Summary: Summary{WER: 0.20, CER: 0.10, Coverage: 1.0},
		Records: []RecordEntry{},
		Errors:  []ErrorEntry{},
	}
	current := Report{
		Summary: Summary{WER: 0.10, CER: 0.05, Coverage: 1.0},
		Records: []RecordEntry{},
		Errors:  []ErrorEntry{},
	}
	baselinePath := createTempReport(t, baseline)
	currentPath := createTempReport(t, current)

	result, err, errcode := Compare(baselinePath, currentPath, 0.02, 0.02)
	if err != nil {
		t.Fatalf("Ошибка сравнения: %v", err)
	}
	if errcode == 0 {
		t.Errorf("Status = %s, want PASS", result.Status)
	}
	werDelta := math.Round(result.Summary.WERdelta*1000) / 1000
	cerDelta := math.Round(result.Summary.CERdelta*1000) / 1000
	if werDelta >= 0 {
		t.Errorf("WERDelta = %v, want >0", werDelta)
	}
	if cerDelta >= 0 {
		t.Errorf("CERDelta = %v, want >0", cerDelta)
	}

}
//Дельта равна порогу -> PASS
func TestCompare_PASS_DeltaEqualThreshold(t *testing.T) {
	baseline := Report{
		Summary: Summary{WER: 0.20, CER: 0.10, Coverage: 1.0},
		Records: []RecordEntry{},
		Errors:  []ErrorEntry{},
	}
	current := Report{
		Summary: Summary{WER: 0.22, CER: 0.12, Coverage: 1.0},
		Records: []RecordEntry{},
		Errors:  []ErrorEntry{},
	}
	baselinePath := createTempReport(t, baseline)
	currentPath := createTempReport(t, current)

	result, err, errcode:= Compare(baselinePath, currentPath, 0.02, 0.02)
	if err != nil {
		t.Fatalf("Ошибка сравнения: %v", err)
	}
	if errcode == 0 {
		t.Errorf("Status = %s, want PASS", result.Status)
	}
	werDelta := math.Round(result.Summary.WERdelta*1000) / 1000
	cerDelta := math.Round(result.Summary.CERdelta*1000) / 1000
	if werDelta != 0.02 {
		t.Errorf("WERDelta = %v, want 0.02", werDelta)
	}
	if cerDelta != 0.02 {
		t.Errorf("CERDelta = %v, want 0.02", cerDelta)
	}

}
//Запись есть в baseline и нет в current -> missing
func TestCompare_Record_missing(t *testing.T) {
	baseline := Report{
		Summary: Summary{WER: 0.20, CER: 0.10, Coverage: 1.0},
		Records: []RecordEntry{
			{ID: "rec1", Reference: "текст", Hypothesis: "текст", WER: 0.0},
			{ID: "rec2", Reference: "текст2", Hypothesis: "текст2", WER: 0.0},
		},
		Errors:  []ErrorEntry{},
	}
	current := Report{
		Summary: Summary{WER: 0.10, CER: 0.05, Coverage: 1.0},
		Records: []RecordEntry{
			{ID: "rec1", Reference: "текст", Hypothesis: "текст", WER: 0.0},
		},
		Errors:  []ErrorEntry{},
	}
	baselinePath := createTempReport(t, baseline)
	currentPath := createTempReport(t, current)

	result, err, _ := Compare(baselinePath, currentPath, 0.02, 0.02)
	if err != nil {
		t.Fatalf("Ошибка сравнения: %v", err)
	}
	found := false
	for _, r :=range result.Record{
		if r.ID == "rec2"&&r.Status=="missing"{
			found = true
		}
	}

	if !found {
		t.Errorf("Запись rec2 не отмечена как missing")
	}
}

//Нечитаемый файл -> CODE 2 
func TestCompare_File_Unread(t *testing.T){
	baselinePath := filepath.Join(t.TempDir(), "baseline_does-not-exist.json")
	currentPath := filepath.Join(t.TempDir(), "current_does-not-exist.json")
	_, _, errcode :=Compare(baselinePath, currentPath, 0.02, 0.02)
	if errcode != 2 {
		t.Errorf("Ожидался возврат с кодом 2, но его нет")
	}
	
}

func createTempReport(t *testing.T, report Report) string {
	t.Helper()
	tmpFile := filepath.Join(t.TempDir(), "report.json")
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Ошибка маршалинга: %v", err)
	}
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("Ошибка записи: %v", err)
	}
	return tmpFile
}
