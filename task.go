package workerreloader

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerManager defines an interface for managing worker tasks.
// It includes methods to start, stop tasks, and manage task versions.
type WorkerManager interface {
	Start(handle func(ctx context.Context) error, interval time.Duration)
	StartOnce(handle func(ctx context.Context) error)
	StartOnceWithDelay(handle func(ctx context.Context) error, delay time.Duration)
	ExtendStartOnceWithDelay(delay time.Duration)
	GetVersion() string
	Stop()
	WatchErrors(handler func(error))
	IsRunning() bool
}

// WorkerTask represents a scheduled task with mechanisms to handle
// task execution repeatedly or once based on the given context.
type WorkerTask struct {
	name     string
	version  string     // The version identifier used to store the task
	errChan  chan error // Memory error channel
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mutex    sync.Mutex
	running  int32         // It is used to track whether a task is running
	timer    *time.Timer   // Timer that delays task execution
	delayDur time.Duration // Records the delay time for the renewal function
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

// IsRunning returns whether the task is currently running.
func (wt *WorkerTask) IsRunning() bool {
	return atomic.LoadInt32(&wt.running) == 1
}

// Start initiates a work routine that executes a given handle function repeatedly at specified intervals.
// Before beginning the interval execution, it performs a one-time immediate execution of the handle function.
// If any error occurs during the initial execution or subsequent interval executions, it is sent on the errChan.
//
// Important behaviors:
//   - Cancels any previously running routine associated with this WorkerTask before starting a new one.
//   - Creates a new context for the new work routine, ensuring a fresh execution environment.
//   - Checks if the context is already cancelled right after the initial execution of the handle,
//     which helps in preventing unnecessary work if the task was cancelled during the initial call.
//   - If the context is not cancelled, it schedules the next execution to maintain the defined interval,
//     considering the time taken by the handle function to ensure there is always a pause of at least the specified interval.
//   - This method uses a mutex to ensure that no two routines can run concurrently for the same task,
//     hence avoiding race conditions and potential data inconsistencies.
//
// Parameters:
// - handle: A function to execute which can return an error. This function is called repeatedly until the task is cancelled.
// - interval: The time.Duration between each execution of the handle function, ensuring at least this much delay between finishes of one call and the start of the next.
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
	atomic.StoreInt32(&wt.running, 1) // Set the task to the running state

	// Start the new work routine
	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		defer close(wt.errChan)
		defer atomic.StoreInt32(&wt.running, 0)

		for {
			startTime := time.Now() // Record the start time of the handle execution

			if err := handle(wt.ctx); err != nil {
				select {
				case wt.errChan <- err:
				default:
				}
			}

			// Calculate the next tick time by adding the interval to the start time
			nextTickTime := startTime.Add(interval)

			select {
			case <-wt.ctx.Done():
				log.Printf("Worker task stopped, name: %s\n", wt.name)
				return
			case <-time.After(time.Until(nextTickTime)):
				// Wait until the next tick time
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
	atomic.StoreInt32(&wt.running, 1) // Set the task to the running state

	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		defer close(wt.errChan)
		defer atomic.StoreInt32(&wt.running, 0)

		// Execute task once
		if err := handle(wt.ctx); err != nil {
			select {
			case wt.errChan <- err:
			default:
			}
		}

		log.Printf("One-time worker task completed successfully, name: %s\n", wt.name)
	}()

}

// StartOnceWithDelay initiates a new work routine that runs only once, but after a specified delay.
// This method now reuses StartOnce to execute the task after the delay.
func (wt *WorkerTask) StartOnceWithDelay(handle func(ctx context.Context) error, delay time.Duration) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait()
	}

	wt.delayDur = delay
	wt.timer = time.AfterFunc(delay, func() {
		// Once the delay is over, simply call StartOnce to run the task.
		wt.StartOnce(handle)
	})
}

// ExtendStartOnceWithDelay extends the delay of a task before it starts.
// This method cancels the current timer and restarts it with a new delay.
func (wt *WorkerTask) ExtendStartOnceWithDelay(delay time.Duration) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	// 如果定时器存在且任务还未运行，则取消定时器并重新设置
	if wt.timer != nil && atomic.LoadInt32(&wt.running) == 0 {
		wt.timer.Stop()
		wt.timer.Reset(delay)
		wt.delayDur = delay
	}
}

// Stop terminates the current work routine and stops any running timer for delayed tasks.
func (wt *WorkerTask) Stop() {
	go func() {
		wt.mutex.Lock()
		defer wt.mutex.Unlock()

		// Cancel any active timer to prevent future execution of delayed tasks.
		if wt.timer != nil {
			if !wt.timer.Stop() {
				// If the timer has been triggered but not yet executed, you can try to clean it
				<-wt.timer.C
			}
			wt.timer = nil
		}

		// Cancel the context to stop any ongoing task.
		if wt.cancel != nil {
			wt.cancel()
			wt.wg.Wait() // Ensure the task has fully stopped.
		}
	}()
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
