package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type FakeRunner struct {
	Delay         time.Duration
	ForceTimeout  bool
	ExitCode      int
	Stderr        string
	Hypothesis    string
	Error         string
	FailOnIDs     map[string]bool
	ResumeSkipIDs map[string]bool
}
type FakeWhisperRunner struct {
	config  WhisperConfig
	fake    FakeRunner
	mu      sync.Mutex
	calls   []Task
	started map[string]bool
}

func NewFakeRunner(config WhisperConfig, fake FakeRunner) *FakeWhisperRunner {
	if fake.FailOnIDs == nil {
		fake.FailOnIDs = make(map[string]bool)
	}
	if fake.ResumeSkipIDs == nil {
		fake.ResumeSkipIDs = make(map[string]bool)
	}
	return &FakeWhisperRunner{
		config:  config,
		fake:    fake,
		started: make(map[string]bool),
	}
}
func (f *FakeWhisperRunner) Calls() []Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Task, len(f.calls))
	copy(out, f.calls)
	return out
}
func (f *FakeWhisperRunner) Run(ctx context.Context, task Task) Result {
	start := time.Now()
	f.mu.Lock()
	f.calls = append(f.calls, task)

	if f.config.Resume {
		if f.started[task.ID] || f.fake.ResumeSkipIDs[task.ID] {
			f.mu.Unlock()
			return Result{
				ID:              task.ID,
				Status:          "skipped",
				RecognitionTime: time.Since(start),
			}
		}
	}
	f.started[task.ID] = true
	f.mu.Unlock()

	delay := f.fake.Delay
	if delay == 0 {
		delay = 5 * time.Millisecond
	}
	select {
	case <-ctx.Done():
		return Result{
			ID:              task.ID,
			Status:          "error",
			Error:           "context canceled",
			RecognitionTime: time.Since(start),
		}
	case <-time.After(delay):
	}
	if f.fake.ForceTimeout {
		return Result{
			ID:              task.ID,
			Status:          "timeout",
			Error:           "Превышено время ожидания",
			RecognitionTime: time.Since(start),
		}
	}
	if ctx.Err() != nil {
		return Result{
			ID:              task.ID,
			Status:          "error",
			Error:           "context canceled",
			RecognitionTime: time.Since(start),
		}
	}
	if f.fake.Error != "" {
		return Result{
			ID:              task.ID,
			Status:          "error",
			Error:           f.fake.Error,
			RecognitionTime: time.Since(start),
		}
	}
	if f.fake.FailOnIDs[task.ID] {
		return Result{
			ID:              task.ID,
			Status:          "error",
			Error:           fmt.Sprintf("fake fail for id %s", task.ID),
			RecognitionTime: time.Since(start),
		}
	}
	if f.fake.ExitCode != 0 {
		errMsg := fmt.Sprintf("Ошибка выполнения: exit status %d", f.fake.ExitCode)
		if f.fake.Stderr != "" {
			errMsg += "\n" + strings.TrimSpace(f.fake.Stderr)
		}
		return Result{
			ID:              task.ID,
			Status:          "error",
			Error:           errMsg,
			RecognitionTime: time.Since(start),
		}
	}
	text := strings.TrimSpace(f.fake.Hypothesis)
	if text == "" && f.fake.Stderr != "" {
		for _, line := range strings.Split(f.fake.Stderr, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, "whisper_print_timings") ||
				strings.Contains(line, "system_info") ||
				strings.Contains(line, "loading model") ||
				strings.Contains(line, "Processing") ||
				strings.Contains(line, "Detecting language") ||
				strings.Contains(line, "Auto-detected") ||
				strings.Contains(line, "n_vocab") ||
				strings.Contains(line, "n_audio_ctx") ||
				strings.Contains(line, "n_mels") ||
				strings.Contains(line, "ftype") ||
				strings.Contains(line, "qntvr") ||
				strings.Contains(line, "type") {
				continue
			}
			text = line
			break
		}
	}
	if text == "" {
		text = fmt.Sprintf("Ненастоящая гипотеза %s", task.ID)
	}
	return Result{
		ID:              task.ID,
		Hypothesis:      strings.TrimSpace(text),
		Status:          "success",
		RecognitionTime: time.Since(start),
	}
}
