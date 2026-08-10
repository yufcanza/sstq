package runner

import (
	"bufio"
	"encoding/json"
	"os"
)

type ResumeManager struct {
	outputPath string
	completed  map[string]bool
}

func NewResumeManager(outputPath string) (*ResumeManager, error) {
	rm := &ResumeManager{
		outputPath: outputPath,
		completed:  make(map[string]bool),
	}
	if _, err := os.Stat(outputPath); err == nil {
		if err := rm.loadCompleted(); err != nil {
			return nil, err
		}
	}
	return rm, nil
}
func (rm *ResumeManager) loadCompleted() error {
	file, err := os.Open(rm.outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var result Result
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		rm.completed[result.ID] = true
	}
	return scanner.Err()
}

func (rm *ResumeManager) IsCompleted(id string) bool {
	return rm.completed[id]
}
func (rm *ResumeManager) CompletedCount() int {
	return len(rm.completed)
}
