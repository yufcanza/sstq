package app

import (
	"fmt"
	"os"
	"sttq/internal/corpus"
	"sttq/internal/normalize"
	"sttq/internal/report"
	"time"
)

type EvalApp struct {
	manPath     string
	hypPath     string
	outPath     string
	normProfile string
	reader      *corpus.Reader
	writer      *corpus.Writer
	normalizer  *normalize.Normalizer
}

func NewEvalApp(manPath, hypPath, outPath, normProfile string) *EvalApp {
	return &EvalApp{
		manPath:     manPath,
		hypPath:     hypPath,
		outPath:     outPath,
		normProfile: normProfile,
		reader:      corpus.NewReader(),
		writer:      corpus.NewWriter(),
		normalizer:  normalize.NewNormalizer(normProfile),
	}
}
func (a *EvalApp) Run() error {
	manifests, err := a.reader.ReadManifest(a.manPath)
	if err != nil {
		return fmt.Errorf("Ошибкк чтения эталонов: %v", err)
	}
	hypotheses, err := a.reader.ReadHypotheses(a.hypPath)
	if err != nil {
		return fmt.Errorf("Ошибкк чтения гипотез: %v", err)
	}

	result := corpus.Evaluate(manifests, hypotheses, a.normalizer)

	builder := report.NewBuilder()
	reportData := builder.Build(result)

	if a.outPath != "" {
		if err := report.Write(a.outPath, reportData); err != nil {
			return fmt.Errorf("Ошибка записи: %w", err)

		}
	}
	return nil

}

type ImportApp struct {
	archivePath string
	quotas      map[string]int
	seed        string
	outPath     string
}

func NewImportApp(archivePath string, quotas map[string]int, seed, outPath string) *ImportApp {
	return &ImportApp{
		archivePath: archivePath,
		quotas:      quotas,
		seed:        seed,
		outPath:     outPath,
	}
}

func (a *ImportApp) Run() error {
	config := corpus.ImportConfig{
		ArchivePath: a.archivePath,
		OutDir:      a.outPath,
		Limit:       250,
		MaxDuration: 30 * time.Minute,
		Seed:        a.seed,
		Quotas:      a.quotas,
	}

	_, err := corpus.ImportGolos(config)
	if err != nil {
		return fmt.Errorf("Ошибка импорта: %v", err)
	}

	return nil
}

type StatApp struct {
	manPath string
}

func NewStatsApp(manifest string) *StatApp {
	return &StatApp{
		manPath: manifest,
	}
}
func (a *StatApp) Run() error {
	err := corpus.Statistic(a.manPath)
	if err != nil {
		return fmt.Errorf("Ошибка статистики: %v", err)
	}
	return nil
}

type ValidateApp struct {
	manPath string
}

func NewValidateApp(manifest string) *ValidateApp {
	return &ValidateApp{
		manPath: manifest,
	}
}
func (a *ValidateApp) Run() error {
	valid, err := corpus.Validation(a.manPath)
	if err != nil {
		return fmt.Errorf("Ошибка статистики: %v", err)
	}
	if !valid {
		return fmt.Errorf("Corpus invalid")
	}
	fmt.Printf("Corpus is valid\n")

	return nil
}

type ReportApp struct {
	inputPath  string
	format     string
	outputPath string
}

func NewReportApp(inputPath, format, outputPath string) *ReportApp {
	return &ReportApp{
		inputPath:  inputPath,
		format:     format,
		outputPath: outputPath,
	}
}

func (a *ReportApp) Run() error {
	switch a.format {
	case "html":
		return report.WriteHTML(a.inputPath, a.outputPath)

	default:
		return fmt.Errorf("Ошибка формата: формат %s не поддрживается", a.format)
	}
}

type CompareApp struct {
	baselinePath string
	currentPath  string
	maxWER       float64
	maxCER       float64
}

func NewCompareApp(baselinePath, currentPath string, maxWER, maxCER float64) *CompareApp {
	return &CompareApp{
		baselinePath: baselinePath,
		currentPath:  currentPath,
		maxWER:       maxWER,
		maxCER:       maxCER,
	}
}

func (a *CompareApp) Run() int {
	result, err := report.Compare(a.baselinePath, a.currentPath, a.maxWER, a.maxCER)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сравнения: %v\n", err)
		return 2
	}
	fmt.Printf("Baseline WER:	 	%.4f\n", result.Summary.BaselineWER)
	fmt.Printf("Current WER:	 	%.4f\n", result.Summary.CurrentWER)
	fmt.Printf("Delta:			%.4f\n", result.Summary.WERdelta)
	fmt.Printf("Allowed:		%.4f\n", result.Summary.MaxWERdelta)
	fmt.Println()
	fmt.Printf("Baseline CER: 		%.4f\n", result.Summary.BaselineCER)
	fmt.Printf("Current CER: 		%.4f\n", result.Summary.CurrentCER)
	fmt.Printf("Delta:			%.4f\n", result.Summary.CERdelta)
	fmt.Printf("Allowed: 		%.4f\n", result.Summary.MaxCERdelta)
	fmt.Println()
	fmt.Printf("Status: %s\n", result.Status)

	if result.Status == "PASS" {
		return 0
	}
	return 1

}
