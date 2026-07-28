package corpus

import (
	"encoding/json"
	"fmt"
	"os"
)

type Writer struct{}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteManifest(path string, records []Record) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Ошибка создания файла %s: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("Ошибка записи: %w", err)
		}
	}
	return nil
}

func writeJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("не удалось создать файл: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
