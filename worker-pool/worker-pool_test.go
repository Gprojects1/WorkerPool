package workerpool

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	pool, err := New(2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		count := 0
		for range pool.Results() {
			count++
			if count >= 2 {
				return
			}
		}
	}()

	pool.Submit("test-1")
	pool.Submit("test-2")
	wg.Wait()
}

func TestDynamicScaling(t *testing.T) {
	pool, _ := New(1)
	defer pool.Close()

	pool.AddWorker()
	if len(pool.workers) != 2 {
		t.Errorf("Expected 2 workers, got %d", len(pool.workers))
	}

	pool.RemoveWorker()
	if len(pool.workers) != 1 {
		t.Errorf("Expected 1 worker, got %d", len(pool.workers))
	}
}

func TestContextCancel(t *testing.T) {
	pool, _ := New(1)
	worker := pool.workers[0]

	worker.cancel()

	select {
	case res := <-pool.Results():
		if res.Err != context.Canceled {
			t.Errorf("Expected context canceled, got %v", res.Err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Worker didn't respond to cancel")
	}

	pool.Close()
}
