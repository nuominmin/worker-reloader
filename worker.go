package workerreloader

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkerManager defines the interface for managing worker tasks.
type WorkerManager interface {
	Start(name string, handle func(), interval time.Duration)
	StartOnce(name string, handle func())
	Stop(name string)
	StopAll()
}

// WorkerPool manages a collection of worker tasks.
type WorkerPool struct {
	workers map[string]*WorkerTask
}

// WorkerTask represents a single scheduled task.
type WorkerTask struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.Mutex
}

// NewWorkerPool creates a new instance of WorkerPool.
func NewWorkerPool() WorkerManager {
	return &WorkerPool{
		workers: make(map[string]*WorkerTask),
	}
}

// Start initiates a worker task with the specified name, function, and interval. If a task with the same name exists, it restarts it.
func (wp *WorkerPool) Start(name string, handle func(), interval time.Duration) {
	if _, ok := wp.workers[name]; !ok {
		wp.workers[name] = new(WorkerTask)
	}

	wp.workers[name].Start(handle, interval)
}

// StartOnce initiates a worker task that runs only once and then stops.
func (wp *WorkerPool) StartOnce(name string, handle func()) {
	if _, ok := wp.workers[name]; !ok {
		wp.workers[name] = new(WorkerTask)
	}

	wp.workers[name].StartOnce(handle)
}

// Stop terminates the worker task specified by name.
func (wp *WorkerPool) Stop(name string) {
	if worker, ok := wp.workers[name]; ok {
		worker.Stop()
	}
}

// StopAll terminates all worker tasks managed by the pool.
func (wp *WorkerPool) StopAll() {
	for _, worker := range wp.workers {
		worker.Stop()
	}
}

// Start begins a new work routine and cancels any existing one.
func (wt *WorkerTask) Start(handle func(), interval time.Duration) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	// Cancel any existing work routine.
	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Wait for the existing routine to stop
	}

	// Create a new context for the new work routine
	wt.ctx, wt.cancel = context.WithCancel(context.Background())

	// Start the new work routine
	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-wt.ctx.Done():
				fmt.Println("Worker task stopped")
				return
			case <-ticker.C:
				handle()
			}
		}
	}()
}

// StartOnce begins a new work routine that runs only once.
func (wt *WorkerTask) StartOnce(handle func()) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Wait for the existing routine to stop
	}

	wt.ctx, wt.cancel = context.WithCancel(context.Background())
	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		handle() // Execute task once
	}()
}

// Stop cancels the work routine.
func (wt *WorkerTask) Stop() {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()
	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Ensure the routine has stopped
	}
}
