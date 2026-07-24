package app

import (
	"fmt"
	"sttq/internal/corpus"
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

	result := corpus.Evaluate(manifests, hypotheses, a.normalizer)

	if a.outPath != "" {
		if err := a.writer.WriteResult(a.outPath, result); err != nil {
			return fmt.Errorf("Ошибка записи: %w", err)

		}
	}
	return nil

}
