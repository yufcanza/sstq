package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sttq/internal/app"
	"time"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
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
	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(2)
	}

}
func printVersion() {
	fmt.Printf("sstq version %s\n", version)
	os.Exit(0)
}
func runEvaluate() {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var (
		manPath = flags.String("manifest", "corpus/manifest.jsonl", "Путь к файлу с эталонами")
		hypPath = flags.String("hypotheses", "run.jsonl", "Путь к файлу с гипотезами")
		norm    = flags.String("normalization", "ru-default", "Профиль нормализации")
		outPath = flags.String("out", "report.json", "Выход")
	)
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	if *manPath == "" || *hypPath == "" {
		fmt.Fprintf(os.Stderr, "Не указан файл эталона или гипотез\n")
		os.Exit(2)
	}

	StartEval := app.NewEvalApp(*manPath, *hypPath, *outPath, *norm)
	if err := StartEval.Run(); err != nil {
		exitErrors(err)
	}

}
func runCorpus() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Подкоманда corpus не задана\n")
		printUsage()
		os.Exit(2)
	}
	subcommand := os.Args[2]
	switch subcommand {
	case "import-golos":
		runImport()
	case "stats":
		runStats()
	case "validate":
		runValidate()
	default:
		fmt.Fprintf(os.Stderr, "Ошибка входных данных: неизвестная подкоманда corpus: %s\n", os.Args[2])
		printUsage()
		os.Exit(2)
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
	flags := flag.NewFlagSet("import-golos", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var (
		archivePath    = flags.String("archive", "test.tar", "Путь к архиву")
		seed           = flags.String("seed", "internship-2026", "Сид")
		limit          = flags.Int("limit", 250, "Максимальное количество записей")
		maxDurationStr = flags.String("max-duration", "30m", "Максимальная длительность одной записи")
		outPath        = flags.String("out", "corpus", "Вывод")
	)
	var quotas quotaSlice
	flags.Var(&quotas, "quota", "Квота")
	if err := flags.Parse(os.Args[3:]); err != nil {
		os.Exit(2)
	}
	quotaMap := parseQuotas(quotas)
	maxDuration, err := time.ParseDuration(*maxDurationStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Некорректный --max-duration %q: %v\n", *maxDurationStr, err)
		os.Exit(2)
	}
	StartImport := app.NewImportApp(*archivePath, quotaMap, *seed, *outPath, *limit, maxDuration)
	if err := StartImport.Run(); err != nil {
		exitErrors(err)
	}
}
func runStats() {
	flags := flag.NewFlagSet("stats", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var manPath = flags.String("manifest", "corpus/manifest.jsonl", "Путь к эталонам")
	if err := flags.Parse(os.Args[3:]); err != nil {
		os.Exit(2)
	}

	StartStats := app.NewStatsApp(*manPath)
	if err := StartStats.Run(); err != nil {
		exitErrors(err)
	}
}
func runValidate() {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var manPath = flags.String("manifest", "corpus/manifest.jsonl", "Путь к эталонам")
	if err := flags.Parse(os.Args[3:]); err != nil {
		os.Exit(2)
	}

	StartValidate := app.NewValidateApp(*manPath)
	if err := StartValidate.Run(); err != nil {
		exitErrors(err)
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
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var (
		inputPath  = flags.String("input", "report.json", "Путь к json-отчету")
		format     = flags.String("format", "html", "формат вывода: html")
		outputPath = flags.String("out", "report.html", "Путь вывода отчета")
	)

	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	StartReport := app.NewReportApp(*inputPath, *format, *outputPath)
	if err := StartReport.Run(); err != nil {
		exitErrors(err)
	}
}

func runCompare() {
	flags := flag.NewFlagSet("compare", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var (
		baseline = flags.String("baseline", "./reports/baseline.json", "Путь к baseline")
		current  = flags.String("current", "./reports/current.json", "Путь к current")
		maxWER   = flags.Float64("max-wer-delta", 0.02, "Максимальный порог WER")
		maxCER   = flags.Float64("max-cer-delta", 0.02, "Максимальный порог CER")
	)
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	StartCompare := app.NewCompareApp(*baseline, *current, *maxWER, *maxCER)
	exitCode := StartCompare.Run()
	os.Exit(exitCode)
}

func runAudioPrepare() {
	if len(os.Args) < 3 || os.Args[2] != "prepare" {
		fmt.Fprintf(os.Stderr, "Ошибка подкоманды. Использование: sttq audio prepare\n")
		printUsage()
		os.Exit(2)
	}
	flags := flag.NewFlagSet("audio-prepare", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var (
		manifestPath = flags.String("manifest", "./corpus/manifest.jsonl", "путь к манифесту")
		profile      = flags.String("profile", "wav-16k", "профиль: wav-16-k или wav-8-k")
		workers      = flags.Int("workers", 4, "количество воркеров")
		timeoutStr   = flags.String("timeout", "30s", "Таймаут на запись (в секундах)")
		outDir       = flags.String("out", "./corpus", "выхоодная директория")
	)
	if err := flags.Parse(os.Args[3:]); err != nil {
		os.Exit(2)
	}
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Некорректный --max-duration %q: %v\n", *timeoutStr, err)
		os.Exit(2)
	}
	StartPrepare := app.NewAudioPrepareApp(*manifestPath, *profile, *workers, timeout, *outDir)
	if err := StartPrepare.Run(); err != nil {
		exitErrors(err)
	}
}
func runRun() {
	if len(os.Args) < 3 || os.Args[2] != "whispercpp" {
		fmt.Fprintf(os.Stderr, "Ошибка подкоманды. Использование: sttq run whispercpp\n")
		printUsage()
		os.Exit(2)
	}

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
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
	if err := flags.Parse(os.Args[3:]); err != nil {
		os.Exit(2)
	}
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Некорректный --timeout %q: %v\n", *timeoutStr, err)
		os.Exit(2)
	}
	StartRun := app.NewRunApp(*manifestPath, *binaryPath, *modelPath, *language, *workers, timeout, *resume, *outputPath)
	if err := StartRun.Run(); err != nil {
		exitErrors(err)
	}
}

func printUsage() {
	fmt.Println("Использование: sttq <команда>")
	fmt.Println()
	fmt.Println("Доступные команды: ")
	fmt.Println("version				Версия программы")
	fmt.Println("evaluate			Оценка качества")
	fmt.Println("corpus import-golos		Импорт корпуса")
	fmt.Println("corpus validate			Валидация корпуса")
	fmt.Println("corpus stats			Статистика корпуса")
	fmt.Println("report				Отчет")
	fmt.Println("compare				Сравнение в baseline")
	fmt.Println("audio prepare			Подготовка аудио")
	fmt.Println("run whispercpp			Запуск whisper.cpp")
}

func exitErrors(err error) {
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	isInput := strings.Contains(msg, "не найден") ||
		strings.Contains(msg, "не существует") ||
		strings.Contains(msg, "неизвестный") ||
		strings.Contains(msg, "некорректный") ||
		strings.Contains(msg, "ffprobe") ||
		strings.Contains(msg, "ffmpeg")
	if isInput {
		fmt.Fprintf(os.Stderr, "Ошибка входных данных: %v\n", err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "Ошибка выполнения: %v\n", err)
	os.Exit(1)
}
