package workerreloader

import (
	"context"
	"time"
)

// WorkerPoolManager defines the interface for managing a pool of worker tasks.
// It provides methods to start tasks on a regular interval or just once, as well as stopping individual or all tasks.
type WorkerPoolManager interface {
	Start(name string, handle func(ctx context.Context) error, interval time.Duration)
	StartOnce(name string, handle func(ctx context.Context) error)
	StartOnceWithVersion(name string, version string, handle func(ctx context.Context) error)
	Stop(name string)
	StopAll()
	WatchErrors(handler func(error))
	WatchErrorsForName(name string, handler func(error))
	IsRunning(name string) bool
}

// WorkerPool manages a collection of worker tasks.
type WorkerPool struct {
	workers map[string]WorkerManager
}

// NewWorkerPool creates a new instance of WorkerPool.
func NewWorkerPool() WorkerPoolManager {
	return &WorkerPool{
		workers: make(map[string]WorkerManager),
	}
}

// Start initiates a worker task with the specified name, function, and interval. If a task with the same name exists, it restarts it.
// The task is identified by a unique name and will execute the provided handle function in the context of a cancellable context.
func (wp *WorkerPool) Start(name string, handle func(ctx context.Context) error, interval time.Duration) {
	if _, exists := wp.workers[name]; !exists {
		wp.workers[name] = NewWorkerTask(name)
	}

	wp.workers[name].Start(handle, interval)
}

// StartOnce initiates a worker task that runs only once and then stops.
// This is similar to Start but the task will automatically stop after the first execution.
func (wp *WorkerPool) StartOnce(name string, handle func(ctx context.Context) error) {
	if _, exists := wp.workers[name]; !exists {
		wp.workers[name] = NewWorkerTask(name)
	}

	wp.workers[name].StartOnce(handle)
}

// StartOnceWithVersion starts or restarts a worker task with the specified name and version.
// If a task with the same name but a different version exists, it restarts it.
// If the task does not exist or the version is different, it creates and starts a new task.
func (wp *WorkerPool) StartOnceWithVersion(name string, version string, handle func(ctx context.Context) error) {
	if _, exists := wp.workers[name]; exists {
		return
	}
	wp.workers[name] = NewVersionedWorkerTask(name, version)
	wp.workers[name].StartOnce(handle)
}

// Stop terminates the worker task identified by the given name.
// It will wait for the task to properly shut down before returning.
func (wp *WorkerPool) Stop(name string) {
	if worker, ok := wp.workers[name]; ok {
		worker.Stop()
	}
}

// StopAll terminates all currently managed worker tasks.
// This method will block until all tasks have been properly shut down.
func (wp *WorkerPool) StopAll() {
	for _, worker := range wp.workers {
		worker.Stop()
	}
}

// WatchErrors applies a global error handler to all current and future workers in the pool.
func (wp *WorkerPool) WatchErrors(handler func(error)) {
	for _, worker := range wp.workers {
		worker.WatchErrors(handler) // Apply error handler to each worker
	}
}

// WatchErrorsForName applies an error handler to a specific worker task identified by its name.
func (wp *WorkerPool) WatchErrorsForName(name string, handler func(error)) {
	if worker, exists := wp.workers[name]; exists {
		worker.WatchErrors(handler) // Apply error handler to the specific worker
	}
}

// IsRunning checks if the worker task with the given name is currently running.
func (wp *WorkerPool) IsRunning(name string) bool {
	if worker, exists := wp.workers[name]; exists {
		return worker.IsRunning()
	}
	return false
}
