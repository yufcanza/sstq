package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sttq/internal/atomicfile"
)

func Write(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации отчета: %w", err)
	}

	return atomicfile.WriteFile(path, data, 0644)
}
func WriteHTML(jsonPath, htmlPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("Входной отчет не найден: %w", err)
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("Ошибка парсинга json: %w", err)
	}

	t, err := template.New("report").Funcs(template.FuncMap{
		"percent": func(v float64) string {
			return fmt.Sprintf("%.1f%%", v*100)
		},
	}).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("Ошибка парсинга шаблона: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, report); err != nil {
		return fmt.Errorf("Ошибка генерации HTML: %w", err)
	}
	if err := atomicfile.WriteFile(htmlPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Ошибка атомарной записи HTML: %w", err)
	}
	return nil
}
