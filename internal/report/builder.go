package report

import (
	"math"
	"sort"
	"strings"
	"sttq/internal/corpus"
)

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(results []corpus.Result) Report {
	var successful []corpus.Result
	var errors []ErrorEntry
	for _, res := range results {
		if res.Error != "" {
			errors = append(errors, ErrorEntry{
				ID:      res.ID,
				Code:    "no hypothesis",
				Message: res.Error,
			})
		} else {
			successful = append(successful, res)
		}
	}
	if len(successful) == 0 {
		return Report{
			FormatVersion: FormatVersion,
			Summary:       Summary{},
			Groups: Groups{
				ByTag:      make(map[string]GroupStats),
				ByDuration: make(map[string]GroupStats),
			},
			Records: []RecordEntry{},
			Errors:  errors,
		}
	}

	sort.Slice(successful, func(i, j int) bool {
		return successful[i].ID < successful[j].ID
	})
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].ID < errors[j].ID
	})

	records := make([]RecordEntry, len(successful))

	for i, res := range successful {
		records[i] = RecordEntry{
			ID:                  res.ID,
			Reference:           res.Reference,
			Hypothesis:          res.Hypothesis,
			NormalizedReference: res.NormalizedReference,
			NormalizeHypothesis: res.NormalizeHypothesis,
			ReferenceWords:      res.ReferenceWords,
			Hits:                res.Hits,
			Substitutions:       res.Substitutions,
			Deletions:           res.Deletions,
			Insertions:          res.Insertions,
			WER:                 math.Round(res.WER*1000000) / 1000000,
			CER:                 math.Round(res.CER*1000000) / 1000000,
			ExactMatch:          res.ExactMatch,
			Tags:                res.Tags,
			Alignment:           res.Alignment,
		}
	}

	summary := b.calculateSummary(successful, errors)
	groups := b.calculateGroups(successful)

	return Report{
		FormatVersion: FormatVersion,
		Summary:       summary,
		Groups:        groups,
		Records:       records,
		Errors:        errors,
	}
}

func (b *Builder) calculateSummary(results []corpus.Result, errors []ErrorEntry) Summary {
	summary := Summary{
		TotalRecords: len(results) + len(errors),
	}

	var (
		totalAudioDuration   int64
		totalRecognitionTime int64
	)
	SubstitutionsCER := 0
	DeletionsCER := 0
	InsertionsCER := 0
	for _, res := range results {
		summary.SuccessfulResults++
		summary.ReferenceWords += res.ReferenceWords
		summary.Hits += res.Hits
		summary.Substitutions += res.Substitutions
		summary.Deletions += res.Deletions
		summary.Insertions += res.Insertions
		SubstitutionsCER += res.SubstitutionsCER
		DeletionsCER += res.DeletionsCER
		InsertionsCER += res.InsertionsCER

		if res.ExactMatch {
			summary.ExactMatches++
		}
		if res.DurationMS > 0 {
			totalAudioDuration += res.DurationMS
		}
		if res.RecognitionTimeMS > 0 {
			totalRecognitionTime += res.RecognitionTimeMS
		}
	}
	for _, err := range errors {
		msg := strings.ToLower(err.Message)
		if strings.Contains(msg, "no hypothesis") || strings.Contains(msg, "missing") {
			summary.MissingResults++
		} else {
			summary.EngineErrors++
		}
	}
	totalErrors := summary.Substitutions + summary.Deletions + summary.Insertions
	totalErrorsChar := SubstitutionsCER + DeletionsCER + InsertionsCER
	if summary.ReferenceWords > 0 {
		summary.WER = math.Round(float64(totalErrors)/float64(summary.ReferenceWords)*100000) / 100000
	}
	totalChars := 0

	for _, res := range results {
		totalChars += len([]rune(res.NormalizedReference))
	}
	if totalChars > 0 {
		summary.CER = math.Round(float64(totalErrorsChar)/float64(totalChars)*100000) / 100000
	}
	if summary.TotalRecords > 0 {
		summary.Coverage = math.Round(float64(summary.SuccessfulResults)/float64(summary.TotalRecords)*100000) / 100000
	}
	summary.AudioDurationMS = totalAudioDuration
	summary.RecognitionTimeMS = totalRecognitionTime
	if totalAudioDuration > 0 {
		summary.RTF = math.Round(float64(totalRecognitionTime)/float64(totalAudioDuration)*100000) / 100000
	}

	return summary
}

func (b *Builder) calculateGroups(results []corpus.Result) Groups {
	byTag := make(map[string]GroupStats)
	byDuration := make(map[string]GroupStats)
	tagCER := make(map[string]struct {
		charErrors int
		charTotal  int
	})
	durationCER := make(map[string]struct {
		charErrors int
		charTotal  int
	})

	byDuration["short"] = GroupStats{}
	byDuration["medium"] = GroupStats{}
	byDuration["long"] = GroupStats{}
	var group string
	for _, res := range results {

		for _, tag := range res.Tags {
			cer := tagCER[tag]
			stats := byTag[tag]
			stats.Samples++
			stats.Hits += res.Hits
			stats.Substitutions += res.Substitutions
			stats.Deletions += res.Deletions
			stats.Insertions += res.Insertions
			cer.charErrors += res.SubstitutionsCER + res.DeletionsCER + res.InsertionsCER
			cer.charTotal += len([]rune(res.NormalizedReference))
			if res.ExactMatch {
				stats.ExactMatches++
			}
			if res.DurationMS > 0 {
				stats.AudioDurationMS += res.DurationMS
			}
			if res.RecognitionTimeMS > 0 {
				stats.RecognitionTimeMS += res.RecognitionTimeMS
			}
			tagCER[tag] = cer
			byTag[tag] = stats
		}

		if res.DurationMS <= 3000 {
			group = "short"
		} else if res.DurationMS <= 10000 {
			group = "medium"
		} else {
			group = "long"
		}

		stats := byDuration[group]
		cer := durationCER[group]
		stats.Samples++
		stats.Hits += res.Hits
		stats.Substitutions += res.Substitutions
		stats.Deletions += res.Deletions
		stats.Insertions += res.Insertions
		cer.charErrors += res.SubstitutionsCER + res.DeletionsCER + res.InsertionsCER
		cer.charTotal += len([]rune(res.NormalizedReference))
		if res.ExactMatch {
			stats.ExactMatches++
		}
		if res.DurationMS > 0 {
			stats.AudioDurationMS += res.DurationMS
		}
		if res.RecognitionTimeMS > 0 {
			stats.RecognitionTimeMS += res.RecognitionTimeMS
		}
		durationCER[group] = cer
		byDuration[group] = stats

	}

	for tag, stats := range byTag {
		totalErrors := stats.Substitutions + stats.Deletions + stats.Insertions
		totalWords := 0
		for _, res := range results {
			for _, t := range res.Tags {
				if t == tag {
					totalWords += res.ReferenceWords
					break
				}
			}
		}
		if totalWords > 0 {
			stats.WER = math.Round(float64(totalErrors)/float64(totalWords)*100000) / 100000
		}
		if cer, ok := tagCER[tag]; ok && cer.charTotal > 0 {
			stats.CER = math.Round(float64(cer.charErrors)/float64(cer.charTotal)*100000) / 100000
		}

		if stats.AudioDurationMS > 0 {
			stats.RTF = math.Round(float64(stats.RecognitionTimeMS)/float64(stats.AudioDurationMS)*100000) / 100000
		}
		byTag[tag] = stats
	}
	for group, stats := range byDuration {
		totalErrors := stats.Substitutions + stats.Deletions + stats.Insertions
		totalWords := 0
		for _, res := range results {
			var g string
			if res.DurationMS <= 3000 {
				g = "short"
			} else if res.DurationMS <= 10000 {
				g = "medium"
			} else {
				g = "long"
			}
			if g == group {
				totalWords += res.ReferenceWords
			}
		}
		if totalWords > 0 {
			stats.WER = math.Round(float64(totalErrors)/float64(totalWords)*100000) / 100000
		}
		if cer, ok := durationCER[group]; ok && cer.charTotal > 0 {
			stats.CER = math.Round(float64(cer.charErrors)/float64(cer.charTotal)*100000) / 100000
		}
		if stats.AudioDurationMS > 0 && stats.RecognitionTimeMS > 0 {
			stats.RTF = math.Round(float64(stats.RecognitionTimeMS)/float64(stats.AudioDurationMS)*100000) / 100000
		}
		byDuration[group] = stats
	}
	durationOrder := []string{"short", "medium", "long"}
	var tagNames []string
	for tag := range byTag {
		tagNames = append(tagNames, tag)
	}
	sort.Strings(tagNames)

	var durationNames []string
	for _, name := range durationOrder {
		if _, exists := byDuration[name]; exists {
			durationNames = append(durationNames, name)
		}
	}
	sortedByTag := make(map[string]GroupStats)
	for _, tag := range tagNames {
		sortedByTag[tag] = byTag[tag]
	}

	sortedByDuration := make(map[string]GroupStats)
	for _, name := range durationNames {
		sortedByDuration[name] = byDuration[name]
	}

	return Groups{
		ByTag:      sortedByTag,
		ByDuration: sortedByDuration,
	}
}
