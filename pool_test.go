package workerreloader

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	workerPool := NewWorkerPool(context.Background())

	workerPool.Start("hello", func(ctx context.Context) error {
		fmt.Println("hello world")
		return errors.New("xxxx")
	}, 1*time.Second)

	// 模拟并发调用Start
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

	// 等待足够长的时间来观察协程的启动和停止
	time.Sleep(300 * time.Second)

	// 最后，确保停止worker
	workerPool.StopAll()
	fmt.Println("All workers stopped")
}

func TestWorkerPool_StopAllOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPool(ctx)

	pool.Start("testWorker", func(ctx context.Context) error {
		fmt.Println("testWorker start")
		return nil
	}, time.Second*time.Duration(1))

	cancel()

	pool.StopAll()

	time.Sleep(time.Duration(50) * time.Second)
}
