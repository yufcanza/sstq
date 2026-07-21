package app

import (
	"fmt"
	"sttq/internal/corpus"
)

type App struct {
	manPath string
	hypPath string
	outPath string
	reader  *corpus.Reader
	writer  *corpus.Writer
}

func NewApp(manPath, hypPath, outPath string) *App {
	return &App{
		manPath: manPath,
		hypPath: hypPath,
		outPath: outPath,
		reader:  corpus.NewReader(),
		writer:  corpus.NewWriter(),
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

	return nil
}
