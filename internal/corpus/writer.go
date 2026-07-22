package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	Substitution        int                     `json:"substitutions"`
	Detetions           int                     `json:"deletions"`
	Insertion           int                     `jsin:"insertions"`
	WER                 float64                 `json:"wer"`
	CER                 float64                 `json:"cer"`
	ExactMath           bool                    `json:"exact_math"`
	Tags                []string                `json:"tags,omitempty"`
	Alignment           []metrics.AlignmentItem `json:"alignment"`
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
