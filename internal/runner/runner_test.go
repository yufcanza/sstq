package runner

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func collectResults(p *Pool, expected int) []Result {
	results := make([]Result, 0, expected)
	for res := range p.Results() {
		results = append(results, res)
		if len(results) == expected {
			break
		}
	}
	return results
}

// fake funner
func TestFakeSuccess(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Hypothesis: "привет мир",
	})

	pool := NewPool(2, r)
	pool.Start()

	pool.Submit(Task{ID: "1", Audio: "a.wav"})
	pool.Submit(Task{ID: "2", Audio: "b.wav"})
	pool.CloseTasks()

	results := collectResults(pool, 2)

	if len(results) != 2 {
		t.Fatalf("ожидалось 2 результата, получено %d", len(results))
	}
	for _, res := range results {
		if res.Status != "success" {
			t.Errorf("id=%s: ожидался success, получен %q", res.ID, res.Status)
		}
		if res.Hypothesis != "привет мир" {
			t.Errorf("id=%s: неверная гипотеза %q", res.ID, res.Hypothesis)
		}
	}
}

// timeout
func TestFakeTimeout(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		ForceTimeout: true,
	})

	pool := NewPool(1, r)
	pool.Start()

	pool.Submit(Task{ID: "1"})
	pool.CloseTasks()

	results := collectResults(pool, 1)
	if results[0].Status != "timeout" {
		t.Fatalf("ожидался timeout, получен %q", results[0].Status)
	}
}

// exit code
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

// strerr
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
		t.Fatalf("ожидался success, получен %q", res.Status)
	}
	if res.Hypothesis != "привет из stderr" {
		t.Fatalf("ожидалась гипотеза из stderr, получен %q", res.Hypothesis)
	}
}

// отмена
func TestFakeCancel(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Delay: 300 * time.Millisecond,
	})
	pool := NewPool(1, r)
	pool.Start()
	pool.Submit(Task{ID: "1"})
	go func() {
		time.Sleep(30 * time.Millisecond)
		pool.Stop()
	}()
	pool.CloseTasks()
	results := collectResults(pool, 1)
	if results[0].Status != "error" {
		t.Fatalf("ожидался error при отмене, получен %q", results[0].Status)
	}
}

// параллельность
func TestFakeParallel(t *testing.T) {
	const n = 20

	r := NewFakeRunner(WhisperConfig{}, FakeRunner{
		Delay: 20 * time.Millisecond,
	})

	pool := NewPool(4, r)
	pool.Start()

	start := time.Now()
	for i := 0; i < n; i++ {
		pool.Submit(Task{ID: fmt.Sprintf("p-%d", i)})
	}
	pool.CloseTasks()

	results := collectResults(pool, n)
	elapsed := time.Since(start)

	if len(results) != n {
		t.Fatalf("ожидалось %d результатов, получено %d", n, len(results))
	}

	// 4 воркера × 20ms → примерно 100ms, а не 400ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("параллельность не работает: заняло %v", elapsed)
	}
}

// resume
func TestFakeResume(t *testing.T) {
	r := NewFakeRunner(WhisperConfig{Resume: true}, FakeRunner{
		ResumeSkipIDs: map[string]bool{"1": true},
	})
	pool := NewPool(1, r)
	pool.Start()

	pool.Submit(Task{ID: "1"})
	pool.Submit(Task{ID: "2"})
	pool.CloseTasks()

	results := collectResults(pool, 2)

	var skipped, success int
	for _, res := range results {
		switch res.Status {
		case "skipped":
			skipped++
		case "success":
			success++
		}
	}

	if skipped != 1 {
		t.Errorf("ожидался 1 skipped, получено %d", skipped)
	}
	if success != 1 {
		t.Errorf("ожидался 1 success, получено %d", success)
	}
}
