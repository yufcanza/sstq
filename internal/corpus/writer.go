package corpus

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sttq/internal/atomicfile"
)

type Writer struct{}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteManifest(path string, records []Record) error {
	var buf bytes.Buffer

	for _, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("Ошибка маршалинга записи %s: %w", rec.ID, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	if err := atomicfile.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Ошибка записи манифеста: %w", err)
	}
	return nil
}

func writeJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("Ошибка маршалинга json: %w", err)
	}
	if err := atomicfile.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("Ошибка атомарной записи: %w", err)
	}
	return nil
}
func Statistic(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Ошибка открытия файла:%v", err)
	}
	defer file.Close()

	var (
		records  int
		duration int64
	)
	languages := make(map[string]int)
	tags := make(map[string]int)
	sampleRates := make(map[int]int)
	channels := make(map[int]int)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		records++
		duration += int64(rec.Duration)
		if rec.Language != "" {
			languages[rec.Language]++
		}
		for _, tag := range rec.Tags {
			tags[tag]++
		}
		sampleRates[rec.SampleRate]++
		channels[rec.Channels]++
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Ошибка чления файла:%v", err)
	}

	fmt.Printf("Records: %d\n", records)
	fmt.Printf("Duration: %s\n", formatDuration(duration))
	fmt.Printf("Languages: ")
	printMap(languages)
	fmt.Printf("Tags: ")
	printMap(tags)
	fmt.Printf("Sample rates: ")
	printIntMap(sampleRates)
	fmt.Printf("Channels: ")
	printIntMap(channels)
	return nil
}

func Validation(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("Ошибка открытия файла:%v", err)
	}
	defer file.Close()

	Dir := filepath.Dir(path)
	var errors []string
	seenID := make(map[string]int)
	lineNum := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		var rec Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: синтаксическая ошибка: %v", lineNum, err))
			continue
		}

		if rec.ID == "" {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: ID пустой", lineNum))
		}
		if rec.ID != "" {
			if prevLine, ok := seenID[rec.ID]; ok {
				errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: дубликат ID %q, впервые в строке %d", lineNum, rec.ID, prevLine))
			} else {
				seenID[rec.ID] = lineNum
			}
		}

		if rec.Audio == "" {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: запись %q: поле  audio пустое", lineNum, rec.ID))
		} else if strings.Contains(rec.Audio, "..") {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: audio path %q выходит за пределы каталога", lineNum, rec.ID, rec.Audio))
		} else {
			audioPath := filepath.Join(Dir, filepath.FromSlash(rec.Audio))
			if _, err := os.Stat(audioPath); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q audio file %q не найден", lineNum, rec.ID, rec.Audio))
			} else {
				audioInfo, err := Probe(audioPath)
				if err != nil {
					errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: ffprobe ошибка: %v", lineNum, rec.ID, err))
				} else {

					if audioInfo.SampleRate != rec.SampleRate {
						errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: sample_rate mismatch: expected %d, got %d",
							lineNum, rec.ID, rec.SampleRate, audioInfo.SampleRate))
					}

					if audioInfo.Channels != rec.Channels {
						errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: channels mismatch: expected %d, got %d",
							lineNum, rec.ID, rec.Channels, audioInfo.Channels))
					}
					diff := audioInfo.DurationMS - int64(rec.Duration)
					if diff < 0 {
						diff = -diff
					}
					if diff > 100 {
						errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: duration mismatch: expected %dms, got %dms",
							lineNum, rec.ID, rec.Duration, audioInfo.DurationMS))
					}
				}

				if rec.SHA256 != "" {
					hash, err := SHA256(audioPath)
					if err != nil {
						errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: ошибка вычисления SHA-256: %v", lineNum, rec.ID, err))
					} else if hash != rec.SHA256 {
						errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: SHA-256 mismatch: expected %s, got %s",
							lineNum, rec.ID, rec.SHA256, hash))
					}
				}
			}
		}

		if rec.Duration <= 0 {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: duration_ms должно быть > 0", lineNum, rec.ID))
		}
		if rec.SampleRate <= 0 {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: sample_rate должно быть > 0", lineNum, rec.ID))
		}
		if rec.Channels <= 0 {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: channels должно быть > 0", lineNum, rec.ID))
		}
		if rec.Language == "" {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: поле language пустое", lineNum, rec.ID))
		}
		if rec.Text == "" {
			errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q: поле text пустое", lineNum, rec.ID))
		}

	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("Ошибка чления файла:%v", err)
	}

	if len(errors) > 0 {
		for _, errMsg := range errors {
			fmt.Fprintln(os.Stderr, errMsg)
		}
		return false, nil
	}

	return true, nil
}
func formatDuration(ms int64) string {
	seconds := ms / 1000
	minutes := seconds / 60
	sec := seconds % 60

	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, sec)
	}
	return fmt.Sprintf("%ds", sec)
}

func printMap(m map[string]int) {
	first := true
	for k, v := range m {
		if !first {
			fmt.Printf(", ")
		}
		fmt.Printf("%s=%d", k, v)
		first = false
	}
	fmt.Println()
}
func printIntMap(m map[int]int) {
	first := true
	for k, v := range m {
		if !first {
			fmt.Printf(", ")
		}
		fmt.Printf("%d=%d", k, v)
		first = false
	}
	fmt.Println()
}

func SHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("открытие файла: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("чтение файла: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
