package corpus

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func ImportGolos(config ImportConfig) (*ImportSummary, error) {
	tmpDir, err := os.MkdirTemp("", "golos-import-*")
	if err != nil {
		return nil, fmt.Errorf("Ошибка создания временной папки:%v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ExtractTar(config.ArchivePath, tmpDir); err != nil {
		return nil, fmt.Errorf("Ошика распаковки архива: %v", err)
	}
	crowdManifest := filepath.Join(tmpDir, "crowd", "manifest.jsonl")
	farfieldManifest := filepath.Join(tmpDir, "farfield", "manifest.jsonl")
	crowdData, err := os.ReadFile(crowdManifest)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения crowd/manifest.jsonl: %w", err)
	}
	farfieldData, err := os.ReadFile(farfieldManifest)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения farfield/manifest.jsonl: %w", err)
	}
	crowdRecords, err := parseManifest(crowdData, "crowd", config.Seed)
	if err != nil {
		return nil, fmt.Errorf("Ошибка парсинга файла crowd: %v", err)
	}
	farfieldRecords, err := parseManifest(farfieldData, "crowd", config.Seed)
	if err != nil {
		return nil, fmt.Errorf("Ошибка парсинга файла farfield: %v", err)
	}

	allRecords := append(crowdRecords, farfieldRecords...)

	selected := SelectRecord(allRecords, config.Quotas, config.Limit)

	audioDir := filepath.Join(config.OutDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, fmt.Errorf("Ошибка создания папки audio: %v", err)
	}

	for _, rec := range selected {
		if config.MaxDuration > 0 && time.Duration(rec.Duration*float64(time.Second)) > config.MaxDuration {
			continue
		}
		filename := filepath.Base(rec.AudioFilepath)
		srcPath := filepath.Join(tmpDir, rec.Domain, "files", filename)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}
		dstPath := filepath.Join(audioDir, rec.ID+".wav")
		if err := copyFile(srcPath, dstPath); err != nil {
			return nil, fmt.Errorf("Ошибка копирования аудио %s: %w", rec.ID, err)
		}

	}

	summary := &ImportSummary{
		SourceRecords:    0,
		SelectedRecord:   0,
		SelectedDuration: 0,
		ByTag:            nil,
		Skipped:          nil,
	}
	return summary, nil

}

func SelectRecord(records []ProcessedRecord, quotas map[string]int, limit int) []ProcessedRecord {
	byDomain := make(map[string][]ProcessedRecord)
	for _, rec := range records {
		byDomain[rec.Domain] = append(byDomain[rec.Domain], rec)
	}

	var selected []ProcessedRecord

	for domain, quota := range quotas {
		recs := byDomain[domain]
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].ID < recs[j].ID
		})

		take := quota
		if len(recs) < take {
			for i := 0; i < take; i++ {
				selected = append(selected, recs[i])
			}
		}

	}
	return selected
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}
