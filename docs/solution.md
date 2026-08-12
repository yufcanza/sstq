# Архитектура STTQ

#### Пакеты и их применение
**cmd/sttq** - Точка входа, которая парсит команды и флаги, и передает управление в app
**internal/app** - Содержит структуры запуска для каждой команды
* EvalApp - оценка качества
* ImportApp - импорт корпуса
* RunApp - запуск распознавания
* ReportApp - конвертация отчетов
* CompareApp - сравнение прогонов
* ValidateApp - валидация
* StatsApp - статистика
Каждый app получает конфигурацию запуска, создает нужные компоненты и запускает процесс.
**internal/corpus** - Работа с корпусом данных. Содержит:
* Record - структуру записи в манифесте
* Manifest - структура манифеста
* Reader/Writer - чтение, запись JSONL
* ImportGolos() - испорт корпуса из Golos
* Validation() - проверка манифеста
* Stats() - сбор статистики
* Probe() - получение метаданных для проверки манифеста
**internal/audio** - Работа с аудиофайлами. Содержит:
Prepare() - конвертация через FFmpeg
**internal/normalize** - Нормализация текста. Содержит:
* Normalizer - применяет преобразования
* Transform - интерфейс для отдельных преобразований
* Профили: strict и ru-default
**internal/metrics** - Метрики оценки. Содержит:
* Levenshtein() - алгоритм Левенштейна
* CalculateWER() - рассчет Word Error Rate
* CalculateCER() - рассчет Character Error Rate
* CreateAlignment() - выравнивание
**internal/runner** - Запуск распознавания. Содержит:
* WhisperRunne - адаптер для whisper.cpp
* FakeRunner - заглушка для тестов, повторяющая WhisperRunner не запуская whisper.cpp
* Pool - создает пул воркеров для параллельной обработки
* Resume - поддержка возобновления прерванных прогонов
**internal/report** - Генерация отчетов. Содержит
* Write() - JSON-отчет
* WriteHTML() -  HTML-отчет
* Compare  - сравнение с baseline
#### Потоки данных
###### Импорт корпуса
test.tar -> ExtractTar() -> во временную папку
        -> ReadManifest() -> FolosRecord
        -> parseManifest() -> ProcessedRecord
        -> SelectRecord() -> сформированная выборка, corpus/selection.json
        -> копирование аудио -> corpus/audio/<id>.wav
        ->SHA -> WriteManifest() -> corpus/manifest.jsonl, corpus/import-summary.json

###### Подготовка аудио
manifest.jsonl -> ReadRecords() -> Record
              -> для каждой записи:
                   FFmpeg -i input.wav -ac 1 -ar 16000 output.wav.tmp
                   rename -> output.wav
              -> статистика (ok/skipped/error)

###### Распознавание
manifest.jsonl -> ReadRecords() -> Record
              -> whisper-cli -m model.bin -f audio.wav -l ru
              -> парсинг stdout -> Hypothesis
              -> WriteJSONL() -> runs/whisper.jsonl

###### Оценка качества
manifest.jsonl ->ReadManifest() -> эталоны
runs/whisper.jsonl -> ReadHypotheses() -> гипотезы
              -> для каждой записи:
                   Normalize() -> нормализованные тексты
                   CalculateWER() / CalculateCER() -> Levenshtein() -> операции
              -> WriteReport() -> report.json

# Разбор Golos
#### Процесс импорта
1. Домен определяется по пути к манифесту:
сrowd/manifest.jsonl    -> domain = "crowd"
farfield/manifest.jsonl -> domain = "farfield"
2. Чтение манифеста происходит в структуру GolosRecord:
```
type GolosRecord struct {
    AudioFilepath string  `json:"audio_filepath"`
    Text          string  `json:"text"`
    Duration      float64 `json:"duration"`
}
```
3. Генерация стабильных ID. ID не зависит от сида, только от домена и пути.
```
hash := SHA256(domain + ":" + audio_filepath)
id := domain + "-" + hex(hash[:6])
```
4. Выборка с квотами уже формируется с использованием сида, что позволяет создавать разные выборки с разными сидами
```
sortHash := SHA256(seed + ":" + domain + ":" + audio_filepath)
```
После sortHash сортируется, и первые N записей, определенных квотой, выбираются
5. Копирование аудио. 
6. Проверка целостности

# Формирование стабильных id
По заданию необходимо обеспечить стабильность ID.
Идентификатор должен:
* Не зависеть от абсолютного пути на компьютере пользователя
* Не зависеть от порядка файлов
* Не зависеть от сида
* Быть одинаковом при повторном импорте
* Быть уникальным в пределах корпуса.

```
hash = SHA256(domain + ":" + audio_filepath)
id = domain + "-" + hex(hash[:6])
```
Почему я сделала так:
domain, audio_filepath - это категория записи и ее относительный путь внутри test.tar. Так я не обращаюсь к абсолютному пути на компьютере и завишу только от домена и относительного пути
После этого имя аудиозаписи составляется из ее домена (для удобства дальнейшей обработки) и первых 6 байтов хеша прошлой сформированной строки. 6 байтов достаточно в претелах корпуса. 

# Алгоритм выборки

1. Группировка по категориям - Все записи разделяются на 2 группы - crowd и farfield.
2. Для каждой записи вычисляется SorhHash, на который влияет seed
3. Сортировка внутри домена - по SorhHash обе категории сортируются.
```
sort.Slice(recs, func(i, j int) bool {
    return recs[i].SortHash < recs[j].SortHash
})
```
4. Для каждого домена в пределах квоты выбираются первые N записей
```
take := quota
if len(recs) < take {
    take = len(recs)
}
selected = recs[:take]
```
5. Если указан --max-duration, записи длинее порога при этом пропускаются.
```
if config.MaxDuration > 0 && rec.Duration > config.MaxDuration {
    continue
}
```
6. Если происходит превышение порога  limit, обрезаем
```
if limit > 0 && len(selected) > limit {
    selected = selected[:limit]
}
```
7. Сортировка результата - Для стабильности итоговый список сортируется по ID
```
sort.Slice(selected, func(i, j int) bool {
    return selected[i].ID < selected[j].ID
})
```

# Модель данных
#### Исходный манифест
**Формат: JSONL**
```
{"audio_filepath":"files/00b1dd4be8a32fc7e4ab7aaffb9666e0.wav","text":"афина воспроизведи музыку вперемешку","duration":4.9}
```
**Структура**
```
type GolosRecord struct {
    AudioFilepath string  `json:"audio_filepath"`
    Text          string  `json:"text"`
    Duration      float64 `json:"duration"`
}
```
#### Внутренние структуры
**ProcessedRecord** - используется на этапе импорта для обработки записей. 
```
type ProcessedRecord struct {
    Domain        string  
    AudioFilepath string
    Text          string
    Duration      float64
    ID            string
    SortHash      [32]byte
}
```
**SelectionInfo** - параметры выборки
```
type SelectionInfo struct {
    Source         string         `json:"source"`
    Seed           string         `json:"seed"`
    RequestRecords int            `json:"request_records"`
    MaxDuration    int64          `json:"max_duration_ms"`
    Quotas         map[string]int `json:"quotas"`
    SelectedIDs    []string       `json:"selected_ids"`
}
```
#### Итоговый манифест
**Формат: JSONL**
```
{
  "id": "crowd-0d4a81f2417b",
  "audio": "audio/crowd-0d4a81f2417b.wav",
  "text": "афина воспроизведи музыку вперемешку",
  "language": "ru",
  "duration_ms": 4900,
  "sample_rate": 16000,
  "channels": 1,
  "tags": ["crowd"],
  "sha256": "d09f..."
}
```
**Структура**
```
type Record struct {
    ID         string   `json:"id"`
    Audio      string   `json:"audio"`
    Text       string   `json:"text"`
    Language   string   `json:"language"`
    Duration   int      `json:"duration_ms"`
    SampleRate int      `json:"sample_rate"`
    Channels   int      `json:"channels"`
    Tags       []string `json:"tags"`
    SHA256     string   `json:"sha256"`
}
```
#### Гипотеза
**Формат: JSON**
```
{"id":"crowd-0d4a81f2417b","text":"афина воспроизведи музыку","recognition-time-ms":1234,"status":"success"}
```
**Структура**
```
type Result struct {
	ID              string        `json:"id"`
	Hypothesis      string        `json:"text"`
	Error           string        `json:"error,omitempty"`
	RecognitionTime time.Duration `json:"recognition-time-ms"`
	Status          string        `json:"status"`
}
```
#### Результат оценки
**Формат: JSON**
```
{
  "id": "crowd-0d4a81f2417b",
  "reference": "афина воспроизведи музыку вперемешку",
  "hypothesis": "афина воспроизведи музыку",
  "normalized_reference": "афина воспроизведи музыку вперемешку",
  "normalized_hypothesis": "афина воспроизведи музыку",
  "reference_words": 5,
  "hits": 4,
  "substitutions": 0,
  "deletions": 1,
  "insertions": 0,
  "wer": 0.2,
  "cer": 0.05,
  "exact_match": false,
  "tags": ["crowd"],
  "alignment": [
    {"type": "equal", "reference": "афина", "hypothesis": "афина"},
    {"type": "equal", "reference": "воспроизведи", "hypothesis": "воспроизведи"},
    {"type": "equal", "reference": "музыку", "hypothesis": "музыку"},
    {"type": "delete", "reference": "вперемешку", "hypothesis": ""}
  ]
}
```
**Структура**
```
type Result struct {
	ID                  string   `json:"id"`
	Reference           string   `json:"reference"`
	Hypothesis          string   `json:"hypothesis"`
	NormalizedReference string   `json:"normalized_reference"`
	NormalizeHypothesis string   `json:"normalized_hypothesis"`
	ReferenceWords      int      `json:"reference_words"`
	Hits                int      `json:"hits"`
	Substitutions       int      `json:"substitutions"`
	Deletions           int      `json:"deletions"`
	Insertions          int      `json:"insertions"`
	WER                 float64  `json:"wer"`
	SubstitutionsCER    int      `json:"substitutions_cer"`
	DeletionsCER        int      `json:"deletions_cer"`
	InsertionsCER       int      `json:"insertions_cer"`
	CER                 float64  `json:"cer"`
	ExactMatch          bool     `json:"exact_match"`
	Tags                []string `json:"tags,omitempty"`
	DurationMS          int64    `json:"duration_ms,omitempty"`
	RecognitionTimeMS   int64    `json:"recognition_time_ms,omitempty"`

	Alignment []metrics.AlignmentItem `json:"alignment"`
	Error     string                  `json:"error,omitempty"`
    
}
type AlignmentItem struct {
    Type      string `json:"type"`
    Reference string `json:"reference"`
    Hypothesis string `json:"hypothesis"`
}
```
# Левенштейн и alignment

Алгорим Левенштейна - способ измерить расстояние между двумя строками, по другому сколько минимальных операций нужно сделать, чтобы превратить одну строку в другую.
1. Создается матрица, размером (N+1)х(M+1), где N, M - количество слов (для WER) или символов (для CER)в исходных предложениях(словах)
2. Первые строка и стоблец заполняются от 0 до N/M
3. Каждая клетка вычисляется по формуле:
```
cost = 0 если a[i] == b[j], иначе 1
matrix[i][j] = min(
    matrix[i-1][j] + 1,      // удаление
    matrix[i][j-1] + 1,      // вставка
    matrix[i-1][j-1] + cost, // замена или совпадение
)
```
4. Обратный проход от правого нижнего угла к левому верхнему с приоритетом операций: совпадение, замена, удаление, вставка (от высшего приоритета к наименьшему)

Alignment - выравнивание - показывает, какие операции были выполнены при сравнении эталона и гипотезы.
**Типы операций**
|Операция|Тип|Описание|
|-|-|-|
|Match|equal|Символы/слова совпали|
|Substitute|substitute|Замена|
|Delete|delete|Удаление|
|Insert|insert|Вставка|

Мы получаем из алгоритма Левенштейна порядок операций, и расшифровываем и раскрываем его в выравнивании.
```
func CreateAlignment(manifest, hypothesis []string) Alignment {
	result := Levenshtein(manifest, hypothesis)

	items := make([]AlignmentItem, 0, len(result.Ops))
	i, j := 0, 0
	for _, op := range result.Ops {
		switch op {
		case "M":
			items = append(items, AlignmentItem{
				Type:       "equal",
				Manifest:   manifest[i],
				Hypothesis: hypothesis[j],
			})
			i++
			j++
        }
    }
}
```
В примере кода видно, что мы получили результат из Левенштейна, и считываем его, преобразовывая в AlignmetItem. У нас буква M (Match) - что означает совпадение. Мы создаем AlignmentItem с типом "equal" и записываем манифест и гипотезу, в которой мы получили M

# Нормализация
Каждое преобразование реализует интерфейс:
```
type Transform interface {
    Apply(text string) string
}
```
Существующие преобразования:
NFCTransform
LowerCaseTransform
ReplaceTransform
PunctuationToSpacesTransform
CollapseSpacesTransform
TrimSpaceTransform

В системе два профиля: strict и ru-default
```
func NewStrictProfile() *Profile {
    return &Profile{
        Name: "strict",
        Transforms: []Transform{
            NFCTransform{},
            CollapseSpacesTransform{},
            TrimSpaceTransform{},
        },
    }
}
```
```
func NewRuDefaultProfile() *Profile {
    return &Profile{
        Name: "ru-default",
        Transforms: []Transform{
            NFCTransform{},
            LowerCaseTransform{},
            ReplaceYoTransform{},
            PunctuationToSpacesTransform{},
            CollapseSpacesTransform{},
            TrimSpaceTransform{},
        },
    }
}
```
Метод Normalize применяет все преобразования профиля.

# Внешние процессы

Испольуется 3 внешних процесса
**FFmreg** - необходим для конвертации аудио 
**ffprobe** - необходим для получения метаданных аудио
**whisper-cli** - распознавание речи

Все эти процессы запускаются через os.exec

##### FFmpeg 
Команда:
```
ffmpeg -i input.wav -ac 1 -ar 16000 -c:a pcm_s16le output.wav
```
###### Флаги
|Флаг|Описание|
|-|-|
|-i| Входной файл|
|-ac|Количество каналов|
|-ar|Частота дискретизации|
|-c:a|Аудиокодек|

###### Использование в коде

```
func processRecord(ctx context.Context, rec Record, config PrepareConfig) Result {
    // Аргументы без shell-интерпретации
    args := []string{
        "-i", rec.Audio,
        "-ac", "1",
        "-ar", "16000",
        "-c:a", "pcm_s16le",
        tmpPath,
    }
    
    cmd := exec.CommandContext(ctx, "ffmpeg", args...)
    ...
}
```
##### ffbrobe 
Команда:
```
ffprobe -v quiet -print_format json -show_streams audio.wav
```
###### Флаги
|Флаг|Описание|
|-|-|
|-v quiet| Минимум вывода|
|-print_format json|Вывод в JSON|
|-show_streams|Информация о потоках|

###### Использование в коде

```
func Probe(filePath string) (*AudioInfo, error) {
    cmd := exec.Command("ffprobe",
        "-v", "quiet",
        "-print_format", "json",
        "-show_streams",
        filePath,
    )
    
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("ffprobe: %w", err)
    }
    
    var result struct {
        Streams []struct {
            SampleRate string `json:"sample_rate"`
            Channels   int    `json:"channels"`
            Duration   string `json:"duration"`
        } `json:"streams"`
    }
    ...
}
```
##### whisper-cli 
Команда:
```
whisper-cli -m model.bin -f audio.wav -l ru --no-timestamps
```
###### Флаги
|Флаг|Описание|
|-|-|
|-m| Путь к модели|
|-f|Путь к аудиофайлу|
|-l|Язык|
|--no-timestamp|Не выводить таймстемпы|

###### Использование в коде

```
func (w *WhisperRunner) Run(ctx context.Context, task Task) Result {
    start := time.Now()
    
    args := []string{
        "-m", w.config.ModelPath,
        "-f", task.Audio,
        "-l", w.config.Language,
        "--no-timestamps",
    }
    
    cmd := exec.CommandContext(ctx, w.config.BinaryPath, args...)
    
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return Result{Status: "error", Error: err.Error()}
    }
    defer stdout.Close()
    ...
}
```
# Детерминированность

В системе детерминированность достигнула на всех этапах.

* Генерация ID - только по входным данным
* Выборка с квотами - при одинаковом сиде одинаковая выборка 
* Сортировка манифеста - всегда сортируется по ID
* Отчет имеет детерминированную структуру - сортировка всех полей: и записей, и ошибок
* Демо прогон без внешних зависимостей

# Benchmark

Бенчмарки направлены на проверку производительности. Цель - 10 000 пар текстов должны обрабатываться <10 секунд с потреблением памяти не более 256 МБ.
1. WER
Результат бенчмарка:
```
=== RUN   BenchmarkWER_10k
BenchmarkWER_10k
BenchmarkWER_10k-8
      51          23022269 ns/op        12320046 B/op     200000 allocs/op
PASS
ok      sttq/internal/metrics   2.304s
```
2. CER
```
=== RUN   BenchmarkCER_10k
BenchmarkCER_10k
BenchmarkCER_10k-8
       3         367560700 ns/op        199281290 B/op   1590006 allocs/op
PASS
ok      sttq/internal/metrics   2.410s
```
3. Нормализация
```
=== RUN   BenchmarkNormalize_RuDefault_10k
BenchmarkNormalize_RuDefault_10k
BenchmarkNormalize_RuDefault_10k-8
      14          72137079 ns/op        11326536 B/op     180025 allocs/op
PASS
ok      sttq/internal/normalize 1.317s
```
4. Чтение JSON
```
=== RUN   BenchmarkParseManifest
BenchmarkParseManifest
BenchmarkParseManifest-8
    1928            707883 ns/op          885669 B/op         13 allocs/op
PASS
ok      sttq/internal/corpus    11.013s
```
5. Формирование отчета
```
=== RUN   BenchmarkReport
BenchmarkReport
BenchmarkReport-8
      73          17197207 ns/op        14075987 B/op         28 allocs/op
PASS
ok      sttq/internal/report    1.591s
```
```
=== RUN   TestReportMemory
    d:/searchinform/sttq/internal/report/benchmark_test.go:62: Размер отчета: 2.38 MB
--- PASS: TestReportMemory (0.05s)
PASS
ok      sttq/internal/report    0.350s
```

# Принятые технические решения

Все технические решения были приняты в соответствии с заданием и представлены в README.md


