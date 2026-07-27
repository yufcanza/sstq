package app

import (
	"fmt"
	"strings"
	"sttq/internal/corpus"
	"sttq/internal/normalize"
)

type EvalApp struct {
	manPath     string
	hypPath     string
	outPath     string
	normProfile string
	reader      *corpus.Reader
	writer      *corpus.Writer
	normalizer  *normalize.Normalizer
}

func NewEvalApp(manPath, hypPath, outPath, normProfile string) *EvalApp {
	return &EvalApp{
		manPath:     manPath,
		hypPath:     hypPath,
		outPath:     outPath,
		normProfile: normProfile,
		reader:      corpus.NewReader(),
		writer:      corpus.NewWriter(),
		normalizer:  normalize.NewNormalizer(normProfile),
	}
}
func (a *EvalApp) Run() error {
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

type ImportApp struct {
	archivePath string
	quotas      string
	seed        string
	outPath     string
}

func NewImportApp(archivePath, quotas, seed, outPath string) *ImportApp {
	return &ImportApp{
		archivePath: archivePath,
		quotas:      quotas,
		seed:        seed,
		outPath:     outPath,
	}
}

func (a *ImportApp) Run() error {
	quotas := parseQuotas(a.quotas)
	config := corpus.ImportConfig{
		ArchivePath: a.archivePath,
		OutDir:      a.outPath,
		Limit:       250,
		MaxDuration: 30,
		Seed:        a.seed,
		Quotas:      quotas,
	}

	_, err := corpus.ImportGolos(config)
	if err != nil {
		return fmt.Errorf("Ошибка импорта: %v", err)
	}

	return nil
}

func parseQuotas(s string) map[string]int {
	quotas := make(map[string]int)
	if s == "" {
		return quotas
	}
	parts := strings.Split(s, ",")
	for _, part := range parts {
		k := strings.Split(part, "=")
		if len(k) == 2 {
			val := 0
			fmt.Sscanf(k[1], "%d", &val)
			quotas[k[0]] = val
		}
	}
	return quotas
}
