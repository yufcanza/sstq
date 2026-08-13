package report

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sttq/internal/corpus"
	"sttq/internal/normalize"
	"testing"
)

var update = flag.Bool("update", false, "Обновление golden")

func TestGoldenReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")

	manifestPath := filepath.Join("..", "..", "testdata", "golden", "manifest.jsonl")
	hypPath := filepath.Join("..", "..", "testdata", "golden", "hyphotheses.jsonl")

	manifests, err := corpus.NewReader().ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Ошибка чтения манифеста: %v", err)
	}
	hypotheses, err := corpus.NewReader().ReadHypotheses(hypPath)
	if err != nil {
		t.Fatalf("Ошибка чтения гипотез: %v", err)
	}

	normalizer := normalize.NewNormalizer("ru-default")
	result := corpus.Evaluate(manifests, hypotheses, normalizer)

	builder := NewBuilder()
	reportData := builder.Build(result)

	if err := Write(out, reportData); err != nil {
		t.Fatalf("Ошибка записи отчета: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("Ошибка чтения результата: %v", err)
	}
	goldenPath := filepath.Join("..", "..", "testdata", "control", "expected_report.json")

	if *update {
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("Ошибка обновления golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Ошибка чтения golden: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch")
		t.Logf(" got len: %d, want len: %d", len(got), len(want))
	}

}

func TestGoldenHTML(t *testing.T) {
	tmpD := t.TempDir()
	out := filepath.Join(tmpD, "report.html")

	manifestPath := filepath.Join("..", "..", "testdata", "golden", "manifest.jsonl")
	hypPath := filepath.Join("..", "..", "testdata", "golden", "hyphotheses.jsonl")

	manifests, err := corpus.NewReader().ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Ошибка чтения манифеста: %v", err)
	}
	hypotheses, err := corpus.NewReader().ReadHypotheses(hypPath)
	if err != nil {
		t.Fatalf("Ошибка чтения гипотез: %v", err)
	}

	normalizer := normalize.NewNormalizer("ru-default")
	result := corpus.Evaluate(manifests, hypotheses, normalizer)

	builder := NewBuilder()
	reportData := builder.Build(result)

	tmpJson := filepath.Join(tmpD, "report.json")
	if err := Write(tmpJson, reportData); err != nil {
		t.Fatalf("Ошибка записи отчета: %v", err)
	}
	//t.Logf("jsonPath: %s", tmpJson)
	//t.Logf("htmlPath: %s", out)

	if err := WriteHTML(tmpJson, out); err != nil {
		t.Fatalf("Ошибка записи отчета: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("Ошибка чтения результата: %v", err)
	}
	goldenPath := filepath.Join("..", "..", "testdata", "control", "expected_report.html")

	if *update {
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("Ошибка обновления golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Ошибка чтения golden: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("golden HTML mismatch")
		t.Logf(" got len: %d, want len: %d", len(got), len(want))
	}

}

func TestGoldenCompare(t *testing.T) {

	baselinePath := filepath.Join("..", "..", "testdata", "golden", "baseline.json")
	currentPath := filepath.Join("..", "..", "testdata", "control", "expected_report.json")

	result, err, _ := Compare(baselinePath, currentPath, 0.02, 0.02)
	if err != nil {
		t.Fatalf("Ошибка сравнения: %v", err)
	}
	got, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		t.Fatalf("Ошибка маршаллинга: %v", err)
	}
	goldenPath := filepath.Join("..", "..", "testdata", "control", "expected_compare.json")

	if *update {
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("Ошибка обновления golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Ошибка чтения golden: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("golden compare mismatch")
		//t.Logf(" got len: %d, want len: %d", len(got), len(want))
	}

	//t.Logf("got: %s", string(got[:200]))
	//t.Logf("want: %s", string(want[:200]))

}
