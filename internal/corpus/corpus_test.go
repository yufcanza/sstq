package corpus

import (
	"os"

	"testing"
)
//корректный исходный manifest
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
//Неверный JSON
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

//Пустой текст
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

//Определение domain

//Стабильные ID

//Одинаковая выборка при одинаковом seed

//Квоты crowd/farfield

//Повторяющийся файл

//Неверная SHA-256

//Небезопасный пусть


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
