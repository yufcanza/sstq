package main

import (
	"flag"
	"log"
	"os"
	"sttq/internal/app"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	command := os.Args[1]
	if command == "evaluate" {
		runEvaluate()
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
	//flag.CommandLine.Parse(os.Args[2:])
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

	StartEval := app.NewApp(*manPath, *hypPath, *outPath, *norm)
	if err := StartEval.Run(); err != nil {
		log.Fatalf("Ошибка выполнения: %v", err)
	}

}
