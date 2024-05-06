# worker-reloader

## 概览
`worker-reloader` 是一个 Go 语言包，用于在一个池中动态管理和重载工作任务。支持间隔执行和一次性执行任务，以及任务的版本管理。

## 功能
- 动态任务管理：启动和停止池中的任务。
- 间隔和一次性执行：按固定间隔或一次性执行任务。
- 版本管理：自动重启更新的任务。

## 安装
```bash
go get github.com/nuominmin/worker-reloader
```

## 简单示例
```go
// 创建任务池
pool := workerreloader.NewWorkerPool()

// 启动定时任务
pool.Start("task1", func(ctx context.Context) error {
    // 任务逻辑
    return nil
}, 10*time.Minute)

// 启动一次性任务
pool.StartOnce("task2", func(ctx context.Context) error {
    // 任务逻辑
    return nil
})

// 带版本的一次性任务
pool.StartOnceWithVersion("task3", "v1.0", func(ctx context.Context) error {
    // 任务逻辑
    return nil
})

// 停止任务
pool.Stop("task1")

// 停止所有任务
pool.StopAll()
```

## 场景：
这些示例代码提供了一个基础框架，显示如何在实际应用中使用 worker-reloader 包的不同方法来处理各种任务。您可以根据具体的应用需求进一步扩展或修改这些示例。

### 定期数据同步：
在需要定期从一个系统同步数据到另一个系统的场景中，可以使用 WorkerPool 的 Start 方法。这可以包括定期从数据库同步最新数据到缓存服务器，或者定期检查外部API以更新本地数据存储。

### 健康检查和监控：
对于需要持续监控应用健康状态或系统性能的情况，可以设置一个定期执行的任务，该任务检查系统的关键指标，并在检测到问题时发出警报或执行修复措施。

### 事件驱动的任务执行：
在需要对某些事件做出一次性响应的应用中，如接收到用户的特定请求或外部事件时启动处理任务，可以使用 StartOnce 方法。这适用于那些不需要重复执行，只在特定条件下触发一次的任务。

### 版本控制的任务部署：
使用 StartOnceWithVersion 方法，可以在系统中灵活部署和管理基于版本的任务。这一方法特别适用于需要根据不同版本执行不同逻辑的软件部署场景。例如，在软件开发和测试阶段，可以部署具有不同版本号的任务来对比不同版本之间的性能和功能表现。此外，当新版本的软件或功能准备好推向生产环境时，StartOnceWithVersion 允许开发者推出新版本任务，自动替换旧版本的任务实例，确保新功能的快速部署和无缝过渡。这种灵活的版本管理不仅提高了部署效率，也降低了因版本更新不当而引起的系统风险。

### 错误监控和异常处理：
使用 WorkerTask 中的 WatchErrors 方法可以实时监控任务执行中的错误并进行相应处理。这对于需要高可靠性的系统尤为重要，能够即时响应并处理任务执行中的任何异常。

### 资源密集型任务的负载管理：
对于需要大量计算资源的任务，比如视频处理或大规模数据分析，可以通过 Start 方法定期执行，并利用锁和上下文管理来确保不同任务间的资源使用不会相互干扰，保证系统的稳定运行。

#### 以下是示例代码，以展示如何实现上述场景：
```go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"github.com/nuominmin/worker-reloader"
)

func handleError(err error) {
	log.Printf("处理错误: %v\n", err)
}


func syncData(ctx context.Context) error {
	// 模拟数据库到缓存服务器的数据同步
	fmt.Println("同步数据库数据到缓存服务器...")
	// 添加同步逻辑
	return nil
}

func healthCheck(ctx context.Context) error {
	// 模拟健康检查逻辑
	fmt.Println("执行系统健康检查...")
	// 添加健康检查逻辑
	return nil
}


func processEvent(ctx context.Context) error {
	// 处理接收到的事件
	fmt.Println("处理特定事件...")
	// 添加事件处理逻辑
	return nil
}

func deployVersion(ctx context.Context) error {
	// 部署新版本
	fmt.Println("部署新版本...")
	// 添加部署逻辑
	return nil
}

func riskyOperation(ctx context.Context) error {
	// 模拟一个可能出错的操作
	fmt.Println("执行高风险操作...")
	// 假设这里出现了一个错误
	return fmt.Errorf("something went wrong")
}

func riskyOperationError(err error) {
	// 错误处理逻辑
	log.Printf("处理错误: %v\n", err)
}

func intensiveTask(ctx context.Context) error {
	fmt.Println("执行资源密集型任务...")
	// 模拟资源密集型操作，如视频处理或大规模数据分析
	time.Sleep(5 * time.Second) // 假设任务执行需要一定时间
	return nil
}

func main() {
	// 创建任务池
	pool := workerreloader.NewWorkerPool()
	defer pool.StopAll()

	// 定期数据同步
	pool.Start("DataSyncTask", syncData, 30*time.Minute) // 每30分钟同步一次
	
	// 健康检查和监控
	pool.Start("HealthCheckTask", healthCheck, 5*time.Minute) // 每5分钟检查一次
	
	// 事件驱动的任务执行
	pool.StartOnce("ProcessEventTask", processEvent) // 只执行一次

	// 版本控制的任务部署
	pool.StartOnceWithVersion("DeployTask", "v1.2", deployVersion) // 部署版本 v1.2
	
	// 错误监控和异常处理
    pool.Start("RiskyTask",riskyOperation, 10*time.Minute) // 每10分钟执行一次
	pool.WatchErrorsForName("RiskyTask",riskyOperationError)
	
	// 资源密集型任务的负载管理
	var mutex sync.Mutex
	pool.Start("IntensiveTask", func(ctx context.Context) error {
		mutex.Lock()    // 确保在同一时间只有一个任务实例在运行
		defer mutex.Unlock()

		return intensiveTask(ctx)
	}, 1*time.Hour) // 每小时执行一次


	select {}
}
```









