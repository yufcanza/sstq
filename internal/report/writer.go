package report

import (
	"encoding/json"
	"fmt"
	"os"
)

func Write(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации отчета: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}
	return nil
}
