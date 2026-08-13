package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

func Compare(baselinePath, currentPath string, maxWERdelta, maxCERdelta float64) (*CompareResult, error, int) {
	baseline, err := ReadReport(baselinePath)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения baseline: %w", err), 2
	}
	current, err := ReadReport(currentPath)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения current: %w", err), 2
	}

	baselineMap := make(map[string]RecordEntry)
	for _, r := range baseline.Records {
		baselineMap[r.ID] = r
	}
	currentMap := make(map[string]RecordEntry)
	for _, r := range current.Records {
		currentMap[r.ID] = r
	}
	baselineErrors := make(map[string]ErrorEntry)
	for _, r := range baseline.Errors {
		baselineErrors[r.ID] = r
	}
	currentErrors := make(map[string]ErrorEntry)
	for _, r := range current.Errors {
		currentErrors[r.ID] = r
	}

	result := &CompareResult{
		BaselinePath: baselinePath,
		CurrentPath:  currentPath,
		ByTag:        make(map[string]CompareTag),
	}

	result.Summary = CompareSummary{
		BaselineWER:      baseline.Summary.WER,
		CurrentWER:       current.Summary.WER,
		WERdelta:         current.Summary.WER - baseline.Summary.WER,
		MaxWERdelta:      maxWERdelta,
		BaselineCER:      baseline.Summary.CER,
		CurrentCER:       current.Summary.CER,
		CERdelta:         current.Summary.CER - baseline.Summary.CER,
		MaxCERdelta:      maxCERdelta,
		BaselineCoverage: baseline.Summary.Coverage,
		CurrentCoverage:  current.Summary.Coverage,
		CoverageDelta:    current.Summary.Coverage - baseline.Summary.Coverage,
	}
	for tag, baselineStats := range baseline.Groups.ByTag {
		currentStats := current.Groups.ByTag[tag]
		result.ByTag[tag] = CompareTag{
			BaselineWER: baselineStats.WER,
			CurrentWER:  currentStats.WER,
			WERdelta:    currentStats.WER - baselineStats.WER,
		}
	}
	var records []CompareRecord

	for id, currentRec := range currentMap {
		baselineRec, exist := baselineMap[id]
		if !exist {
			records = append(records, CompareRecord{
				ID:         id,
				Status:     "new",
				CurrentWER: currentRec.WER,
				CurrentCER: currentRec.CER,
			})
			continue
		}
		werDelta := currentRec.WER - baselineRec.WER
		cerDelta := currentRec.CER - baselineRec.CER

		status := "unchanged"
		if werDelta < 0 {
			status = "improved"
		} else if werDelta > 0 {
			status = "degraded"
		}

		records = append(records, CompareRecord{
			ID:          id,
			Status:      status,
			BaselineWER: baselineRec.WER,
			CurrentWER:  currentRec.WER,
			WERdelta:    werDelta,
			BaselineCER: baselineRec.CER,
			CurrentCER:  currentRec.CER,
			CERdelta:    cerDelta,
		})
	}
	for id := range baselineMap {
		if _, exists := currentMap[id]; !exists {
			records = append(records, CompareRecord{
				ID:          id,
				Status:      "missing",
				BaselineWER: baselineMap[id].WER,
			})
		}
	}

	var newErrors, fixedErrors []ErrorEntry

	for id, err := range currentErrors {
		if _, exists := baselineErrors[id]; !exists {
			newErrors = append(newErrors, err)
		}
	}
	for id, err := range baselineErrors {
		if _, exists := currentErrors[id]; !exists {
			fixedErrors = append(fixedErrors, err)
		}
	}

	result.Record = records
	result.NewErrors = newErrors
	result.FixedErrors = fixedErrors

	sort.Slice(result.Record, func(i, j int) bool {
		return result.Record[i].ID < result.Record[j].ID
	})

	sort.Slice(result.NewErrors, func(i, j int) bool {
		return result.NewErrors[i].ID < result.NewErrors[j].ID
	})

	sort.Slice(result.FixedErrors, func(i, j int) bool {
		return result.FixedErrors[i].ID < result.FixedErrors[j].ID
	})

	if result.Summary.WERdelta <= maxWERdelta && result.Summary.CERdelta <= maxCERdelta {
		result.Status = "PASS"
		return result, nil, 0
	} else {
		result.Status = "FAIL"
		return result, nil, 1
	}
}

func ReadReport(path string) (Report, error) {
	var report Report
	data, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	err = json.Unmarshal(data, &report)
	return report, err
}
