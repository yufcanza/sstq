package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sttq/internal/metrics"
)

type Writer struct {
}
type Result struct {
	ID                  string                  `json:"id"`
	Reference           string                  `json:"reference"`
	Hypothesis          string                  `json:"hypothesis"`
	NormalizedReference string                  `json:"normalized_reference"`
	NormalizeHypothesis string                  `json:"normalized_hypothesis"`
	ReferenceWords      int                     `json:"reference_words"`
	Hits                int                     `json:"hits"`
	Substitutions       int                     `json:"substitutions"`
	Deletions           int                     `json:"deletions"`
	Insertions          int                     `jsin:"insertions"`
	WER                 float64                 `json:"wer"`
	CER                 float64                 `json:"cer"`
	ExactMatch          bool                    `json:"exact_match"`
	Tags                []string                `json:"tags,omitempty"`
	Alignment           []metrics.AlignmentItem `json:"alignment"`
}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteJSON(path string, items interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Ошибка создания файла %s: %v", path, err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	switch v := items.(type) {
	case []Result:
		for _, result := range v {
			if err := encoder.Encode(result); err != nil {
				return fmt.Errorf("Ошибка записи результата: %v", err)
			}
		}
	case []Record:
		for _, result := range v {
			if err := encoder.Encode(result); err != nil {
				return fmt.Errorf("Ошибка записи результата: %v", err)
			}
		}
	default:
		return fmt.Errorf("неподдерживаемый тип: %T", items)
	}
	return nil
}
func (w *Writer) WriteResult(path string, results []Result) error {

	roundedResult := make([]Result, len(results))
	for i, r := range results {
		roundedResult[i] = r
		roundedResult[i].WER = math.Round(r.WER*1000000) / 1000000
		roundedResult[i].CER = math.Round(r.CER*1000000) / 1000000
	}
	return w.WriteJSON(path, roundedResult)
}
func (w *Writer) WriteManifest(path string, records []Record) error {
	return w.WriteJSON(path, records)
}
