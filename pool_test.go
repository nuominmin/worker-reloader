package workerreloader

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	workerPool := NewWorkerPool()

	workerPool.Start("hello", func(ctx context.Context) error {
		fmt.Println("hello world")
		return errors.New("xxxx")
	}, 1*time.Second)

	// Simulate a concurrent call to Start
	for i := 0; i < 5; i++ {
		go func(n int) {
			workerPool.Start("once", func(ctx context.Context) error {
				fmt.Printf("Worker iteration %d is running\n", n)
				return errors.New("cccccccccc")
			}, 2*time.Second)
			time.Sleep(5 * time.Second)
		}(i)
	}

	workerPool.WatchErrors(func(err error) {
		fmt.Println("error:", err)
	})

	// Wait long enough to watch the coroutine start and stop
	time.Sleep(300 * time.Second)

	// Finally, make sure to stop the worker
	workerPool.StopAll()
	fmt.Println("All workers stopped")
}

func TestWorkerPool_StopAllOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPoolWithContext(ctx)

	pool.Start("testWorker", func(ctx context.Context) error {
		fmt.Println("testWorker start")
		return nil
	}, time.Second*time.Duration(1))

	cancel()

	pool.StopAll()

	time.Sleep(time.Duration(50) * time.Second)
}

// TestWorkerTask_StartOnceWithDelay tests that the task runs after the specified delay.
func TestWorkerTask_StartOnceWithDelay(t *testing.T) {
	fmt.Println(time.Now())
	w := NewVersionedWorkerTask("testWorker", "v1.0.0")
	w.StartOnceWithDelay(func(ctx context.Context) error {
		fmt.Println("testWorker start")
		fmt.Println(time.Now())
		return nil
	}, 60*time.Second)

	w.ExtendStartOnceWithDelay(1 * time.Second)

	time.Sleep(1000 * time.Second)
}

// TestWorkerTask_StartOnce .
func TestWorkerTask_StartOnce(t *testing.T) {
	fmt.Println(time.Now())
	pool := NewWorkerPool()
	for i := 0; i < 5; i++ {
		pool.StartOnce("xxxxxxxxx", func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				fmt.Println("ss start")
				return nil
			}
		})

		time.Sleep(1 * time.Second)
	}

	time.Sleep(1000 * time.Second)
}
