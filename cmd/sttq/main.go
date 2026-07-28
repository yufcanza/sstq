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

	case "import-golos":
		runImport()
	}
}
func runEvaluate() {
	//flags := flag.NewFlagSet(command, flag.ExitOnError)
	var (
		manPath = flag.String("manifest", "corpus/manifest.jsonl", "Путь к файлу с эталонами")
		hypPath = flag.String("hypotheses", "run.jsonl", "Путь к файлу с гипотезами")
		norm    = flag.String("normalization", "ru-default", "Профиль нормализации")
		outPath = flag.String("out", "report.json", "Выход")
	)
	//fmt.Print(os.Args[1:], "\n")
	flag.CommandLine.Parse(os.Args[2:])
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
		archivePath = flag.String("archive", "test.tar", "Путь к архиву")
		//quota       = flag.String("quota", "crowd=200,farfield=50", "Квоты")
		seed    = flag.String("seed", "intership-2026", "Сид")
		outPath = flag.String("out", "corpus", "Вывод")
	)
	var quotas quotaSlice
	flags.Var(&quotas, "quota", "Квота")

	fmt.Print(os.Args[1:], "\n")
	flags.Parse(os.Args[2:])
	quotaMap := parseQuotas(quotas)
	fmt.Printf("%v, %v, %v, %v", *archivePath, quotaMap, *seed, *outPath)

	StartImport := app.NewImportApp(*archivePath, quotaMap, *seed, *outPath)
	if err := StartImport.Run(); err != nil {
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
