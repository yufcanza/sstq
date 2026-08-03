package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sttq/internal/app"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {
	case "evaluate":
		runEvaluate()

	case "corpus":
		runCorpus()
	case "report":
		runReport()
	}

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
		seed    = flags.String("seed", "intership-2026", "Сид")
		outPath = flags.String("out", "corpus", "Вывод")
	)
	var quotas quotaSlice
	flags.Var(&quotas, "quota", "Квота")
	fmt.Print(os.Args[1:], "\n")
	flags.Parse(os.Args[3:])
	quotaMap := parseQuotas(quotas)
	fmt.Printf("%v, %v, %v, %v", *archivePath, quotaMap, *seed, *outPath)

	StartImport := app.NewImportApp(*archivePath, quotaMap, *seed, *outPath)
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

func runReport(){
	flags:=flag.NewFlagSet("report", flag.ExitOnError)
	var(
		inputPath = flags.String("input", "report.json", "Путь к json-отчету")
		format = flags.String("format", "html", "формат вывода: hmtl")
		outputPath = flags.String("out", "report.html", "Путь вывода отчета")
	)

	flags.Parse(os.Args[2:])
	StartReport := app.NewReportApp(*inputPath, *format, *outputPath)
	if err := StartReport.Run(); err != nil{
		log.Fatalf("Ошибка: %v", err)
	}
}
