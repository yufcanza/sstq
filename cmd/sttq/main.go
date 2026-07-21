package main

import (
	"flag"
	"log"
)

func main() {
	var (
		manPath = flag.String("manifest", "", "Путь к файлу с эталонами")
		hypPath = flag.String("hypotheses", "", "Путь к файлу с гипотезами")
		norm    = flag.String("normalization", "", "Профиль нормализации")
		outPath = flag.String("out", "", "Выход")
	)
	flag.Parse()

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

}
