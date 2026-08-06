package runner

import (
	"context"
	"time"
)

type Task struct {
	ID      string
	Audio   string
	Text    string
	Timeout time.Duration
}
type Result struct {
	ID string `json:"id"`
	Hypothesis string `json:"hypothesis"`
	Error string `json:"error,omitempty"`
	RecognitionTime time.Duration `json:"recognition-time-ms"`
	Status string `json:"status"`
}

type Runner interface {
	Run(ctx context.Context, task Task) Result
}