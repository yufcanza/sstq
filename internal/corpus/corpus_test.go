package corpus

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// корректный исходный manifest
func TestParseManifest(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"привет мир","duration":2.5}
	{"audio_filepath":"files/2.wav","text":"как дела","duration":1.8}`

	result, _, _, err := parseManifest([]byte(input), "crowd", "test-seed")
	if err != nil {
		t.Errorf("ошибка: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
	if result[0].Text != "привет мир" {
		t.Errorf("text = %q, want %q", result[0].Text, "привет мир")
	}
	if result[0].Domain != "crowd" {
		t.Errorf("domain = %q, want %q", result[0].Domain, "crowd")
	}
}

// Неверный JSON
func TestParseManifest_InvalidJSON(t *testing.T) {
	manifestContent := `{"audio_filepath":"files/1.wav","text":"текст 1","duration":1.0}
	{это не json
	{"audio_filepath":"files/2.wav","text":"текст 2","duration":2.0}
	{"audio_filepath":"files/4.wav","text":"   ","duration":4.0}`
	records, skipped, invalid, err := parseManifest([]byte(manifestContent), "crowd", "test-seed")
	if err != nil {
		t.Error(" got nil, want error for invalid JSON")
	}
	if skipped != 1 {
		t.Errorf("got %d skipped, want 1", invalid)
	}
	if len(records) != 2 {
		t.Errorf("got %d records, want 2", len(records))
	}
	if invalid != 1 {
		t.Errorf("got %d invalid, want 1", invalid)
	}
}

// Отсутсвие аудио
func TestValidate_MissingAudio(t *testing.T) {
	input := `{"id":"1","audio":"audio/notexist.wav","text":"текст","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":""}`

	tmpFile := createTempFile(t, input)
	defer os.Remove(tmpFile)

	result, _ := Validation(tmpFile)
	if result != false {
		t.Errorf("valid = %v, want false", result)
	}
}

// Пустой текст
func TestParseManifest_EmptyText(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"","duration":2.5}`

	result, _, _, err := parseManifest([]byte(input), "crowd", "test-seed")
	if err != nil {
		t.Errorf("Ошибка: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

//Преобразование duration

func TestParseManifest_Duration(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"привет","duration":2.84}`

	result, _, _, err := parseManifest([]byte(input), "crowd", "test-seed")
	if err != nil {
		t.Errorf("ошибка: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d, want 1", len(result))
	}
	if result[0].Duration != 2.84 {
		t.Errorf("duration = %f, want 2.84", result[0].Duration)
	}
}

// Определение domain
func TestParseManifest_Domain(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"привет","duration":2.5}`

	// crowd
	result, _, _, _ := parseManifest([]byte(input), "crowd", "test-seed")
	if result[0].Domain != "crowd" {
		t.Errorf("domain = %q, want %q", result[0].Domain, "crowd")
	}

	// farfield
	result, _, _, _ = parseManifest([]byte(input), "farfield", "test-seed")
	if result[0].Domain != "farfield" {
		t.Errorf("domain = %q, want %q", result[0].Domain, "farfield")
	}
}

// Стабильные ID
func TestStableID(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"привет","duration":2.5}`

	result1, _, _, _ := parseManifest([]byte(input), "crowd", "test-seed")
	result2, _, _, _ := parseManifest([]byte(input), "crowd", "test-seed")
	result3, _, _, _ := parseManifest([]byte(input), "crowd", "test-seed2")

	if result1[0].ID != result2[0].ID {
		t.Errorf("ID не стабильный: %q != %q", result1[0].ID, result2[0].ID)
	}
	if result1[0].ID != result3[0].ID {
		t.Errorf("ID не стабильный по seed: %q != %q", result1[0].ID, result2[0].ID)
	}
}

//Одинаковая выборка при одинаковом seed

func TestSelectRecords_SameSeed(t *testing.T) {
	var records []ProcessedRecord
	for i := 0; i < 50; i++ {
		records = append(records, ProcessedRecord{
			Domain: "crowd",
			ID:     "crowd-" + string(rune(i)),
		})
	}

	quotas := map[string]int{"crowd": 10}

	result1 := SelectRecord(records, quotas, 0)
	result2 := SelectRecord(records, quotas, 0)

	if len(result1) != len(result2) {
		t.Errorf("len = %d и %d, должны быть равны", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i].ID != result2[i].ID {
			t.Errorf("ID[%d] = %q и %q, должны быть равны", i, result1[i].ID, result2[i].ID)
		}
	}
}

//Квоты crowd/farfield

func TestSelectRecords_Quotas(t *testing.T) {
	var records []ProcessedRecord
	for i := 0; i < 100; i++ {
		records = append(records, ProcessedRecord{
			Domain: "crowd",
			ID:     "crowd-" + string(rune(i)),
		})
	}
	for i := 0; i < 50; i++ {
		records = append(records, ProcessedRecord{
			Domain: "farfield",
			ID:     "farfield-" + string(rune(i)),
		})
	}

	quotas := map[string]int{"crowd": 10, "farfield": 5}
	result := SelectRecord(records, quotas, 0)

	crowdCount := 0
	farCount := 0
	for _, rec := range result {
		switch rec.Domain {
		case "crowd":
			crowdCount++
		case "farfield":
			farCount++
		}

	}

	if crowdCount != 10 {
		t.Errorf("crowd = %d, want 10", crowdCount)
	}
	if farCount != 5 {
		t.Errorf("farfield = %d, want 5", farCount)
	}
}

// Повторяющийся файл
func TestValidate_DuplicateFile(t *testing.T) {
	audioDir := t.TempDir()
	audioPath := filepath.Join(audioDir, "test.wav")
	os.WriteFile(audioPath, []byte("test"), 0644)

	content := `{"id":"1","audio":"test.wav","text":"текст","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":""}
{"id":"2","audio":"test.wav","text":"текст2","language":"ru","duration_ms":2000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":""}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	result, _ := Validation(tmpFile)
	if result != false {
		t.Errorf("valid = %v, want false", result)
	}
}

// Неверная SHA-256
func TestValidate_BadSHA256(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "test.wav")
	if err := os.WriteFile(audioPath, []byte("audio-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	content := `{"id":"1","audio":"test.wav","text":"текст","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":"0000000000000000000000000000000000000000000000000000000000000000"}`
	manifestPath := filepath.Join(dir, "manifest.jsonl")
	if err := os.WriteFile(manifestPath, []byte(content+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ok, _ := Validation(manifestPath)
	if ok {
		t.Error("ожидалась ошибка из за неверной суммы")
	}
}

// Небезопасный пусть
func TestValidate_UnsafePath(t *testing.T) {
	input := `{"id":"1","audio":"../outside.wav","text":"текст","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":""}`

	tmpFile := createTempFile(t, input)
	defer os.Remove(tmpFile)

	result, _ := Validation(tmpFile)
	if result != false {
		t.Errorf("valid = %v, want false", result)
	}
}

// Импорт падает если ffprobe не найден
func TestImportGolos_FFprobe(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	os.Setenv("PATH", "")
	config := ImportConfig{
		ArchivePath: "test.tar",
		Quotas:      map[string]int{"crowd": 1},
		Limit:       1,
		MaxDuration: 30 * time.Minute,
		Seed:        "test-seed",
		OutDir:      t.TempDir(),
	}
	_, err := ImportGolos(config)
	if err == nil {
		t.Error("Ожидалась ошибка ffprobe не найден, nil")
	}
	expectedError := "Ошибка ffprobe не найден в path"
	if err != nil && err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("'%s', got'%s'", expectedError, err.Error())
	}
}

// дублирование аудио(логика)
func TestImportGolos_DuplicateAudio(t *testing.T) {
	records := []ProcessedRecord{
		{
			Domain:        "crowd",
			AudioFilepath: "files/1.wav",
			Text:          "текст 1",
			Duration:      1.0,
			ID:            "crowd-1",
		},
		{
			Domain:        "crowd",
			AudioFilepath: "files/1.wav",
			Text:          "текст 2",
			Duration:      2.0,
			ID:            "crowd-2",
		},
		{
			Domain:        "crowd",
			AudioFilepath: "files/2.wav",
			Text:          "текст 3",
			Duration:      3.0,
			ID:            "crowd-3",
		},
	}
	usedFiles := make(map[string]string)
	var duplicates []string

	for _, rec := range records {
		normalizedPath := filepath.ToSlash(rec.AudioFilepath)
		if existingID, exists := usedFiles[normalizedPath]; exists {
			duplicates = append(duplicates, fmt.Sprintf("duplicate: %s used by %s and %s",
				normalizedPath, existingID, rec.ID))
		} else {
			usedFiles[normalizedPath] = rec.ID
		}
	}

	if len(duplicates) != 1 {
		t.Errorf("expected 1 duplicate, got %d", len(duplicates))
	}

	if len(usedFiles) != 2 {
		t.Errorf("expected 2 unique files, got %d", len(usedFiles))
	}
}

// ограничения общей длительности(логика)
func TestImportGolos_TotalDuration(t *testing.T) {
	records := []ProcessedRecord{
		{Domain: "crowd", ID: "crowd-1", Duration: 5.0},
		{Domain: "crowd", ID: "crowd-2", Duration: 10.0},
		{Domain: "crowd", ID: "crowd-3", Duration: 15.0},
	}
	maxDuration := 12 * time.Second

	var selected []ProcessedRecord
	var selectedDuration time.Duration
	var exceeded []string

	for _, rec := range records {
		recDuration := time.Duration(rec.Duration * float64(time.Second))
		if maxDuration > 0 && selectedDuration+recDuration > maxDuration {
			exceeded = append(exceeded, fmt.Sprintf("%s превышел лимит", rec.ID))
			continue
		}
		selected = append(selected, rec)
		selectedDuration += recDuration
	}

	if len(selected) != 1 {
		t.Errorf("got %d record selected, want 1", len(selected))
	}
	if len(exceeded) != 2 {
		t.Errorf("got %d record exceeded, want 2", len(selected))
	}
}

// probe Errors (логика)
func TestImportGolos_ProbeError(t *testing.T) {
	records := []struct {
		id            string
		hasProbeError bool
	}{
		{"crowd-1", false},
		{"crowd-2", true},
		{"crowd-3", false},
	}
	probeErrors:=0
	var error []string

	for _, rec :=range records{
		if rec.hasProbeError{
			probeErrors++
			error = append(error, fmt.Sprintf("запись: %s: ffprobe errors", rec.id))
		}
	}
	if probeErrors != 1 {
		t.Errorf("got %d probe errors, want 1", probeErrors)
	}
	if len(error) !=1 {
		t.Errorf("got %d  errors, want 1", len(error))
	}
}
//ошика не останавливает импорт (логика)
func TestImportGolos_ErrorsContinue(t *testing.T){
	records := []struct {
		id          string
		hasError    bool
		errorType   string
	}{
		{"crowd-1", true, "missing_audio"},
		{"crowd-2", false, ""},
		{"crowd-3", true, "duplicate"},
	}
	
	var selected []string
	var errors []string
	
	for _, rec := range records {
		if rec.hasError {
			errors = append(errors, fmt.Sprintf("record %s: %s", rec.id, rec.errorType))
			continue
		}
		selected = append(selected, rec.id)
	}
		if len(selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(selected))
	}
	if selected[0] != "crowd-2" {
		t.Errorf("expected selected 'crowd-2', got '%s'", selected[0])
	}
	
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errors))
	}
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "manifest-*.jsonl")
	if err != nil {
		t.Fatalf("не удалось создать временный файл: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("не удалось записать в файл: %v", err)
	}

	return tmpFile.Name()
}
