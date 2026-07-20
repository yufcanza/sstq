package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Reader struct {
}

func NewReader() *Reader {
	return &Reader{}
}

func (r *Reader) ReadManifest(path string) ([]Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Ошибка открытия файла %s: %v", path, err)
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
			return nil, fmt.Errorf("Ошибка обработки строки %d: %v", lineNum, err)
		}
		manifest = append(manifest, man)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Ошибка чтение файла%v", err)
	}
	return manifest, nil
}

func (r *Reader) ReadHypotheses(path string) ([]Hypothesis, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Ошибка открытия файла %s: %v", path, err)
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
			return nil, fmt.Errorf("Ошибка обработки строки %d: %v", lineNum, err)
		}
		hypotheses = append(hypotheses, hyp)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Ошибка чтение файла%v", err)
	}
	return hypotheses, nil
}
