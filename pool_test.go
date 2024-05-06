package workerreloader

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	workerPool := NewWorkerPool()

	workerPool.Start("hello", func(ctx context.Context) error {
		fmt.Println("hello world")
		return nil
	}, 10*time.Second)

	// 模拟并发调用Start
	for i := 0; i < 5; i++ {
		go func(n int) {
			workerPool.Start("once", func(ctx context.Context) error {
				fmt.Printf("Worker iteration %d is running\n", n)
				return nil
			}, 2*time.Second)
			time.Sleep(5 * time.Second)
		}(i)
	}

	// 等待足够长的时间来观察协程的启动和停止
	time.Sleep(300 * time.Second)

	// 最后，确保停止worker
	workerPool.StopAll()
	fmt.Println("All workers stopped")
}
