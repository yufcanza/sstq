package corpus

import "time"

type Manifest struct {
	ID   string   `json:"id"`
	Text string   `json:"text"`
	Tags []string `json:"tags,omitempty"`
}
type Hypothesis struct {
	ID   string `json:"id"`
	Text string `json:"text"`
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
	SelectedDuration int            `json:"selected_duration_ms"`
	ByTag            map[string]int `json:"by_tag"`
	Skipped          map[string]int `json:"skipped"`
}
