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

## 使用示例
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