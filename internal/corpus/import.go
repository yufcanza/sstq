package corpus

import (
	"crypto/sha256"
	"encoding/hex"
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
	crowdManifest := filepath.Join(tmpDir, "test", "crowd", "manifest.jsonl")
	farfieldManifest := filepath.Join(tmpDir, "test", "farfield", "manifest.jsonl")
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
	farfieldRecords, err := parseManifest(farfieldData, "farfield", config.Seed)
	if err != nil {
		return nil, fmt.Errorf("Ошибка парсинга файла farfield: %v", err)
	}

	allRecords := append(crowdRecords, farfieldRecords...)

	selected := SelectRecord(allRecords, config.Quotas, config.Limit)

	audioDir := filepath.Join(config.OutDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, fmt.Errorf("Ошибка создания папки audio: %v", err)
	}

	var finalRecords []Record
	var totalDuration int64

	for _, rec := range selected {
		if config.MaxDuration > 0 && time.Duration(rec.Duration*float64(time.Second)) > config.MaxDuration {
			continue
		}
		filename := filepath.Base(rec.AudioFilepath)
		srcPath := filepath.Join(tmpDir, "test", rec.Domain, "files", filename)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}
		dstPath := filepath.Join(audioDir, rec.ID+".wav")
		if err := copyFile(srcPath, dstPath); err != nil {
			return nil, fmt.Errorf("Ошибка копирования аудио %s: %w", rec.ID, err)
		}

		sha256Hash, err := FindSHA256(dstPath)
		if err != nil {
			return nil, fmt.Errorf("Ошика вычисления sha256 для %s: %v", dstPath, err)
		}

		sampleRate := 16000
		channels := 1
		durationS := int64(rec.Duration * 1000)

		finalRecords = append(finalRecords, Record{
			ID:         rec.ID,
			Audio:      filepath.Join("audio", rec.ID+".wav"),
			Text:       rec.Text,
			Language:   "ru",
			Duration:   int(durationS),
			SampleRate: sampleRate,
			Channels:   channels,
			Tags:       []string{rec.Domain},
			SHA256:     sha256Hash,
		})
		totalDuration += durationS

	}

	sort.Slice(finalRecords, func(i, j int) bool {
		return finalRecords[i].ID < finalRecords[j].ID
	})

	manifestPath := filepath.Join(config.OutDir, "manifest.jsonl")
	if err := NewWriter().WriteManifest(manifestPath, finalRecords); err != nil {

	}

	summary := &ImportSummary{
		SourceRecords:    len(allRecords),
		SelectedRecord:   len(finalRecords),
		SelectedDuration: int(totalDuration),
		ByTag:            nil,
		Skipped:          map[string]int{},
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
			return string(recs[i].SortHash[:]) < string(recs[j].SortHash[:])
		})

		take := quota
		if len(recs) < take {
			take = len(recs)
		}
		selected = append(selected, recs[:take]...)

	}
	if limit > 0 && len(selected) > limit {
		sort.Slice(selected, func(i, j int) bool {
			return string(selected[i].SortHash[:]) < string(selected[j].SortHash[:])
		})
		selected = selected[:limit]
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].ID < selected[j].ID
	})

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

func FindSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
