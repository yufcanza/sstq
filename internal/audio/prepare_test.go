package audio

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestPrepare(t *testing.T) {
	tmpDir := t.TempDir()
	var fakeFFmpeg string
	var content string

	if runtime.GOOS == "windows" {
		fakeFFmpeg = filepath.Join(tmpDir, "ffmpeg.bat")
		content = `@echo off
copy %1 %~4
exit 0
`
	} else {
		fakeFFmpeg = filepath.Join(tmpDir, "ffmpeg")
		content = `#!/bin/sh
for arg; do :; done
cp "$1" "$arg"
exit 0
`
	}
	if err := os.WriteFile(fakeFFmpeg, []byte(content), 0755); err != nil {
		t.Fatalf("Ошибка создания ffmpeg: %v", err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	manifestPath := filepath.Join(tmpDir, "manifest.jsonl")
	manifestContent := `{"id":"test1","audio":"test1.wav","text":"привет"}
{"id":"test2","audio":"test2.wav","text":"мир"}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Ошибка создания манифеста: %v", err)
	}
	for i := 1; i <= 2; i++ {
		audioPath := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".wav")
		if err := os.WriteFile(audioPath, []byte("no-audio"), 0644); err != nil {
			t.Fatalf("Ошибка создания аудио: %v", err)
		}
	}
	cfg := PrepareConfig{
		ManifestPath: manifestPath,
		Profile:      "wav-16k",
		Workers:      2,
		Timeout:      10 * time.Second,
		OutDir:       filepath.Join(tmpDir, "output"),
	}

	results, err := Prepare(cfg)
	if err != nil {
		t.Fatalf("Ошибка Prepare: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Результатов = %d, хотели 2", len(results))
	}

	for _, r := range results {
		if r.Status == "error" {
			t.Errorf("Ошибка для %s: %s", r.ID, r.Error)
		}
	}
}
