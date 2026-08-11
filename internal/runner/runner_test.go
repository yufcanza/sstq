package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

//fake funner
func TestFakeSuccess(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Hypothesis: "привет мир",
	})
	res := r.Run(context.Background(), Task{ID: "test-1", Audio: "a.wav"})
	if res.Status != "success" {
		t.Fatalf("Ожидаемый status success, got %q", res.Status)
	}
	if res.Hypothesis != "привет мир" {
		t.Fatalf("Ожидаемый hypothesis %q, got %q", "привет мир", res.Hypothesis)
	}
	if res.Error != "" {
		t.Fatalf("Ожидаемый empty error, got %q", res.Error)
	}
	if res.ID != "test-1" {
		t.Fatalf("Ожидаемый id test-1, got %q", res.ID)
	}
	if res.RecognitionTime <= 0 {
		t.Fatal("Ожидаемый positive RecognitionTime")
	}
}
//timeout
func TestFakeTimeout(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		ForceTimeout: true,
	})
	res := r.Run(context.Background(), Task{ID: "1"})

	if res.Status != "timeout" {
		t.Fatalf("expected status timeout, got %q", res.Status)
	}
	if res.Error != "Превышено время ожидания" {
		t.Fatalf("unexpected error: %q", res.Error)
	}
	if res.Hypothesis != "" {
		t.Fatalf("expected empty hypothesis on timeout")
	}
}
//exit code
func TestFakeExitCode(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		ExitCode: 1,
		Stderr:   "something went wrong",
	})
	res := r.Run(context.Background(), Task{ID: "1"})
	if res.Status != "error" {
		t.Fatalf("expected status error, got %q", res.Status)
	}
	if !strings.Contains(res.Error, "exit status 1") {
		t.Fatalf("expected exit status in error, got %q", res.Error)
	}
}
//strerr
func TestFakeStderrAsHypothesis(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Hypothesis: "", 
		Stderr: `loading model
system_info: ...
Processing audio...
Detecting language
привет из stderr
whisper_print_timings
`,
	})

	res := r.Run(context.Background(), Task{ID: "stderr-1"})

	if res.Status != "success" {
		t.Fatalf("expected success, got %q", res.Status)
	}
	if res.Hypothesis != "привет из stderr" {
		t.Fatalf("expected hypothesis from stderr, got %q", res.Hypothesis)
	}
}
//отмена
func TestFakeCancel(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Delay: 100 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	res := r.Run(ctx, Task{ID: "1"})
	if res.Status != "error" {
		t.Fatalf("expected error on cancel, got %q", res.Status)
	}
	if res.Error != "context canceled" {
		t.Fatalf("expected 'context canceled', got %q", res.Error)
	}

}
//параллельность
func TestFakeParallel(t *testing.T) {
	const n = 20

	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Delay: 20 * time.Millisecond,
	})

	var wg sync.WaitGroup
	results := make([]Result, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = r.Run(context.Background(), Task{
				ID: fmt.Sprintf("p-%d", idx),
			})
		}(i)
	}
	wg.Wait()

	calls := r.Calls()
	if len(calls) != n {
		t.Fatalf("expected %d calls, got %d", n, len(calls))
	}

	for i, res := range results {
		if res.Status != "success" {
			t.Errorf("task %d: expected success, got %q", i, res.Status)
		}
		if res.ID != fmt.Sprintf("p-%d", i) {
			t.Errorf("task %d: wrong id %q", i, res.ID)
		}
	}
}
//resume
func TestFakeResume(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{Resume: true}, FakeRunner{
		ResumeSkipIDs: map[string]bool{"1": true},
	})
	res := r.Run(context.Background(), Task{ID: "1"})
	if res.Status != "skipped" {
		t.Fatalf("expected skipped, got %q", res.Status)
	}
	if res.Hypothesis != "" {
		t.Fatalf("expected empty hypothesis on skip")
	}
}