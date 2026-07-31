package app

import (
	"os"
	"path/filepath"
	"testing"
)

//Успешный запуск evaluate
func TestEvalApp_Run(t *testing.T) {
	manifest := createTestFile(t, `{"id":"1","text":"привет"}`)
	hyp := createTestFile(t, `{"id":"1","text":"привет"}`)
	out := filepath.Join(t.TempDir(), "report.json")

	app := NewEvalApp(manifest, hyp, out, "ru-default")
	if err := app.Run(); err != nil {
		t.Errorf("Ошибка: %v", err)
	}
}
//Ошибка при неправильном манифесте
func TestEvalApp_InvalidManifest(t *testing.T) {
	manifest := createTestFile(t, `invalid json`)
	hyp := createTestFile(t, `{"id":"1","text":"привет"}`)

	app := NewEvalApp(manifest, hyp, "", "ru-default")
	if err := app.Run(); err == nil {
		t.Error("Ожидалась ошибка")
	}
}
//Валидация если нет аудио а запись есть
func TestValidateApp_Valid(t *testing.T) {
	manifest := createTestFile(t, `{"id":"1","text":"текст","audio":"1.wav","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1}`)

	app := NewValidateApp(manifest)
	if err := app.Run(); err == nil {
		t.Error("Ожидалась ошибка: нет аудиофайла")
	}
}

//Валидация в пустом манифесте
func TestValidateApp_Invalid(t *testing.T) {
	manifest := createTestFile(t, `{"id":"1"}`)

	app := NewValidateApp(manifest)
	if err := app.Run(); err == nil {
		t.Error("Ожидалась ошибка: некорректный манифест")
	}
}
//Тест import-app при несуществующем архиве
func TestImportApp_Run(t *testing.T) {
	quotas := map[string]int{"crowd": 5, "farfield": 2}
	out := t.TempDir()

	app := NewImportApp("nonexistent.tar", quotas, "test-seed", out)
	if err := app.Run(); err == nil {
		t.Error("Ожидалась ошибка: нет архива")
	}
} 

func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "test.jsonl")
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatalf("Не удалось создать файл: %v", err)
	}
	return tmp
}