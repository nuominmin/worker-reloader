package workerreloader

import (
	"context"
	"sync"
	"time"
)

// WorkerPoolManager defines the interface for managing a pool of worker tasks.
// It provides methods to start tasks on a regular interval or just once, as well as stopping individual or all tasks.
type WorkerPoolManager interface {
	Start(name string, handle func(ctx context.Context) error, interval time.Duration)
	StartOnce(name string, handle func(ctx context.Context) error)
	StartOnceWithDelay(name string, handle func(ctx context.Context) error, delay time.Duration)
	StartOnceWithVersion(name, version string, handle func(ctx context.Context) error)
	StartOnceWithVersionAndDelay(name, version string, handle func(ctx context.Context) error, delay time.Duration)
	Stop(name string)
	StopAll()
	WatchErrors(handler func(error))
	WatchErrorsForName(name string, handler func(error))
	IsRunning(name string) bool
}

// WorkerPool manages a collection of worker tasks.
type WorkerPool struct {
	workers map[string]WorkerManager
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWorkerPool Creates a new instance of WorkerPool.
func NewWorkerPool() WorkerPoolManager {
	return &WorkerPool{
		workers: make(map[string]WorkerManager),
	}
}

// NewWorkerPoolWithContext Creates a new instance of WorkerPool with context.
func NewWorkerPoolWithContext(ctx context.Context) WorkerPoolManager {
	wp := &WorkerPool{
		workers: make(map[string]WorkerManager),
	}

	wp.ctx, wp.cancel = context.WithCancel(ctx)

	go func() {
		<-wp.ctx.Done()
		wp.stopAll()
	}()

	return wp
}

// getWorker safely retrieves a worker with the specified name.
func (wp *WorkerPool) getWorker(name string) (WorkerManager, bool) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	worker, exists := wp.workers[name]
	return worker, exists
}

// newWorker safely creates and sets a worker with the specified name.
func (wp *WorkerPool) newWorker(name string, creator func(name string) WorkerManager) WorkerManager {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	worker := creator(name)
	wp.workers[name] = worker
	return worker
}

// Start initiates a worker task with the specified name, function, and interval. If a task with the same name exists, it restarts it.
// The task is identified by a unique name and will execute the provided handle function in the context of a cancellable context.
func (wp *WorkerPool) Start(name string, handle func(ctx context.Context) error, interval time.Duration) {
	worker, exists := wp.getWorker(name)
	if !exists {
		worker = wp.newWorker(name, NewWorkerTask)
	}
	worker.Start(handle, interval)
}

// StartOnce initiates a worker task that runs only once and then stops.
// This is similar to Start but the task will automatically stop after the first execution.
func (wp *WorkerPool) StartOnce(name string, handle func(ctx context.Context) error) {
	worker, exists := wp.getWorker(name)
	if !exists {
		worker = wp.newWorker(name, NewWorkerTask)
	}
	worker.StartOnce(handle)
}

// StartOnceWithDelay starts a new worker task that executes once after a specified delay.
func (wp *WorkerPool) StartOnceWithDelay(name string, handle func(ctx context.Context) error, delay time.Duration) {
	worker, exists := wp.getWorker(name)
	if exists {
		worker.ExtendStartOnceWithDelay(delay)
		return
	}

	worker = wp.newWorker(name, NewWorkerTask)
	worker.StartOnceWithDelay(handle, delay)
}

// StartOnceWithVersion starts or restarts a worker task with the specified name and version.
// If a task with the same name but a different version exists, it restarts it.
// If the task does not exist or the version is different, it creates and starts a new task.
func (wp *WorkerPool) StartOnceWithVersion(name string, version string, handle func(ctx context.Context) error) {
	worker, exists := wp.getWorker(name)
	if exists && worker.GetVersion() == version {
		return
	}
	worker = wp.newWorker(name, func(name string) WorkerManager {
		return NewVersionedWorkerTask(name, version)
	})
	worker.StartOnce(handle)
}

// StartOnceWithVersionAndDelay starts a task with a specific version and delay if it does not already exist.
// If the task already exists and the version matches, it extends the start delay of the existing task.
// Otherwise, it creates a new worker for the task and starts it with the specified delay.
func (wp *WorkerPool) StartOnceWithVersionAndDelay(name string, version string, handle func(ctx context.Context) error, delay time.Duration) {
	worker, exists := wp.getWorker(name)
	if exists && worker.GetVersion() == version {
		worker.ExtendStartOnceWithDelay(delay)
		return
	}
	worker = wp.newWorker(name, func(name string) WorkerManager {
		return NewVersionedWorkerTask(name, version)
	})
	worker.StartOnceWithDelay(handle, delay)
}

// Stop terminates the worker task identified by the given name.
// It will wait for the task to properly shut down before returning.
func (wp *WorkerPool) Stop(name string) {
	if worker, exists := wp.getWorker(name); exists {
		worker.Stop()
	}
}

// StopAll cancels the context.
func (wp *WorkerPool) StopAll() {
	wp.cancel()
}

// stopAll terminates all currently managed worker tasks.
// This method will block until all tasks have been properly shut down.
func (wp *WorkerPool) stopAll() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	for _, worker := range wp.workers {
		worker.Stop()
	}
}

// WatchErrors applies a global error handler to all current and future workers in the pool.
func (wp *WorkerPool) WatchErrors(handler func(error)) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	for _, worker := range wp.workers {
		worker.WatchErrors(handler) // Apply error handler to each worker
	}
}

// WatchErrorsForName applies an error handler to a specific worker task identified by its name.
func (wp *WorkerPool) WatchErrorsForName(name string, handler func(error)) {
	if worker, exists := wp.getWorker(name); exists {
		worker.WatchErrors(handler) // Apply error handler to the specific worker
	}
}

// IsRunning checks if the worker task with the given name is currently running.
func (wp *WorkerPool) IsRunning(name string) bool {
	if worker, exists := wp.getWorker(name); exists {
		return worker.IsRunning()
	}
	return false
}
