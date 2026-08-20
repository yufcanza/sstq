package audio

import (
	"os"
	"path/filepath"
	"runtime"
	"sttq/internal/corpus"
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
	manifestContent := `{"id":"test1","audio":"audio/test1.wav","text":"привет","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["crowd"],"sha256":"old_sha1"}
						{"id":"test2","audio":"audio/test2.wav","text":"мир","duration_ms":2000,"sample_rate":16000,"channels":1,"tags":["crowd"],"sha256":"old_sha2"}
						{"id":"test3","audio":"audio/test3.wav","text":"тест","duration_ms":3000,"sample_rate":16000,"channels":1,"tags":["crowd"],"sha256":"old_sha3"}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Ошибка создания манифеста: %v", err)
	}
	audioDir := filepath.Join(tmpDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		t.Fatalf("Ошибка создания audio: %v", err)
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
		if err == nil {
			t.Error("Ошибка из за отсутствия test3 нет")
		}

		if len(results) != 3 {
			t.Errorf("Результатов = %d, хотели 3", len(results))
		}
		if len(results) == 3 {
			if results[0].ID != "test1" || results[1].ID != "test2" || results[2].ID != "test3" {
				t.Errorf("Недетерминированный порядок результатов = `%s, %s,%s` хотели `test1, test2, test2`", results[0].ID, results[1].ID, results[2].ID)
			}
		}
		var okCount, errorCount int
		for _, r := range results {
			switch r.Status {
			case "ok":
				okCount++
			case "error":
				errorCount++
			}
		}
		if okCount != 2 {
			t.Errorf("Успешно %d, хотели 2", okCount)
		}
		if errorCount != 1 {
			t.Errorf("Ошибок %d, хотели 1", errorCount)
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

		skipDir := t.TempDir()
		manifestPath2 := filepath.Join(skipDir, "manifest.jsonl")
		audioDir2 := filepath.Join(skipDir, "audio")
		if err := os.MkdirAll(audioDir2, 0755); err != nil {
			t.Fatalf("Ошибка создания audio: %v", err)
		}
		audioPath2 := filepath.Join(audioDir2, "test1.wav")
		if err := os.WriteFile(audioPath2, []byte("no-audio"), 0644); err != nil {
			t.Fatalf("Ошибка создания аудио: %v", err)
		}
		sha, err := corpus.FindSHA256(audioPath2)
		if err != nil {
			t.Fatalf("Ошибка sha: %v", err)
		}
		manifestContent2 := `{"id":"test1","audio":"audio/test1.wav","text":"привет","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["test"],"sha256":"` + sha + `"}`
		if err := os.WriteFile(manifestPath2, []byte(manifestContent2), 0644); err != nil {
			t.Fatalf("Ошибка создания манифеста: %v", err)
		}
		results1, err := Prepare(cfg)
		if err != nil {
			t.Logf("Первый запуск вернул ошибку: %v", err)
		}

		if len(results1) > 0 && results1[0].Status == "error" {
			t.Errorf("Первый запуск: ошибка %s", results1[0].Error)
		}
		results2, err := Prepare(cfg)
		if err != nil {
			t.Logf("Второй запуск вернул ошибку: %v", err)
		}

		// Проверяем, что во втором запуске файл либо пропущен, либо обработан
		if len(results2) > 0 && results2[0].Status == "error" {
			t.Errorf("Второй запуск: ошибка %s", results2[0].Error)
		}

		t.Logf("Первый запуск: %s, Второй запуск: %s",
			results1[0].Status, results2[0].Status)
	}
}
