package runner

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	//"strings"
	"time"
)

type WhisperConfig struct {
	BinaryPath string
	ModelPath  string
	Language   string
	Timeout    time.Duration
	Workers int
	Resume bool
	OutputPath string
}
type WhisperRunner struct{
	config WhisperConfig
}
func NewWhisperRunner (config WhisperConfig) *WhisperRunner{
	return &WhisperRunner{
		config: config,
	}
}

func (w *WhisperRunner) Run(ctx context.Context, task Task) Result{
	ctx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	start :=time.Now()

	args := []string{
		"-m", w.config.ModelPath,
		"-f", task.Audio,
		"-l", w.config.Language,
	}

	cmd := exec.CommandContext(ctx, w.config.BinaryPath, args...)

	strout, err := cmd.StdoutPipe()
	if err != nil{
		return Result{
			ID: task.ID,
			Error: fmt.Sprintf("Ошибка создания канала: %v", err),
			Hypothesis: "",
			Status: "error",
			RecognitionTime: time.Since(start),
		}
	}
	defer strout.Close()

	if err := cmd.Start(); err != nil{
		return Result{
			ID: task.ID,
			Error: fmt.Sprintf("Ошибка запуска: %v", err),
			Hypothesis: "",
			Status: "error",
			RecognitionTime: time.Since(start),
		}
	}

	scanner := bufio.NewScanner(strout)
	
	for scanner.Scan(){
	line:=scanner.Text()
	fmt.Printf("%s\n", line)
	}
	if err :=scanner.Err(); err!=nil{
		return Result{
			ID: task.ID,
			Error: fmt.Sprintf("Ошибка чтения вывода: %v", err),
			Hypothesis: "",
			Status: "error",
			RecognitionTime: time.Since(start),
		}
	}

	return Result{
		ID: task.ID,
		Hypothesis: "nil",
		Status: "success",
		RecognitionTime: time.Since(start),
	}
}