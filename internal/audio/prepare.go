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
	records, err := ReadRecords(config.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения манифеста: %w", err)
	}

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
					result := processRecord(ctx, rec, config, audioDir, ffmpegArgs)
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
	for res := range results {
		out = append(out, res)
	}

	return out, nil
}

func processRecord(ctx context.Context, rec corpus.Record, config PrepareConfig, audioDir string, ffArgs []string) Result {
	source := rec.Audio
	if source == "" {
		return Result{ID: rec.ID,
			Status: "error",
			Error:  "Нет пути к аудио",
		}
	}
	destination := filepath.Join(audioDir, rec.ID+".wav")
	tmp := destination + ".tmp"

	if canSkip(destination) {
		return Result{
			ID:     rec.ID,
			Status: "skipped",
		}

	}
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	args := []string{"i", source}
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

func canSkip(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}
