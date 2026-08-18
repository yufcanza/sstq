package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sttq/internal/atomicfile"
	"sttq/internal/audio"
	"sttq/internal/corpus"
	"sttq/internal/normalize"
	"sttq/internal/report"
	"sttq/internal/runner"
	"sync"
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
		return fmt.Errorf("Ошибкк чтения эталонов: %w", err)
	}
	hypotheses, err := a.reader.ReadHypotheses(a.hypPath)
	if err != nil {
		return fmt.Errorf("Ошибкк чтения гипотез: %w", err)
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
	limit       int
	maxDuration time.Duration
}

func NewImportApp(archivePath string, quotas map[string]int, seed, outPath string, limit int, maxDuration time.Duration) *ImportApp {
	return &ImportApp{
		archivePath: archivePath,
		quotas:      quotas,
		seed:        seed,
		outPath:     outPath,
		limit:       limit,
		maxDuration: maxDuration,
	}
}

func (a *ImportApp) Run() error {
	config := corpus.ImportConfig{
		ArchivePath: a.archivePath,
		OutDir:      a.outPath,
		Limit:       a.limit,
		MaxDuration: a.maxDuration,
		Seed:        a.seed,
		Quotas:      a.quotas,
	}

	_, err := corpus.ImportGolos(config)
	if err != nil {
		return fmt.Errorf("Ошибка импорта: %w", err)
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
		return fmt.Errorf("Ошибка статистики: %w", err)
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
		return fmt.Errorf("Ошибка статистики: %w", err)
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
	result, err, errcode := report.Compare(a.baselinePath, a.currentPath, a.maxWER, a.maxCER)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сравнения: %v\n", err)
		return errcode
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
	fmt.Printf("Baseline coverage:	%.1f%%\n", result.Summary.BaselineCoverage*100)
	fmt.Printf("Current coverage:	%.1f%%\n", result.Summary.CurrentCoverage*100)
	fmt.Printf("Delta: 			%.1f%%\n", result.Summary.CoverageDelta*100)
	fmt.Println()
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Println()
	if len(result.ByTag) > 0 {
		for tag, stats := range result.ByTag {
			fmt.Printf("for tag - %v: baseline WER: %.4f, current WER: %.4f, delta: (%.4f)\n", tag, stats.BaselineWER, stats.CurrentWER, stats.WERdelta)
		}
		fmt.Println()
	}

	var improved, degraded []report.CompareRecord
	for _, r := range result.Record {
		switch r.Status {
		case "improved":
			improved = append(improved, r)
		case "degraded":
			degraded = append(degraded, r)
		}
	}
	if len(improved) > 0 || len(degraded) > 0 {
		fmt.Printf("Degraded count: %d\n", len(degraded))
		for _, r := range degraded {
			fmt.Printf(" %s: baseline WER: %.4f, current WER: %.4f, delta %.4f\n", r.ID, r.BaselineWER, r.CurrentWER, r.WERdelta)
		}
		fmt.Printf("Improved count: %d\n", len(improved))
		for _, r := range improved {
			fmt.Printf(" %s: baseline WER: %.4f, current WER: %.4f, delta %.4f\n", r.ID, r.BaselineWER, r.CurrentWER, r.WERdelta)
		}
		fmt.Println()
	}

	if len(result.NewErrors) > 0 {
		fmt.Printf("New errors count: %d\n", len(degraded))
		for _, r := range result.NewErrors {
			fmt.Printf("Error %s: %s\n", r.ID, r.Message)
		}
		fmt.Println()
	}
	if len(result.FixedErrors) > 0 {
		fmt.Printf("Fixed errors count: %d\n", len(degraded))
		for _, r := range result.FixedErrors {
			fmt.Printf("Error %s: %s\n", r.ID, r.Message)
		}
		fmt.Println()
	}

	var missing []report.CompareRecord
	for _, r := range result.Record {
		if r.Status == "missing" {
			missing = append(missing, r)
		}

	}

	if len(missing) > 0 {
		fmt.Printf("Missing count: %d\n", len(missing))
		for _, r := range missing {
			fmt.Printf("Id: %s, WER=%.4f\n", r.ID, r.BaselineWER)
		}
		println()
	}

	return errcode

}

type AudioPrepareApp struct {
	manifestPath string
	profile      string
	workers      int
	timeout      time.Duration
	outDir       string
}

func NewAudioPrepareApp(manifestPath, profile string, workers int, timeout time.Duration, outDir string) *AudioPrepareApp {
	return &AudioPrepareApp{
		manifestPath: manifestPath,
		profile:      profile,
		workers:      workers,
		timeout:      timeout,
		outDir:       outDir,
	}
}
func (a *AudioPrepareApp) Run() error {
	config := audio.PrepareConfig{
		ManifestPath: a.manifestPath,
		Profile:      a.profile,
		Workers:      a.workers,
		Timeout:      a.timeout,
		OutDir:       a.outDir,
	}
	results, err := audio.Prepare(config)
	if err != nil {
		return fmt.Errorf("Ошибка подготовки аудио: %w", err)
	}
	if len(results) == 0 {
		return fmt.Errorf("Результаты подготовки пустые")
	}
	return nil
}

type RunApp struct {
	manifestPath string
	binaryPath   string
	modelPath    string
	language     string
	workers      int
	timeout      time.Duration
	resume       bool
	outputPath   string
}

func NewRunApp(manifestPath, binaryPath, modelPath, language string, workers int, timeout time.Duration, resume bool, outputPath string) *RunApp {
	return &RunApp{
		manifestPath: manifestPath,
		binaryPath:   binaryPath,
		modelPath:    modelPath,
		language:     language,
		workers:      workers,
		timeout:      timeout,
		resume:       resume,
		outputPath:   outputPath,
	}
}
func (a *RunApp) Run() error {

	records, err := audio.ReadRecords(a.manifestPath)
	if err != nil {
		return fmt.Errorf("Ошибка чтения манифеста: %w", err)
	}
	resumeMgr, err := runner.NewResumeManager(a.outputPath)
	if err != nil {
		return fmt.Errorf("Ошибка создания менеджера возобновления: %w", err)
	}
	whisperRunner := runner.NewWhisperRunner(runner.WhisperConfig{
		BinaryPath: a.binaryPath,
		ModelPath:  a.modelPath,
		Language:   a.language,
		Timeout:    a.timeout,
	})
	var allResult []runner.Result
	if a.resume {
		existing, err := readExistingResult(a.outputPath)
		if err != nil {
			return fmt.Errorf("Ошибка чтения текущих результатов: %w", err)
		}
		allResult = append(allResult, existing...)
	}

	pool := runner.NewPool(a.workers, whisperRunner)
	pool.Start()

	done := make(chan struct{})
	var newResults []runner.Result
	var mu sync.Mutex
	go func() {
		for result := range pool.Results() {
			mu.Lock()
			newResults = append(newResults, result)
			mu.Unlock()
		}
		close(done)
	}()
	totalTasks := 0
	skipped := 0

	for _, rec := range records {
		if a.resume && resumeMgr.IsCompleted(rec.ID) {
			skipped++
			continue
		}
		totalTasks++
		audioPath := filepath.Join(filepath.Dir(a.manifestPath), filepath.FromSlash(rec.Audio))

		pool.Submit(runner.Task{
			ID:      rec.ID,
			Audio:   audioPath,
			Text:    rec.Text,
			Timeout: a.timeout,
		})
	}

	pool.CloseTasks()
	<-done
	pool.Stop()
	allResult = append(allResult, newResults...)
	sort.Slice(allResult, func(i, j int) bool {
		return allResult[i].ID < allResult[j].ID
	})
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range allResult {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("Oшибка сериализации результата %s: %w", r.ID, err)
		}
	}
	if err := atomicfile.WriteFile(a.outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Oшибка атомарной записи результатов: %w", err)
	}

	success := 0
	errors := 0
	timeouts := 0
	for _, r := range newResults {
		switch r.Status {
		case "success":
			success++
		case "error":
			errors++
		case "timeout":
			timeouts++
		}
	}
	fmt.Printf("Успешно:  %d\n", success)
	fmt.Printf("Ошибок:   %d\n", errors)
	fmt.Printf("Таймаут:  %d\n", timeouts)
	fmt.Printf("Результаты сохранены в: %s\n", a.outputPath)

	return nil
}

func readExistingResult(path string) ([]runner.Result, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var results []runner.Result
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r runner.Result
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, scanner.Err()
}
