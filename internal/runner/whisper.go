package runner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type WhisperConfig struct {
	BinaryPath string
	ModelPath  string
	Language   string
	Timeout    time.Duration
	Workers    int
	Resume     bool
	OutputPath string
}
type WhisperRunner struct {
	config WhisperConfig
}

func NewWhisperRunner(config WhisperConfig) *WhisperRunner {
	return &WhisperRunner{
		config: config,
	}
}

func (w *WhisperRunner) Run(ctx context.Context, task Task) Result {
	start := time.Now()
	if _, err := os.Stat(w.config.BinaryPath); os.IsNotExist(err) {
		return Result{
			ID:     task.ID,
			Status: "error",
			Error:  fmt.Sprintf("Ошибка: бинарник не найден: %s", w.config.BinaryPath),
		}
	}
	if _, err := os.Stat(w.config.ModelPath); os.IsNotExist(err) {
		return Result{
			ID:     task.ID,
			Status: "error",
			Error:  fmt.Sprintf("Ошибка: модель не найдена: %s", w.config.ModelPath),
		}
	}

	if _, err := os.Stat(task.Audio); os.IsNotExist(err) {
		return Result{
			ID:     task.ID,
			Status: "error",
			Error:  fmt.Sprintf("Ошибка: аудио не найдено: %s", task.Audio),
		}
	}
	ctx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	args := []string{
		"-m", w.config.ModelPath,
		"-f", task.Audio,
		"-l", w.config.Language,
		"--no-timestamps",
		"--output-txt",
	}

	cmd := exec.CommandContext(ctx, w.config.BinaryPath, args...)

	strout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{
			ID:              task.ID,
			Error:           fmt.Sprintf("Ошибка strout: %v", err),
			Status:          "error",
			RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
		}
	}
	defer strout.Close()
	strerr, err := cmd.StderrPipe()
	if err != nil {
		return Result{
			ID:              task.ID,
			Error:           fmt.Sprintf("Ошибка strerr:%v", err),
			Status:          "error",
			RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
		}
	}
	defer strerr.Close()

	if err := cmd.Start(); err != nil {
		return Result{
			ID:              task.ID,
			Error:           fmt.Sprintf("Ошибка запуска: %v", err),
			Status:          "error",
			RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
		}
	}

	stroutScanner := bufio.NewScanner(strout)
	var hypothesis strings.Builder
	for stroutScanner.Scan() {
		line := stroutScanner.Text()
		if !strings.Contains(line, "whisper_print_timings") &&
			!strings.Contains(line, "system_info") &&
			!strings.Contains(line, "loading model") &&
			!strings.Contains(line, "Processing") &&
			!strings.Contains(line, "Detecting language") &&
			strings.TrimSpace(line) != "" {
			hypothesis.WriteString(line)
			hypothesis.WriteString(" ")
		}
	}

	if err := stroutScanner.Err(); err != nil {
		return Result{
			ID:              task.ID,
			Error:           fmt.Sprintf("Ошибка чтения вывода: %v", err),
			Status:          "error",
			RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
		}
	}
	var strerrOutput strings.Builder
	strerrScanner := bufio.NewScanner(strerr)
	for strerrScanner.Scan() {
		line := strerrScanner.Text()
		strerrOutput.WriteString(line)
		strerrOutput.WriteString("\n")
		if !strings.Contains(line, "whisper_print_timings") &&
			!strings.Contains(line, "system_info") &&
			!strings.Contains(line, "loading model") &&
			!strings.Contains(line, "Processing") &&
			!strings.Contains(line, "Detecting language") &&
			!strings.Contains(line, "Auto-detected") &&
			!strings.Contains(line, "n_vocab") &&
			!strings.Contains(line, "n_audio_ctx") &&
			!strings.Contains(line, "n_mels") &&
			!strings.Contains(line, "ftype") &&
			!strings.Contains(line, "qntvr") &&
			!strings.Contains(line, "type") &&
			strings.TrimSpace(line) != "" {
			if hypothesis.Len() == 0 {
				hypothesis.WriteString(line)
				hypothesis.WriteString(" ")
			}
		}
	}
	if err := strerrScanner.Err(); err != nil {
		return Result{
			ID:              task.ID,
			Error:           fmt.Sprintf("Ошибка чтения strerr: %v", err),
			Status:          "error",
			RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Result{
				ID:              task.ID,
				Error:           "Превышено время ожидания",
				Status:          "timeout",
				RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
			}
		}
		return Result{
			ID:              task.ID,
			Error:           fmt.Sprintf("Ошибка выполнения: %v", err),
			Status:          "error",
			RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
		}
	}
	text := strings.TrimSpace(hypothesis.String())
	if text == "" && strerrOutput.Len() > 0 {
		lines := strings.Split(strerrOutput.String(), "\n")
		for _, line := range lines {
			if !strings.Contains(line, "whisper_print_timings") &&
				!strings.Contains(line, "system_info") &&
				!strings.Contains(line, "loading model") &&
				!strings.Contains(line, "Processing") &&
				!strings.Contains(line, "Detecting language") &&
				!strings.Contains(line, "Auto-detected") &&
				!strings.Contains(line, "n_vocab") &&
				!strings.Contains(line, "n_audio_ctx") &&
				!strings.Contains(line, "n_mels") &&
				!strings.Contains(line, "ftype") &&
				!strings.Contains(line, "qntvr") &&
				!strings.Contains(line, "type") &&
				strings.TrimSpace(line) != "" {
				text = strings.TrimSpace(line)
				break
			}

		}
	}
	return Result{
		ID:              task.ID,
		Hypothesis:      text,
		Status:          "success",
		RecognitionTime: time.Duration(time.Since(start).Milliseconds()),
	}
}
