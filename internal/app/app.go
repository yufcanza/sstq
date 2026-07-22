package app

import (
	"fmt"
	"strings"
	"sttq/internal/corpus"
	"sttq/internal/metrics"
	"sttq/internal/normalize"
)

type App struct {
	manPath     string
	hypPath     string
	outPath     string
	normProfile string
	reader      *corpus.Reader
	writer      *corpus.Writer
	normalizer  *normalize.Normalizer
}

func NewApp(manPath, hypPath, outPath, normProfile string) *App {
	return &App{
		manPath:     manPath,
		hypPath:     hypPath,
		outPath:     outPath,
		normProfile: normProfile,
		reader:      corpus.NewReader(),
		writer:      corpus.NewWriter(),
		normalizer:  normalize.NewNormalizer(normProfile),
	}
}

func (a *App) Run() error {
	manifests, err := a.reader.ReadManifest(a.manPath)
	if err != nil {
		return fmt.Errorf("Ошибкк чтения эталонов: %v", err)
	}
	hypotheses, err := a.reader.ReadHypotheses(a.hypPath)
	if err != nil {
		return fmt.Errorf("Ошибкк чтения гипотез: %v", err)
	}

	result := a.evaluate(manifests, hypotheses)

	if a.outPath != "" {
		if err := a.writer.WriteResult(a.outPath, result); err != nil {
			return fmt.Errorf("Ошибка записи: %w", err)

		}
	}
	return nil

}

func (a *App) evaluate(manif []corpus.Manifest, hyps []corpus.Hypothesis) []corpus.Result {
	hypMap := make(map[string]corpus.Hypothesis)
	for _, h := range hyps {
		hypMap[h.ID] = h
	}
	var results []corpus.Result

	for _, man := range manif {
		hyp, exists := hypMap[man.ID]
		if !exists {
			continue
		}
		normalizedRef := a.normalizer.Normalize(man.Text)
		normalizedHyp := a.normalizer.Normalize(hyp.Text)

		tokenRef := strings.Fields(normalizedRef)
		tokenHyp := strings.Fields(normalizedHyp)

		werResult := metrics.CalculateWER(tokenRef, tokenHyp)
		cerResult := metrics.CalculateCER(normalizedRef, normalizedHyp)
		alignment := metrics.CreateAlignment(tokenRef, tokenHyp)

		result := corpus.Result{
			ID:                  man.ID,
			Reference:           man.Text,
			Hypothesis:          hyp.Text,
			NormalizedReference: normalizedRef,
			NormalizeHypothesis: normalizedHyp,
			ReferenceWords:      werResult.ManifestLen,
			Hits:                werResult.H,
			Substitution:        werResult.S,
			Detetions:           werResult.D,
			Insertion:           werResult.I,
			WER:                 werResult.Value,
			CER:                 cerResult.Value,
			ExactMath:           normalizedRef == normalizedHyp,
			Tags:                man.Tags,
			Alignment:           alignment.Items,
		}
		results = append(results, result)

	}
	return results
}
