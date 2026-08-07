package report

import "sttq/internal/metrics"

const FormatVersion = 1

type Report struct {
	FormatVersion int           `json:"format_version"`
	Summary       Summary       `json:"summary"`
	Groups        Groups        `json:"groups"`
	Records       []RecordEntry `json:"records"`
	Errors        []ErrorEntry  `json:"errors,omitempty"`
}

type Summary struct {
	TotalRecords      int     `json:"total_records"`
	SuccessfulResults int     `json:"successful_result"`
	EngineErrors      int     `json:"engine_errors"`
	MissingResults    int     `json:"missing_results"`
	Coverage          float64 `json:"coverage"`

	ReferenceWords int     `json:"reference_words"`
	Hits           int     `json:"hits"`
	Substitutions  int     `json:"substitutions"`
	Deletions      int     `json:"deletions"`
	Insertions     int     `json:"insertions"`
	WER            float64 `json:"wer"`
	// SubstitutionsCER int     `json:"substitutions_cer"`
	// DeletionsCER     int     `json:"deletions_cer"`
	// InsertionsCER    int     `json:"insertions_cer"`
	CER          float64 `json:"cer"`
	ExactMatches int     `json:"exact_matches"`

	AudioDurationMS   int64   `json:"audio_duration_ms"`
	RecognitionTimeMS int64   `json:"recognition_time_ms"`
	RTF               float64 `json:"rtf"`
}
type Groups struct {
	ByTag      map[string]GroupStats `json:"by_tag"`
	ByDuration map[string]GroupStats `json:"by_duration"`
}

type GroupStats struct {
	Samples           int     `json:"samples"`
	Hits              int     `json:"hits"`
	Substitutions     int     `json:"substitutions"`
	Deletions         int     `json:"deletions"`
	Insertions        int     `json:"insertions"`
	WER               float64 `json:"wer"`
	CER               float64 `json:"cer"`
	ExactMatches      int     `json:"exact_matches"`
	AudioDurationMS   int64   `json:"audio_duration_ms"`
	RecognitionTimeMS int64   `json:"recognition_time_ms"`
	RTF               float64 `json:"rtf"`
}
type RecordEntry struct {
	ID                  string                  `json:"id"`
	Reference           string                  `json:"reference"`
	Hypothesis          string                  `json:"hypothesis"`
	NormalizedReference string                  `json:"normalized_reference"`
	NormalizeHypothesis string                  `json:"normalized_hypothesis"`
	ReferenceWords      int                     `json:"reference_words"`
	Hits                int                     `json:"hits"`
	Substitutions       int                     `json:"substitutions"`
	Deletions           int                     `json:"deletions"`
	Insertions          int                     `json:"insertions"`
	WER                 float64                 `json:"wer"`
	CER                 float64                 `json:"cer"`
	ExactMatch          bool                    `json:"exact_match"`
	Tags                []string                `json:"tags,omitempty"`
	Alignment           []metrics.AlignmentItem `json:"alignment"`
}

type ErrorEntry struct {
	ID      string `json:"id,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CompareResult struct {
	BaselinePath string                `json:"-"`
	CurrentPath  string                `json:"-"`
	Summary      CompareSummary        `json:"summary"`
	ByTag        map[string]CompareTag `json:"by_tag"`
	Record       []CompareRecord       `json:"records"`
	NewErrors    []ErrorEntry          `json:"new_errors"`
	FixedErrors  []ErrorEntry          `json:"fixed_errors"`
	Status       string                `json:"status"`
}

type CompareSummary struct {
	BaselineWER      float64 `json:"baseline_wer"`
	CurrentWER       float64 `json:"current_wer"`
	WERdelta         float64 `json:"wer_delta"`
	MaxWERdelta      float64 `json:"max_wer_delta"`
	BaselineCER      float64 `json:"baseline_cer"`
	CurrentCER       float64 `json:"current_cer"`
	CERdelta         float64 `json:"cer_delta"`
	MaxCERdelta      float64 `json:"max_cer_delta"`
	BaselineCoverage float64 `json:"baseline_coverage"`
	CurrentCoverage  float64 `json:"current_coverage"`
	CoverageDelta    float64 `json:"coverage_delta"`
	Status           string  `json:"status"`
}

type CompareTag struct {
	BaselineWER float64 `json:"baseline_wer"`
	CurrentWER  float64 `json:"current_wer"`
	WERdelta    float64 `json:"wer_delta"`
}

type CompareRecord struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	BaselineWER float64 `json:"baseline_wer,omitempty"`
	CurrentWER  float64 `json:"current_wer,omitempty"`
	WERdelta    float64 `json:"wer_delta,omitempty"`
	BaselineCER float64 `json:"baseline_cer,omitempty"`
	CurrentCER  float64 `json:"current_cer,omitempty"`
	CERdelta    float64 `json:"cer_delta,omitempty"`
}
