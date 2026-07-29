package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Writer struct{}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteManifest(path string, records []Record) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Ошибка создания файла %s: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("Ошибка записи: %w", err)
		}
	}
	return nil
}

func writeJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("не удалось создать файл: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
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
			audioPath := filepath.Join(Dir, rec.Audio)
			if _, err := os.Stat(audioPath); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("corpus/manifest.jsonl:%d: record %q audio file %q не найден", lineNum, rec.ID, rec.Audio))
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
	fmt.Printf("Coprus is valid")
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
