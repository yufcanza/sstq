package corpus

import (
	"sttq/internal/metrics"
	"time"
)

type Manifest struct {
	ID         string   `json:"id"`
	Text       string   `json:"text"`
	Tags       []string `json:"tags,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
}
type Hypothesis struct {
	ID                string `json:"id"`
	Text              string `json:"text"`
	RecognitionTimeMS int64  `json:"recognition_time"`
}
type GolosRecord struct {
	AudioFilePath string  `json:"audio_filepath"`
	Text          string  `json:"text"`
	Duration      float64 `json:"duration"`
}
type ProcessedRecord struct {
	Domain        string
	AudioFilepath string
	Text          string
	Duration      float64
	ID            string
	SortHash      [32]byte
}
type Record struct {
	ID         string   `json:"id"`
	Audio      string   `json:"audio"`
	Text       string   `json:"text"`
	Language   string   `json:"language"`
	Duration   int      `json:"duration_ms"`
	SampleRate int      `json:"sample_rate"`
	Channels   int      `json:"channels"`
	Tags       []string `json:"tags"`
	SHA256     string   `json:"sha256"`
}

type ImportConfig struct {
	ArchivePath string
	OutDir      string
	Limit       int
	MaxDuration time.Duration
	Seed        string
	Quotas      map[string]int
}
type ImportSummary struct {
	SourceRecords    int            `json:"source_records"`
	SelectedRecord   int            `json:"selected_records"`
	SelectedDuration int64          `json:"selected_duration_ms"`
	ByTag            map[string]int `json:"by_tag"`
	Skipped          map[string]int `json:"skipped"`
}
type SelectionInfo struct {
	Source         string         `json:"source"`
	Seed           string         `json:"string"`
	RequestRecords int            `json:"request_records"`
	MaxDuration    int64          `json:"max_duration_ms"`
	Quotas         map[string]int `json:"quotas"`
	SelectedIDs    []string       `json:"selected_ids"`
}
type Result struct {
	ID                  string   `json:"id"`
	Reference           string   `json:"reference"`
	Hypothesis          string   `json:"hypothesis"`
	NormalizedReference string   `json:"normalized_reference"`
	NormalizeHypothesis string   `json:"normalized_hypothesis"`
	ReferenceWords      int      `json:"reference_words"`
	Hits                int      `json:"hits"`
	Substitutions       int      `json:"substitutions"`
	Deletions           int      `json:"deletions"`
	Insertions          int      `json:"insertions"`
	WER                 float64  `json:"wer"`
	SubstitutionsCER    int      `json:"substitutions_cer"`
	DeletionsCER        int      `json:"deletions_cer"`
	InsertionsCER       int      `json:"insertions_cer"`
	CER                 float64  `json:"cer"`
	ExactMatch          bool     `json:"exact_match"`
	Tags                []string `json:"tags,omitempty"`
	DurationMS          int64    `json:"duration_ms,omitempty"`
	RecognitionTimeMS   int64    `json:"recognition_time_ms,omitempty"`

	Alignment []metrics.AlignmentItem `json:"alignment"`
	Error     string                  `json:"error,omitempty"`
}
