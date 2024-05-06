package workerreloader

import (
	"context"
	"sync"
	"time"
)

// WorkerPoolManager defines the interface for managing a pool of worker tasks.
// It provides methods to start tasks on a regular interval or just once, as well as stopping individual or all tasks.
type WorkerPoolManager interface {
	Start(name string, handle func(ctx context.Context) error, interval time.Duration) (errChan chan error)
	StartOnce(name string, handle func(ctx context.Context) error) (errChan chan error)
	Stop(name string)
	StopAll()
}

// WorkerPool manages a collection of worker tasks.
type WorkerPool struct {
	workers map[string]WorkerManager
}

// WorkerTask represents a single scheduled task.
type WorkerTask struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.Mutex
}

// NewWorkerPool creates a new instance of WorkerPool.
func NewWorkerPool() WorkerPoolManager {
	return &WorkerPool{
		workers: make(map[string]WorkerManager),
	}
}

// Start initiates a worker task with the specified name, function, and interval. If a task with the same name exists, it restarts it.
// The task is identified by a unique name and will execute the provided handle function in the context of a cancellable context.
func (wp *WorkerPool) Start(name string, handle func(ctx context.Context) error, interval time.Duration) (errChan chan error) {
	if _, ok := wp.workers[name]; !ok {
		wp.workers[name] = NewWorkerTask(name)
	}

	return wp.workers[name].Start(handle, interval)
}

// StartOnce initiates a worker task that runs only once and then stops.
// This is similar to Start but the task will automatically stop after the first execution.
func (wp *WorkerPool) StartOnce(name string, handle func(ctx context.Context) error) (errChan chan error) {
	if _, ok := wp.workers[name]; !ok {
		wp.workers[name] = new(WorkerTask)
	}

	return wp.workers[name].StartOnce(handle)
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
