package audio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sttq/internal/corpus"
	"sync"
	"time"
)

type PrepareConfig struct {
	ManifestPath string
	Profile      string
	Workers      int
	Timeout      time.Duration
	OutDir       string
	FFmpegPath   string
}

type Result struct {
	ID     string
	Status string
	Error  string
}

func Prepare(config PrepareConfig) ([]Result, error) {
	isValid := map[string]bool{
		"wav-16k": true,
		"wav-8k":  true,
	}
	if !isValid[config.Profile] {
		return nil, fmt.Errorf("Неизвестный профлиль %s", config.Profile)
	}
	records, err := ReadRecords(config.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения манифеста: %w", err)
	}
	manifestDir := filepath.Dir(config.ManifestPath)
	audioDir := filepath.Join(config.OutDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, fmt.Errorf("Ошибка создания папки audio: %w", err)
	}
	var ffmpegArgs []string
	switch config.Profile {
	case "wav-16k":
		ffmpegArgs = []string{"-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le"}
	case "wav-8k":
		ffmpegArgs = []string{"-ac", "1", "-ar", "8000", "-c:a", "pcm_s16le"}
	default:
		ffmpegArgs = []string{"-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le"}
	}

	tasks := make(chan corpus.Record, len(records))
	results := make(chan Result, len(records))

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rec := range tasks {
				select {
				case <-ctx.Done():
					return
				default:
					result := processRecord(ctx, rec, config, manifestDir, audioDir, ffmpegArgs)
					results <- result
				}
			}
		}()
	}

	for _, m := range records {
		tasks <- m
	}
	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	var out []Result
	var okCount, skipCount, errorCount int
	var errorResults []Result

	for res := range results {
		out = append(out, res)
		switch res.Status {
		case "ok":
			okCount++
		case "skipped":
			skipCount++
		case "error":
			errorCount++
			errorResults = append(errorResults, res)
		}
	}
	fmt.Printf("\nСтатистика подготовки:\n")
	fmt.Printf("Готово:   %d\n", okCount)
	fmt.Printf("Пропущено: %d\n", skipCount)
	fmt.Printf("Ошибок:   %d\n", errorCount)
	if errorCount > 0 {
		for _, res := range errorResults {
			fmt.Printf("[%s] %s\n", res.ID, res.Error)
		}
		return out, fmt.Errorf("Подготовка завершена с %d ошибками", errorCount)
	}

	return out, nil
}

func processRecord(ctx context.Context, rec corpus.Record, config PrepareConfig, manifestDir, audioDir string, ffArgs []string) Result {
	source := filepath.Join(manifestDir, rec.Audio)
	if source == "" {
		return Result{ID: rec.ID,
			Status: "error",
			Error:  "Нет пути к аудио",
		}
	}
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return Result{ID: rec.ID,
			Status: "error",
			Error:  "Аудиофайл не найден",
		}
	}
	destination := filepath.Join(audioDir, rec.ID+".wav")
	tmpFile, err := os.CreateTemp(audioDir, rec.ID+"-*.wav")
	if err != nil {
		return Result{
			ID:     rec.ID,
			Status: "error",
			Error:  fmt.Sprintf("создание временного файла: %v", err),
		}
	}
	tmp := tmpFile.Name()
	tmpFile.Close()
	defer func() {
		if _, err := os.Stat(tmp); err == nil {
			os.Remove(tmp)
		}
	}()

	if canSkip(destination, rec.SHA256, config.Profile) {
		return Result{
			ID:     rec.ID,
			Status: "skipped",
		}

	}
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	args := []string{"-y", "-i", source}
	args = append(args, ffArgs...)
	args = append(args, tmp)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Result{
			ID:     rec.ID,
			Status: "error",
			Error:  err.Error(),
		}
	}
	if err := cmd.Start(); err != nil {
		return Result{
			ID:     rec.ID,
			Status: "error",
			Error:  err.Error(),
		}
	}
	errOutput, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		os.Remove(tmp)
		if ctx.Err() == context.DeadlineExceeded {
			return Result{
				ID:     rec.ID,
				Status: "error",
				Error:  "Таймаут:" + string(errOutput),
			}

		}
		return Result{
			ID:     rec.ID,
			Status: "error",
			Error:  err.Error() + string(errOutput),
		}
	}

	if err := os.Rename(tmp, destination); err != nil {
		os.Remove(tmp)
		return Result{
			ID:     rec.ID,
			Status: "error",
			Error:  "Ошибка переименования: " + err.Error(),
		}
	}

	return Result{ID: rec.ID, Status: "ok"}
}

func ReadRecords(path string) ([]corpus.Record, error) {
	records, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer records.Close()
	var list []corpus.Record
	scanner := bufio.NewScanner(records)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec corpus.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, err
		}
		list = append(list, rec)

	}
	return list, scanner.Err()
}

func canSkip(path string, expSHA string, profile string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Size() == 0 {
		return false
	}
	actSHA, err := corpus.SHA256(path)
	if err != nil {
		return false
	}
	if actSHA != expSHA {
		return false
	}
	audioInfo, err := corpus.Probe(path)
	if err != nil {
		return false
	}
	switch profile {
	case "wav-16k":
		return audioInfo.SampleRate == 16000 && audioInfo.Channels == 1
	case "wav-8k":
		return audioInfo.SampleRate == 8000 && audioInfo.Channels == 1
	default:
		return false
	}

}
