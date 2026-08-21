package corpus

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Reader struct {
}

func NewReader() *Reader {
	return &Reader{}
}

func (r *Reader) ReadManifest(path string) ([]Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Файл не найден: %s: %w", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	maxCapacity := 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var manifest []Manifest

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		var man Manifest
		err := json.Unmarshal([]byte(line), &man)
		if err != nil {
			return nil, fmt.Errorf("Ошибка обработки строки %d: %w", lineNum, err)
		}
		manifest = append(manifest, man)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Ошибка чтение файла: %w", err)
	}
	return manifest, nil
}

func (r *Reader) ReadHypotheses(path string) ([]Hypothesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Файл не найден %s: %w", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	maxCapacity := 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var hypotheses []Hypothesis

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		var hyp Hypothesis
		err := json.Unmarshal([]byte(line), &hyp)
		if err != nil {
			return nil, fmt.Errorf("Ошибка обработки строки %d: %w", lineNum, err)
		}
		hypotheses = append(hypotheses, hyp)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Ошибка чтение файла: %w", err)
	}
	return hypotheses, nil
}

func parseManifest(data []byte, domain string, seed string) ([]ProcessedRecord, int, int, error) {
	var records []ProcessedRecord
	var skipped int
	var invalid int

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var raw GolosRecord
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			invalid++
			continue
		}
		if strings.TrimSpace(raw.Text) == "" {
			skipped++
			continue
		}

		normalizedPath := filepath.ToSlash(raw.AudioFilePath)
		hashString := domain + ":" + normalizedPath
		hash := sha256.Sum256([]byte(hashString))
		id := fmt.Sprintf("%s-%x", domain, hash[:6])
		hashString = seed + ":" + domain + ":" + normalizedPath
		sortHash := sha256.Sum256([]byte(hashString))
		records = append(records, ProcessedRecord{
			Domain:        domain,
			AudioFilepath: raw.AudioFilePath,
			Text:          raw.Text,
			Duration:      raw.Duration,
			ID:            id,
			SortHash:      sortHash,
		})
	}
	return records, skipped, invalid, nil
}
