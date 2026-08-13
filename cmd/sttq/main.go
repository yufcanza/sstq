package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sttq/internal/app"
	"time"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {
	case "version":
		printVersion()
	case "evaluate":
		runEvaluate()
	case "corpus":
		runCorpus()
	case "report":
		runReport()
	case "compare":
		runCompare()
	case "audio":
		runAudioPrepare()
	case "run":
		runRun()
	}

}
func printVersion() {
	fmt.Printf("sstq version %s\n", version)
}
func runEvaluate() {
	flags := flag.NewFlagSet("evaluate", flag.ExitOnError)
	var (
		manPath = flags.String("manifest", "corpus/manifest.jsonl", "Путь к файлу с эталонами")
		hypPath = flags.String("hypotheses", "run.jsonl", "Путь к файлу с гипотезами")
		norm    = flags.String("normalization", "ru-default", "Профиль нормализации")
		outPath = flags.String("out", "report.json", "Выход")
	)
	//fmt.Print(os.Args[1:], "\n")
	flags.Parse(os.Args[2:])
	//fmt.Printf("%v, %v, %v, %v", *manPath, *hypPath, *norm, *outPath)
	if *manPath == "" {
		log.Fatal("Не указан файл эталонов")
	}
	if *hypPath == "" {
		log.Fatal("Не указан файл гипотез")
	}
	if *norm == "" {
		log.Printf("Не указан профиль нормализации")
	}
	if *outPath == "" {
		log.Printf("Не указан выходной файл")
	}

	StartEval := app.NewEvalApp(*manPath, *hypPath, *outPath, *norm)
	if err := StartEval.Run(); err != nil {
		log.Fatalf("Ошибка выполнения: %v", err)
	}

}
func runCorpus() {
	subcommand := os.Args[2]
	switch subcommand {
	case "import-golos":
		runImport()
	case "stats":
		runStats()
	case "validate":
		runValidate()

	}
}

type quotaSlice []string

func (q *quotaSlice) Set(value string) error {
	*q = append(*q, strings.TrimSpace(value))
	return nil
}
func (q *quotaSlice) String() string {
	return strings.Join(*q, ", ")
}

func runImport() {
	flags := flag.NewFlagSet("import-golos", flag.ExitOnError)
	var (
		archivePath = flags.String("archive", "test.tar", "Путь к архиву")
		//quota       = flag.String("quota", "crowd=200,farfield=50", "Квоты")
		seed           = flags.String("seed", "intership-2026", "Сид")
		limit          = flags.Int("limit", 250, "Максимальное количество записей")
		maxDurationStr = flags.String("max-duration", "30m", "Максимальная длительность одной записи")
		outPath        = flags.String("out", "corpus", "Вывод")
	)
	var quotas quotaSlice
	flags.Var(&quotas, "quota", "Квота")
	//fmt.Print(os.Args[1:], "\n")
	flags.Parse(os.Args[3:])
	quotaMap := parseQuotas(quotas)
	maxDuration, err := time.ParseDuration(*maxDurationStr)
	if err != nil {
		log.Fatalf("Некорректный --max-duration %q: %v", *maxDurationStr, err)
	}
	StartImport := app.NewImportApp(*archivePath, quotaMap, *seed, *outPath, *limit, maxDuration)
	if err := StartImport.Run(); err != nil {
		log.Fatalf("Ошибка выполнения: %v", err)
	}
}
func runStats() {
	flags := flag.NewFlagSet("stats", flag.ExitOnError)
	var manPath = flags.String("manifest", "corpus/manifest.jsonl", "Путь к эталонам")
	flags.Parse(os.Args[3:])
	//fmt.Printf("%v\n", *manPath)

	StartStats := app.NewStatsApp(*manPath)
	if err := StartStats.Run(); err != nil {
		log.Fatalf("Ошибка выполнения: %v", err)
	}
}
func runValidate() {
	flags := flag.NewFlagSet("validate", flag.ExitOnError)
	var manPath = flags.String("manifest", "corpus/manifest.jsonl", "Путь к эталонам")
	flags.Parse(os.Args[3:])
	//fmt.Printf("%v\n", *manPath)

	StartValidate := app.NewValidateApp(*manPath)
	if err := StartValidate.Run(); err != nil {
		log.Fatalf("Ошибка выполнения: %v", err)
	}
}
func parseQuotas(qs quotaSlice) map[string]int {
	result := make(map[string]int)

	if len(qs) == 0 {
		// Значения по умолчанию
		result["crowd"] = 200
		result["farfield"] = 50
		return result
	}

	for _, q := range qs {
		parts := strings.Split(q, "=")
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		result[key] = val
	}

	return result
}

func runReport() {
	flags := flag.NewFlagSet("report", flag.ExitOnError)
	var (
		inputPath  = flags.String("input", "report.json", "Путь к json-отчету")
		format     = flags.String("format", "html", "формат вывода: html")
		outputPath = flags.String("out", "report.html", "Путь вывода отчета")
	)

	flags.Parse(os.Args[2:])
	StartReport := app.NewReportApp(*inputPath, *format, *outputPath)
	if err := StartReport.Run(); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}

func runCompare() {
	flags := flag.NewFlagSet("compare", flag.ExitOnError)
	var (
		baseline = flags.String("baseline", "./reports/baseline.json", "Путь к baseline")
		current  = flags.String("current", "./reports/current.json", "Путь к current")
		maxWER   = flags.Float64("max-wer-delta", 0.02, "Максимальный порог WER")
		maxCER   = flags.Float64("max-cer-delta", 0.02, "Максимальный порог CER")
	)
	flags.Parse(os.Args[2:])
	StartCompare := app.NewCompareApp(*baseline, *current, *maxWER, *maxCER)
	exitCode := StartCompare.Run()
	os.Exit(exitCode)
}

func runAudioPrepare() {
	if len(os.Args) < 3 || os.Args[2] != "prepare" {
        log.Fatal("Неккоректное использование: sttq audio prepare")
    }
	flags := flag.NewFlagSet("audio-prepare", flag.ExitOnError)
	var (
		manifestPath = flags.String("manifest", "./corpus/manifest.jsonl", "путь к манифесту")
		profile      = flags.String("profile", "wav-16k", "профиль: wav-16-k или wav-8-k")
		workers      = flags.Int("workers", 4, "количество воркеров")
		timeoutStr   = flags.String("timeout", "30s", "Таймаут на запись (в секундах)")
		outDir       = flags.String("out", "./corpus", "выхоодная директория")
	)
	flags.Parse(os.Args[3:])
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		log.Fatalf("Некорректный --max-duration %q: %v", *timeoutStr, err)
	}
	StartPrepare := app.NewAudioPrepareApp(*manifestPath, *profile, *workers, timeout, *outDir)
	if err := StartPrepare.Run(); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}
func runRun() {
	if len(os.Args) < 3 || os.Args[2] != "whispercpp" {
        log.Fatal("Неккоректное использование: sttq run whispercpp")
    }
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		manifestPath = flags.String("manifest", "./corpus/manifest.jsonl", "путь к манифесту")
		binaryPath   = flags.String("binary", "bin/whisper-cli.exe", "путь к whisper-cli")
		modelPath    = flags.String("model", "./models/ggml-tiny.bin", "путь к модели")
		language     = flags.String("language", "ru", "язык")
		workers      = flags.Int("workers", 2, "количество воркеров")
		timeoutStr   = flags.String("timeout", "30s", "таймаут на одну запись")
		resume       = flags.Bool("resume", false, "возобновить прогон")
		outputPath   = flags.String("out", "./runs/whisper.jsonl", "путь для результатов")
	)
	flags.Parse(os.Args[3:])
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		log.Fatalf("Некорректный --max-duration %q: %v", *timeoutStr, err)
	}
	StartRun := app.NewRunApp(*manifestPath, *binaryPath, *modelPath, *language, *workers, timeout, *resume, *outputPath)
	if err := StartRun.Run(); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}
