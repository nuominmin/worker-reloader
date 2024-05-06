package workerreloader

import (
	"context"
	"log"
	"time"
)

// WorkerManager defines the interface for managing worker tasks.
type WorkerManager interface {
	Start(handle func(ctx context.Context) error, interval time.Duration) (errChan chan error)
	StartOnce(handle func(ctx context.Context) error) (errChan chan error)
	Stop()
}

func NewWorkerTask(name string) WorkerManager {
	return &WorkerTask{
		name: name,
	}
}

// Start initiates a new work routine for the worker task, cancelling any existing one first.
// It first executes the provided 'handle' function immediately and checks for errors. If an error is encountered,
// it sends the error on the returned error channel and exits. If no errors, it continues to execute 'handle'
// at the specified intervals until the context is cancelled.
//
// This method is designed for tasks that need to be run repeatedly at a fixed interval but also require
// immediate first-time execution upon start. The method handles task cancellation and ensures that any
// running instance is properly stopped before starting a new one.
//
// The function uses a mutex to lock the WorkerTask to prevent concurrent execution issues and ensures that
// any running instance is stopped, then sets up a new context and starts the new routine.
func (wt *WorkerTask) Start(handle func(ctx context.Context) error, interval time.Duration) (errChan chan error) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	// 创建一个错误通道
	errChan = make(chan error, 1) // Buffered channel, to ensure goroutine does not block if error is sent and not immediately received.

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
		defer close(errChan)

		// Execute the handle immediately before starting the interval
		if err := handle(wt.ctx); err != nil {
			errChan <- err
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
					errChan <- err
					return
				}
			}
		}
	}()

	return errChan
}

// StartOnce begins a new work routine that runs only once.
func (wt *WorkerTask) StartOnce(handle func(ctx context.Context) error) (errChan chan error) {
	wt.mutex.Lock()
	defer wt.mutex.Unlock()

	// 创建一个错误通道
	errChan = make(chan error, 1) // Buffered channel, to ensure goroutine does not block if error is sent and not immediately received.

	if wt.cancel != nil {
		wt.cancel()
		wt.wg.Wait() // Wait for the existing routine to stop
		log.Printf("Existing worker task stopped before starting a new one-time task, name: %s\n", wt.name)
	}

	wt.ctx, wt.cancel = context.WithCancel(context.Background())
	wt.wg.Add(1)
	go func() {
		defer wt.wg.Done()
		defer close(errChan)

		// Execute task once
		if err := handle(wt.ctx); err != nil {
			errChan <- err
			return
		}

		log.Printf("One-time worker task completed successfully, name: %s\n", wt.name)
	}()

	return errChan
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
