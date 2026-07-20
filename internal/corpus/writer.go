package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Writer struct {
}
type Result struct {
	ID         string  `json:"id"`
	Reference  string  `json:"reference"`
	Hypothesis string  `json:"hypothesis"`
	WER        float64 `json:"wer"`
	CER        float64 `json:"cer"`
}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteResult(path string, results []Result) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Ошибка создания файла %s: %v", path, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	encoder := json.NewEncoder(writer)

	for _, result := range results {
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("Ошибка записи результата: %v", err)
		}
	}
	return nil
}
