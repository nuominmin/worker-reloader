package workerreloader

import (
	"context"
	"log"
	"sync"
	"time"
)

// WorkerManager defines an interface for managing worker tasks.
// It includes methods to start, stop tasks, and manage task versions.
type WorkerManager interface {
	Start(handle func(ctx context.Context) error, interval time.Duration)
	StartOnce(handle func(ctx context.Context) error)
	GetVersion() string
	Stop()
	WatchErrors(handler func(error))
}

// WorkerTask represents a scheduled task with mechanisms to handle
// task execution repeatedly or once based on the given context.
type WorkerTask struct {
	name    string
	version string     // 用于存储任务的版本标识
	errChan chan error // 存储错误通道
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mutex   sync.Mutex
}

// NewWorkerTask creates a new task instance with a specified name.
func NewWorkerTask(name string) WorkerManager {
	return &WorkerTask{
		name: name,
	}
}

// NewVersionedWorkerTask creates a new task instance with a specified name and version.
func NewVersionedWorkerTask(name, version string) WorkerManager {
	return &WorkerTask{
		name:    name,
		version: version,
	}
}

// Start initiates a work routine that executes a given handle function repeatedly at specified intervals.
// Before beginning the interval execution, it performs a one-time immediate execution of the handle function.
// If any error occurs during the initial execution or subsequent interval executions, it is sent on the errChan.
//
// Important behaviors:
//   - Cancels any previously running routine associated with this WorkerTask before starting a new one.
//   - Creates a new context for the new work routine, ensuring fresh execution environment.
//   - Checks if the context is already cancelled right after the initial execution of the handle,
//     which helps in preventing unnecessary work if the task was cancelled during the initial call.
//   - If the context is not cancelled, it sets up a ticker to execute the handle function repeatedly at the defined interval.
//   - This method uses a mutex to ensure that no two routines can run concurrently for the same task,
//     hence avoiding race conditions and potential data inconsistencies.
//
// Parameters:
// - handle: A function to execute which can return an error. This function is called repeatedly until the task is cancelled.
// - interval: The time.Duration between each execution of the handle function.
//
// The method manages the worker task lifecycle, including stopping and cleaning up properly before starting a new routine.
func (wt *WorkerTask) Start(handle func(ctx context.Context) error, interval time.Duration) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	// Cancel any existing work routine.
	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Wait for the existing routine to stop
	}

	// Create a new context for the new work routine
	wt.ctx, wt.cancel = context.WithCancel(context.Background())

	// Buffer size of 1 to prevent blocking when sending errors
	wt.errChan = make(chan error, 1)

	// Start the new work routine
	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		defer close(wt.errChan)

		// Execute the handle immediately before starting the interval
		if err := handle(wt.ctx); err != nil {
			wt.errChan <- err
			return
		}

		// Check if the context is already cancelled after handle execution
		select {
		case <-wt.ctx.Done():
			log.Printf("Worker task stopped after initial execution, name: %s\n", wt.name)
			return
		default:
			// Continue if not cancelled
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-wt.ctx.Done():
				log.Printf("Worker task stopped, name: %s\n", wt.name)
				return
			case <-ticker.C:
				if err := handle(wt.ctx); err != nil {
					wt.errChan <- err
					return
				}
			}
		}
	}()

}

// StartOnce initiates a new work routine that runs only once.
func (wt *WorkerTask) StartOnce(handle func(ctx context.Context) error) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Wait for the existing routine to stop
		log.Printf("Existing worker task stopped before starting a new one-time task, name: %s\n", wt.name)
	}

	wt.ctx, wt.cancel = context.WithCancel(context.Background())

	// Buffer size of 1 to prevent blocking when sending errors
	wt.errChan = make(chan error, 1)

	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		defer close(wt.errChan)

		// Execute task once
		if err := handle(wt.ctx); err != nil {
			wt.errChan <- err
			return
		}

		log.Printf("One-time worker task completed successfully, name: %s\n", wt.name)
	}()

}

// Stop terminates the current work routine.
func (wt *WorkerTask) Stop() {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()
	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Ensure the routine has stopped
	}
}

// GetVersion returns the version of the current task.
func (wt *WorkerTask) GetVersion() string {
	return wt.version
}

// WatchErrors listens for errors from the task's errChan and handles them using the provided handler function.
func (wt *WorkerTask) WatchErrors(handler func(error)) {
	go func() {
		for {
			select {
			case err, ok := <-wt.errChan:
				if !ok {
					return // errChan has been closed
				}
				handler(err) // Handle the received error
			case <-wt.ctx.Done():
				return // The task context has been cancelled, stop listening
			}
		}
	}()
}
