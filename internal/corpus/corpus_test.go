package corpus

import (
	"os"
	"path/filepath"
	"testing"
)

// корректный исходный manifest
func TestParseManifest(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"привет мир","duration":2.5}
	{"audio_filepath":"files/2.wav","text":"как дела","duration":1.8}`

	result, err := parseManifest([]byte(input), "crowd", "test-seed")
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
	input := `{"audio_filepath":"files/1.wav","text":"привет мир","duration":2.5`

	_, err := parseManifest([]byte(input), "crowd", "test-seed")
	if err == nil {
		t.Error("Ожидалась ошибка json, но ее нет")
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

	result, err := parseManifest([]byte(input), "crowd", "test-seed")
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

	result, err := parseManifest([]byte(input), "crowd", "test-seed")
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
	result, _ := parseManifest([]byte(input), "crowd", "test-seed")
	if result[0].Domain != "crowd" {
		t.Errorf("domain = %q, want %q", result[0].Domain, "crowd")
	}

	// farfield
	result, _ = parseManifest([]byte(input), "farfield", "test-seed")
	if result[0].Domain != "farfield" {
		t.Errorf("domain = %q, want %q", result[0].Domain, "farfield")
	}
}

// Стабильные ID
func TestStableID(t *testing.T) {
	input := `{"audio_filepath":"files/1.wav","text":"привет","duration":2.5}`

	result1, _ := parseManifest([]byte(input), "crowd", "test-seed")
	result2, _ := parseManifest([]byte(input), "crowd", "test-seed")

	if result1[0].ID != result2[0].ID {
		t.Errorf("ID не стабильный: %q != %q", result1[0].ID, result2[0].ID)
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
	// Создаем тестовые записи
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
	// Создаем два аудиофайла
	audioDir := t.TempDir()
	audioPath := filepath.Join(audioDir, "test.wav")
	os.WriteFile(audioPath, []byte("test"), 0644)

	// В манифесте две записи ссылаются на один файл
	content := `{"id":"1","audio":"test.wav","text":"текст","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":""}
{"id":"2","audio":"test.wav","text":"текст2","language":"ru","duration_ms":2000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":""}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	result, _ := Validation(tmpFile)
	if result != false {
		t.Errorf("valid = %v, want false", result)
	}
}

//Неверная SHA-256

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
