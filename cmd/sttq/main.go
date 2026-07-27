package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

func runImport() {
	var (
		archivePath = flag.String("archive", "test.tar", "Путь к архиву")
		quota       = flag.String("quota", "crowd=200,farfield=50", "Квоты")
		seed        = flag.String("seed", "intership-2026", "Сид")
		outPath     = flag.String("out", "corpus", "Вывод")
	)
	fmt.Print(os.Args[1:], "\n")
	flag.CommandLine.Parse(os.Args[2:])
	fmt.Printf("%v, %v, %v, %v", *archivePath, *quota, *seed, *outPath)

	StartImport := app.NewImportApp(*archivePath, *quota, *seed, *outPath)
	if err := StartImport.Run(); err != nil {
		log.Fatalf("Ошибка выполнения: %v", err)
	}
}
