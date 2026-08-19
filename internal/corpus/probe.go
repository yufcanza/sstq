package corpus

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

type AudioInfo struct {
	DurationMS int64
	SampleRate int
	Channels   int
}

func Probe(filePath string) (*AudioInfo, error) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, fmt.Errorf("ffprobe не найден в PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams", filePath)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe timeout %v для %s", timeout, filePath)
		}
		return nil, fmt.Errorf("ffprobe: %w", err)
	}
	var result struct {
		Streams []struct {
			SampleRate string `json:"sample_rate"`
			Channels   int    `json:"channels"`
			Duration   string `json:"duration"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("парсинг ffprobe: %w", err)
	}

	if len(result.Streams) == 0 {
		return nil, fmt.Errorf("нет аудио-потоков")
	}
	info := &AudioInfo{}
	info.SampleRate, _ = strconv.Atoi(result.Streams[0].SampleRate)
	info.Channels = result.Streams[0].Channels
	duration, _ := strconv.ParseFloat(result.Streams[0].Duration, 64)
	info.DurationMS = int64(duration * 1000)

	return info, nil
}
