package runner

import (
	"context"
	"sync"
)

type Pool struct {
	workers int
	runner  Runner
	tasks   chan Task
	results chan Result
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPool(workers int, runner Runner) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workers: workers,
		runner:  runner,
		tasks:   make(chan Task, workers*30),
		results: make(chan Result, workers*30),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			result := p.runner.Run(p.ctx, task)
			p.results <- result
		}
	}
}

func (p *Pool) Submit(task Task) {
	p.tasks <- task
}
func (p *Pool) Results() <-chan Result {
	return p.results
}

func (p *Pool) CloseTasks() {
	close(p.tasks)
	p.wg.Wait()
	close(p.results)
}

func (p *Pool) Stop() {
	p.cancel()
}
func (p *Pool) Wait() {
	p.wg.Wait()
}
