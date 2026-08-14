package corpus

import (
	"strings"
	"sttq/internal/metrics"
	"sttq/internal/normalize"
)

func Evaluate(manif []Manifest, hyps []Hypothesis, normalizer *normalize.Normalizer) []Result {
	hypMap := make(map[string]Hypothesis)
	for _, h := range hyps {
		hypMap[h.ID] = h
	}
	var results []Result

	for _, man := range manif {
		hyp, exists := hypMap[man.ID]
		if !exists {
			results = append(results, Result{
				ID:         man.ID,
				Error:      "no hypothesis found",
				Tags:       man.Tags,
				DurationMS: man.DurationMS,
				Reference:  man.Text,
			})
			continue
		}
		if hyp.Error != "" || strings.EqualFold(hyp.Status, "error") || strings.EqualFold(hyp.Status, "timeout") {
			errMsg := hyp.Error
			if errMsg == "" {
				errMsg = "engine error"
				if hyp.Status != "" {
					errMsg = "engine error: " + hyp.Status
				}
			}
			results = append(results, Result{
				ID:                man.ID,
				Reference:         man.Text,
				Hypothesis:        hyp.Text,
				Error:             errMsg,
				Tags:              man.Tags,
				DurationMS:        man.DurationMS,
				RecognitionTimeMS: hyp.RecognitionTimeMS,
			})
			continue
		}
		//fmt.Printf(" Profile: %v", a.normProfile)
		normalizedRef := normalizer.Normalize(man.Text)
		normalizedHyp := normalizer.Normalize(hyp.Text)

		tokenRef := strings.Fields(normalizedRef)
		tokenHyp := strings.Fields(normalizedHyp)

		werResult := metrics.CalculateWER(tokenRef, tokenHyp)
		cerResult := metrics.CalculateCER(normalizedRef, normalizedHyp)
		alignment := metrics.CreateAlignment(tokenRef, tokenHyp)

		result := Result{
			ID:                  man.ID,
			Reference:           man.Text,
			Hypothesis:          hyp.Text,
			NormalizedReference: normalizedRef,
			NormalizeHypothesis: normalizedHyp,
			ReferenceWords:      werResult.ManifestLen,
			Hits:                werResult.H,
			Substitutions:       werResult.S,
			Deletions:           werResult.D,
			Insertions:          werResult.I,
			WER:                 werResult.Value,
			SubstitutionsCER:    cerResult.S,
			DeletionsCER:        cerResult.D,
			InsertionsCER:       cerResult.I,
			CER:                 cerResult.Value,
			ExactMatch:          normalizedRef == normalizedHyp,
			DurationMS:          man.DurationMS,
			RecognitionTimeMS:   hyp.RecognitionTimeMS,
			Tags:                man.Tags,
			Alignment:           alignment.Items,
			Error:               "",
		}
		results = append(results, result)

	}
	return results
}
