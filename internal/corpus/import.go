package corpus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sttq/internal/atomicfile"
	"time"
)

func ImportGolos(config ImportConfig) (*ImportSummary, error) {
	tmpDir, err := os.MkdirTemp("", "golos-import-*")
	if err != nil {
		return nil, fmt.Errorf("Ошибка создания временной папки:%w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := ExtractTar(config.ArchivePath, tmpDir); err != nil {
		return nil, fmt.Errorf("Ошика распаковки архива: %w", err)
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
	crowdRecords, crowdSkip, crowdInvalid, err := parseManifest(crowdData, "crowd", config.Seed)
	if err != nil {
		return nil, fmt.Errorf("Ошибка парсинга файла crowd: %w", err)
	}
	farfieldRecords, farfieldSkip, farfieldInvalid, err := parseManifest(farfieldData, "farfield", config.Seed)
	if err != nil {
		return nil, fmt.Errorf("Ошибка парсинга файла farfield: %w", err)
	}

	allRecords := append(crowdRecords, farfieldRecords...)
	totalSkip := crowdSkip + farfieldSkip
	totalInvalid :=crowdInvalid+farfieldInvalid

	selected := SelectRecord(allRecords, config.Quotas, config.Limit)

	audioDir := filepath.Join(config.OutDir, "audio")
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return nil, fmt.Errorf("Ошибка создания папки audio: %w", err)
	}

	var finalRecords []Record
	var totalDuration int64
	var selected_ids []string
	var importErrors []string
	var durationErrors []string
	var missingAudioErrors int

	usedFiles := make(map[string]string)

	for _, rec := range selected {
		normalizedPath := filepath.ToSlash(rec.AudioFilepath)
		if existingID, exists := usedFiles[normalizedPath]; exists {
			importErrors = append(importErrors, fmt.Sprintf("Запись %s: дублирование аудиофайла %s (уже используется записью %s)",
				rec.ID, normalizedPath, existingID))
			continue
		}
		usedFiles[normalizedPath] = rec.ID

		if config.MaxDuration > 0 && time.Duration(rec.Duration*float64(time.Second)) > config.MaxDuration {
			importErrors = append(importErrors, fmt.Sprintf("Запись: %s, длительность: %.2f сек. максимальная длительность: %s", rec.ID, rec.Duration, config.MaxDuration))
			continue
		}
		filename := filepath.Base(rec.AudioFilepath)
		srcPath := filepath.Join(tmpDir, "test", rec.Domain, "files", filename)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			missingAudioErrors++
			importErrors = append(importErrors, fmt.Sprintf("Запись: %s, аудио файл %s не был найден", rec.ID, srcPath))
			continue
		}
		dstPath := filepath.Join(audioDir, rec.ID+".wav")
		if err := copyFile(srcPath, dstPath); err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Запись: %s, ошибка копирования: %s", rec.ID, err))
			continue
		}

		sha256Hash, err := FindSHA256(dstPath)
		if err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Запись: %s, ошибка SHA256: %s", rec.ID, err))
			os.Remove(dstPath)
			continue
		}
		audioInf, err := Probe(dstPath)
		if err != nil {
			importErrors = append(importErrors, fmt.Sprintf("Запись: %s, ошибка ffprobe: %s", rec.ID, err))
		}
		manifestDuration := int64(rec.Duration * 1000)
		realDuration := audioInf.DurationMS
		diff := realDuration - manifestDuration
		maxDiff := int64(float64(manifestDuration) * 0.1)
		if math.Abs(float64(diff)) > float64(maxDiff) && manifestDuration > 0 {
			durationErrors = append(durationErrors, fmt.Sprintf("Разница в длительности для %s: по манифесту: %dms, в реальности: %dms %.1f%%", rec.ID, manifestDuration, realDuration, float64(diff)/float64(manifestDuration)*100))
		}

		finalRecords = append(finalRecords, Record{
			ID:         rec.ID,
			Audio:      filepath.ToSlash(filepath.Join("audio", rec.ID+".wav")),
			Text:       rec.Text,
			Language:   "ru",
			Duration:   realDuration,
			SampleRate: audioInf.SampleRate,
			Channels:   audioInf.Channels,
			Tags:       []string{rec.Domain},
			SHA256:     sha256Hash,
		})
		selected_ids = append(selected_ids, rec.ID)
		totalDuration += audioInf.DurationMS

	}

	sort.Slice(finalRecords, func(i, j int) bool {
		return finalRecords[i].ID < finalRecords[j].ID
	})

	manifestPath := filepath.Join(config.OutDir, "manifest.jsonl")
	if err := NewWriter().WriteManifest(manifestPath, finalRecords); err != nil {

	}
	selectionPath := filepath.Join(config.OutDir, "selection.json")
	SelectionInfo := SelectionInfo{
		Source:         "golos-test",
		Seed:           config.Seed,
		RequestRecords: config.Limit,
		MaxDuration:    int64(config.MaxDuration / time.Millisecond),
		Quotas:         config.Quotas,
		SelectedIDs:    selected_ids,
	}
	if err := writeJSON(selectionPath, SelectionInfo); err != nil {
		return nil, fmt.Errorf("Ошибка записи selection.json: %w", err)
	}

	summary := &ImportSummary{
		SourceRecords:    len(allRecords),
		SelectedRecord:   len(finalRecords),
		SelectedDuration: totalDuration,
		ByTag:            countByTag(finalRecords),
		Skipped: map[string]int{
			"empty_text":        totalSkip,
			"missing_audio":     missingAudioErrors,
			"invalid_manifest": totalInvalid,
		},
		Errors:           importErrors,
		DurationWarnings: durationErrors,
	}
	summaryPath := filepath.Join(config.OutDir, "import-summary.json")
	if err := writeJSON(summaryPath, summary); err != nil {
		return nil, fmt.Errorf("Ошибка записи import-summary.json: %w", err)
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

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}
	if err := atomicfile.WriteReader(dst, srcFile, info.Mode().Perm()); err != nil {
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

func countByTag(records []Record) map[string]int {
	result := make(map[string]int)
	for _, rec := range records {
		for _, tag := range rec.Tags {
			result[tag]++
		}
	}
	return result
}
