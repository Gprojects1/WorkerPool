package workerpool

import (
	"context"
	"errors"
	"log"
	"sync"
)

type Job string

type Result struct {
	WorkerID int
	Job      Job
	Err      error
}

type Worker struct {
	ID      int
	jobChan <-chan Job
	result  chan<- Result
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

func NewWorker(id int, jobChan <-chan Job, result chan<- Result, parentCtx context.Context, wg *sync.WaitGroup) *Worker {
	ctx, cancel := context.WithCancel(parentCtx)
	return &Worker{
		ID:      id,
		jobChan: jobChan,
		result:  result,
		ctx:     ctx,
		cancel:  cancel,
		wg:      wg,
	}
}

func (w *Worker) Start() {
	defer w.wg.Done()
	defer log.Printf("Worker %d stopped", w.ID)

	for {
		select {
		case job, ok := <-w.jobChan:
			if !ok {
				return
			}
			w.result <- Result{WorkerID: w.ID, Job: job}
		case <-w.ctx.Done():
			w.result <- Result{WorkerID: w.ID, Err: w.ctx.Err()}
			return
		}
	}
}

type WorkerPool struct {
	workers    []*Worker
	jobChan    chan Job
	resultChan chan Result
	wg         sync.WaitGroup
	mu         sync.Mutex
}

func New(workerCount int) (*WorkerPool, error) {
	if workerCount <= 0 {
		return nil, errors.New("worker count must be positive")
	}

	pool := &WorkerPool{
		jobChan:    make(chan Job, workerCount*2),
		resultChan: make(chan Result, workerCount*2),
	}

	pool.wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		worker := NewWorker(
			i+1,
			pool.jobChan,
			pool.resultChan,
			context.Background(),
			&pool.wg,
		)
		pool.workers = append(pool.workers, worker)
		go worker.Start()
	}

	return pool, nil
}

func (p *WorkerPool) Submit(job Job) {
	select {
	case p.jobChan <- job:
	default:
		log.Printf("Job queue is full, dropping job: %s", job)
	}
}

func (p *WorkerPool) AddWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	newID := len(p.workers) + 1
	p.wg.Add(1)
	worker := NewWorker(
		newID,
		p.jobChan,
		p.resultChan,
		context.Background(),
		&p.wg,
	)
	p.workers = append(p.workers, worker)
	go worker.Start()
	log.Printf("Added worker %d", newID)
}

func (p *WorkerPool) RemoveWorker() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workers) == 0 {
		return errors.New("no workers to remove")
	}

	worker := p.workers[len(p.workers)-1]
	worker.cancel()
	p.workers = p.workers[:len(p.workers)-1]
	log.Printf("Removed worker. Total workers now: %d", len(p.workers))
	return nil
}

func (p *WorkerPool) Results() <-chan Result {
	return p.resultChan
}

func (p *WorkerPool) Close() {
	close(p.jobChan)
	p.wg.Wait()
	close(p.resultChan)
}
