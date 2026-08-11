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
setlocal EnableDelayedExpansion
set "out="
for %%a in (%*) do set "out=%%~a"
if not defined out exit /b 1
echo fake-audio> "!out!"
exit /b 0
`
	} else {
		fakeFFmpeg = filepath.Join(tmpDir, "ffmpeg")
		content = `#!/bin/sh
cp "$2" "${@: -1}"
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
		if r.Status != "ok" {
			t.Errorf("Статус для %s = %s, хотели ok", r.ID, r.Status)
		}
	}

	audioDir := filepath.Join(tmpDir, "output", "audio")
	for i := 1; i <= 2; i++ {
		expectedFile := filepath.Join(audioDir, "test"+string(rune('0'+i))+".wav")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Выходной файл не создан: %s", expectedFile)
		}
	}
	for i := 1; i <= 2; i++ {
		tmpFile := filepath.Join(audioDir, "test"+string(rune('0'+i))+".wav.tmp")
		if _, err := os.Stat(tmpFile); err == nil {
			t.Errorf("Временный файл не удален: %s", tmpFile)
		}
	}
	for i := 1; i <= 2; i++ {
		expectedFile := filepath.Join(audioDir, "test"+string(rune('0'+i))+".wav")
		info, err := os.Stat(expectedFile)
		if err != nil {
			continue
		}
		if info.Size() == 0 {
			t.Errorf("Выходной файл пустой: %s", expectedFile)
		}
	}
}
