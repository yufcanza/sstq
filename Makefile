.PHONY: build test bench golden integration demo

BINARY_NAME=sttq.exe
DEMO_DIR=demo
DEMO_MANIFEST=$(DEMO_DIR)/manifest.jsonl
DEMO_HYPOTHESES=$(DEMO_DIR)/hypotheses.jsonl
DEMO_BASELINE=$(DEMO_DIR)/baseline.json
DEMO_REPORT=$(DEMO_DIR)/report.json
DEMO_HTML=$(DEMO_DIR)/report.html

GOLOS_ARCHIVE=test.tar
WHISPER_BINARY=./bin/whisper-cli.exe
WHISPER_MODEL=./models/ggml-small.bin
CORPUS_DIR=./corpus
RUNS_DIR=./runs
INTEGRATION_REPORT=$(RUNS_DIR)/report.json

build:
	go build -o $(BINARY_NAME) ./cmd/sttq

test: 
	go test -v ./...

golden:
	go test ./internal/report/ -update

bench:
	go test ./... -bench=. -benchmem -benchtime=1x

demo: build
	@RECORDS=$$(grep -c "^{" $(DEMO_MANIFEST) 2>/dev/null || echo "0"); \
	echo ""; \
	echo -n "1. Validate demo corpus.............."; \
	if ./$(BINARY_NAME) corpus validate --manifest $(DEMO_MANIFEST) >/dev/null 2>&1; then \
		echo "OK ($$RECORDS records)"; \
	else \
		echo "FAIL"; \
		./$(BINARY_NAME) corpus validate --manifest $(DEMO_MANIFEST); \
		exit 1; \
	fi; \
	\
	echo -n "2. Evaluate hypotheses ..............."; \
	if ./$(BINARY_NAME) evaluate --manifest $(DEMO_MANIFEST) --hypotheses $(DEMO_HYPOTHESES) --normalization ru-default --out $(DEMO_REPORT) >/dev/null 2>&1 && [ -f "$(DEMO_REPORT)" ]; then \
		echo "OK"; \
	else \
		echo "FAIL"; \
		./$(BINARY_NAME) evaluate --manifest $(DEMO_MANIFEST) --hypotheses $(DEMO_HYPOTHESES) --normalization ru-default --out $(DEMO_REPORT); \
		exit 1; \
	fi; \
	echo "3. Write JSON report ............... $(DEMO_REPORT)"; \
	echo -n "4. Write HTML report ............... "; \
	if ./$(BINARY_NAME) report --input $(DEMO_REPORT) --out $(DEMO_HTML) >/dev/null 2>&1 && [ -f "$(DEMO_HTML)" ]; then \
		echo "$(DEMO_HTML)"; \
	else \
		echo "FAIL"; \
		./$(BINARY_NAME) report --input $(DEMO_REPORT) --out $(DEMO_HTML); \
		exit 1; \
	fi; \
	echo -n "5. Compare with baseline ........... "; \
	if ./$(BINARY_NAME) compare \
		--baseline $(DEMO_BASELINE) \
		--current $(DEMO_REPORT) \
		--max-wer-delta 0.02 \
		--max-cer-delta 0.02 2>&1 | grep -q "Status: PASS"; then \
		echo "PASS"; \
	else \
		echo "FAIL"; \
		./$(BINARY_NAME) compare \
			--baseline $(DEMO_BASELINE) \
			--current $(DEMO_REPORT) \
			--max-wer-delta 0.02 \
			--max-cer-delta 0.02; \
		exit 1; \
	fi; \
	echo ""; \
	echo "Overall:"; \
	echo "Records: $$RECORDS"; \
	SUCCESSFUL=$$(awk -F: '/"successful_result"/ {gsub(/[^0-9]/,"",$$2); print $$2; exit}' $(DEMO_REPORT)); \
	echo "Successful: $${SUCCESSFUL:-0}"; \
	ERRORS=$$(awk -F: '/"engine_errors"/ {gsub(/[^0-9]/,"",$$2); print $$2; exit}' $(DEMO_REPORT)); \
	echo "Engine errors: $${ERRORS:-0}"; \
	COVERAGE=$$(awk -F: '/"coverage"/ {gsub(/[^0-9.]/,"",$$2); print $$2; exit}' $(DEMO_REPORT)); \
	COVERAGE_PCT=$$(awk -v v="$$COVERAGE" 'BEGIN {printf "%.2f", (v+0)*100}'); \
	echo "Coverage: $$COVERAGE_PCT%"; \
	WER=$$(awk -F: '/"wer"/ {gsub(/[^0-9.]/,"",$$2); print $$2; exit}' $(DEMO_REPORT)); \
	WER_PCT=$$(awk -v v="$$WER" 'BEGIN {printf "%.2f", (v+0)*100}'); \
	echo "WER: $$WER_PCT%"; \
	CER=$$(awk -F: '/"cer"/ {gsub(/[^0-9.]/,"",$$2); print $$2; exit}' $(DEMO_REPORT)); \
	CER_PCT=$$(awk -v v="$$CER" 'BEGIN {printf "%.2f", (v+0)*100}'); \
	echo "CER: $$CER_PCT%"; \
	EXACT=$$(awk -F: '/"exact_matches"/ {gsub(/[^0-9]/,"",$$2); print $$2; exit}' $(DEMO_REPORT)); \
	echo "Exact matches: $${EXACT:-0}"

integration: build
	@echo "1. Import Golos corpus..."
	./$(BINARY_NAME) corpus import-golos --archive $(GOLOS_ARCHIVE) --limit 50 --max-duration 30m --seed integration-2026 --quota crowd=40 --quota farfield=10 --out $(CORPUS_DIR)
	@echo ""
	@echo "2. Prepare audio (wav-16k)..."
	./$(BINARY_NAME) audio prepare --manifest $(CORPUS_DIR)/manifest.jsonl --profile wav-16k --workers 4 --timeout 30s --out $(CORPUS_DIR)
	@echo ""
	@echo "3. Run whisper.cpp..."
	./$(BINARY_NAME) run whispercpp --manifest $(CORPUS_DIR)/manifest.jsonl --binary $(WHISPER_BINARY) --model $(WHISPER_MODEL) --language ru --workers 2 --timeout 2m --out $(RUNS_DIR)/whisper.jsonl
	@echo ""
	@echo "4. Evaluate..."
	./$(BINARY_NAME) evaluate --manifest $(CORPUS_DIR)/manifest.jsonl --hypotheses $(RUNS_DIR)/whisper.jsonl --normalization ru-default --out $(INTEGRATION_REPORT)
	@echo ""
	@echo "5.  HTML report..."
	./$(BINARY_NAME) report --input $(INTEGRATION_REPORT) --out $(RUNS_DIR)/report.html
	@echo "Results:"
	@echo "  Manifest:  $(CORPUS_DIR)/manifest.jsonl"
	@echo "  Audio:     $(CORPUS_DIR)/audio/"
	@echo "  Hypotheses:$(RUNS_DIR)/whisper.jsonl"
	@echo "  Report:    $(INTEGRATION_REPORT)"
	@echo "  HTML:      $(RUNS_DIR)/report.html"