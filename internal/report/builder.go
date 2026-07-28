package report

import (
	"math"
	"sttq/internal/corpus"
)

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(results []corpus.Result, errors []ErrorEntry) Report {
	if len(results) == 0 {
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
	records := make([]RecordEntry, len(results))

	for i, res := range results {
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

	summary := b.calculateSummary(results, errors)
	groups := b.calculateGroups(results)

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
	for _, res := range results {
		if res.Hypothesis != "" {
			summary.SuccessfulResults++
		}
		summary.ReferenceWords += res.ReferenceWords
		summary.Hits += res.Hits
		summary.Substitutions += res.Substitutions
		summary.Deletions += res.Deletions
		summary.Insertions += res.Insertions

		if res.ExactMatch {
			summary.ExactMatches++
		}
		if res.DurationMS > 0 {
			totalAudioDuration += res.DurationMS
			totalRecognitionTime += res.DurationMS // пока заглушка
		}
	}
	for _, err := range errors {
		if err.Message != "" {
			summary.EngineErrors++
			summary.MissingResults++
		}
	}
	totalErrors := summary.Substitutions + summary.Deletions + summary.Insertions
	if summary.ReferenceWords > 0 {
		summary.WER = float64(totalErrors) / float64(summary.ReferenceWords)
	}
	totalChars := 0

	for _, res := range results {
		totalChars += len([]rune(res.NormalizedReference))
	}
	if totalChars > 0 {
		summary.CER = float64(totalErrors) / float64(totalChars)
	}
	if summary.TotalRecords > 0 {
		summary.Coverage = float64(summary.SuccessfulResults) / float64(summary.TotalRecords)
	}
	summary.AudioDurationMS = totalAudioDuration
	summary.RecognitionTimeMS = totalRecognitionTime
	if totalAudioDuration > 0 {
		summary.RTF = float64(totalRecognitionTime) / float64(totalAudioDuration)
	}

	return summary
}

func (b *Builder) calculateGroups(results []corpus.Result) Groups {
	byTag := make(map[string]GroupStats)
	byDuration := make(map[string]GroupStats)

	byDuration["short"] = GroupStats{}
	byDuration["medium"] = GroupStats{}
	byDuration["long"] = GroupStats{}
	var group string
	for _, res := range results {

		for _, tag := range res.Tags {
			stats := byTag[tag]
			stats.Samples++
			stats.Hits += res.Hits
			stats.Substitutions += res.Substitutions
			stats.Deletions += res.Deletions
			stats.Insertions += res.Insertions
			if res.ExactMatch {
				stats.ExactMatches++
			}
			if res.DurationMS > 0 {
				stats.AudioDurationMS += res.DurationMS
			}
			byTag[tag] = stats
		}

		if res.ReferenceWords <= 5 {
			group = "short"
		} else if res.ReferenceWords <= 15 {
			group = "medium"
		} else {
			group = "large"
		}

		stats := byDuration[group]
		stats.Samples++
		stats.Hits += res.Hits
		stats.Substitutions += res.Substitutions
		stats.Deletions += res.Deletions
		stats.Insertions += res.Insertions
		if res.ExactMatch {
			stats.ExactMatches++
		}
		if res.DurationMS > 0 {
			stats.AudioDurationMS += res.DurationMS
		}
		byDuration[group] = stats

	}

	for tag, stats := range byTag {
		totalErrors := stats.Substitutions + stats.Deletions + stats.Insertions
		totalWords, totalChars := 0, 0
		for _, res := range results {
			for _, t := range res.Tags {
				if t == tag {
					totalWords += res.ReferenceWords
					totalChars += len([]rune(res.NormalizedReference))
					break
				}
			}
		}
		if totalWords > 0 {
			stats.WER = float64(totalErrors) / float64(totalWords)
		}
		if totalChars > 0 {
			stats.WER = float64(totalErrors) / float64(totalChars)
		}

		if stats.AudioDurationMS > 0 {
			stats.RTF = 1.0 // пока заглушка
		}
		byTag[tag] = stats

		for group, stats := range byDuration {
			totalErrors := stats.Substitutions + stats.Deletions + stats.Insertions
			totalWords, totalChars := 0, 0
			for _, res := range results {
				var g string
				if res.ReferenceWords <= 5 {
					g = "short"
				} else if res.ReferenceWords <= 15 {
					g = "medium"
				} else {
					g = "long"
				}
				if g == group {
					totalWords += res.ReferenceWords
					totalChars += len([]rune(res.NormalizedReference))
				}
			}
			if totalWords > 0 {
				stats.WER = float64(totalErrors) / float64(totalWords)
			}
			if totalChars > 0 {
				stats.CER = float64(totalErrors) / float64(totalChars)
			}
			if stats.AudioDurationMS > 0 {
				stats.RTF = 1.0
			}
			byDuration[group] = stats
		}
	}
	return Groups{
		ByTag:      byTag,
		ByDuration: byDuration,
	}
}
