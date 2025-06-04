package main

import (
	workerpool "WorkerPool/worker-pool"
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	pool, err := workerpool.New(3)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	go func() {
		for result := range pool.Results() {
			if result.Err != nil {
				if result.Err == context.Canceled {
					log.Printf("Worker %d stopped gracefully", result.WorkerID)
				} else {
					log.Printf("Worker %d error: %v", result.WorkerID, result.Err)
				}
				continue
			}
			log.Printf("Worker %d processed: %s", result.WorkerID, result.Job)
		}
	}()

	for i := 0; i < 10; i++ {
		job := workerpool.Job(fmt.Sprintf("task-%d", i))
		pool.Submit(job)
		log.Printf("Submitted: %s", job)
		time.Sleep(200 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)
	pool.AddWorker()
	time.Sleep(500 * time.Millisecond)
	pool.RemoveWorker()

	time.Sleep(2 * time.Second)
}
