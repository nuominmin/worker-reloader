package workerreloader

import (
	"fmt"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	workerPool := NewWorkerPool()

	// 模拟并发调用Start
	for i := 0; i < 5; i++ {
		n := i
		//go func(n int) {
		fmt.Printf("Starting worker iteration %d\n", n)
		workerPool.Start("one", func() {
			fmt.Printf("Worker iteration %d is running\n", n)
		}, 2*time.Millisecond)
		time.Sleep(5 * time.Second)
		//}(i)
	}

	// 等待足够长的时间来观察协程的启动和停止
	time.Sleep(15 * time.Second)

	// 最后，确保停止worker
	workerPool.StopAll()
	fmt.Println("All workers stopped")
}
