package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
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
func WriteHTML(jsonPath, htmlPath string) error {
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return err
	}

	t, err := template.New("report").Funcs(template.FuncMap{
		"mul": func(a, b float64) float64 {
			return a * b
		},
	}).Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, report)
}
