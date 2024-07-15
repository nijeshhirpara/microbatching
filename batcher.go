package microbatching

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Job represents a unit of work to be processed.
type Job[T any] struct {
	ID     uuid.UUID         // Unique identifier for the job.
	Data   T                 // Data associated with the job.
	Result chan JobResult[T] // Channel to send the result of the job.
}

// JobResult represents the result of a processed job.
type JobResult[T any] struct {
	JobID uuid.UUID // ID of the job this result corresponds to.
	Data  T         // Processed data from the job.
	Error error     // Error encountered during processing, if any.
}

// BatchProcessor defines the interface for processing batches of jobs.
type BatchProcessor[T any] interface {
	ProcessBatch(batchID uuid.UUID, jobIDs []uuid.UUID, jobs []T) []JobResult[T] // Method to process a batch of jobs and return their results.
}

// MicroBatcher is the main struct that manages job submission and batch processing using Batch instances.
type MicroBatcher[T any] struct {
	batchSize       int               // Maximum number of jobs to be included in a batch.
	batchFrequency  time.Duration     // Frequency at which batches are processed.
	batchProcessor  BatchProcessor[T] // Processor that handles the batch processing.
	batches         []*batch[T]       // List of active batches.
	shutdownChannel chan struct{}     // Channel to signal shutdown.
	wg              sync.WaitGroup    // WaitGroup to wait for all goroutines to finish.
}

// NewMicroBatcher creates a new MicroBatcher and starts batching.
func NewMicroBatcher[T any](batchSize int, batchFrequency time.Duration, processor BatchProcessor[T]) *MicroBatcher[T] {
	mb := &MicroBatcher[T]{
		batchSize:       batchSize,
		batchFrequency:  batchFrequency,
		batchProcessor:  processor,
		batches:         []*batch[T]{},
		shutdownChannel: make(chan struct{}),
	}

	// Start batching process
	mb.StartBatching()

	return mb
}

// SubmitJob submits a job to be processed by the MicroBatcher.
func (mb *MicroBatcher[T]) SubmitJob(data T) (chan JobResult[T], error) {
	// Try to submit the job to an existing batch
	for _, batch := range mb.batches {
		resultChan, err := batch.submitJob(data)

		// Find next batch in the case of ErrBatchFull
		switch err {
		case nil:
			return resultChan, nil
		case ErrBatchFull:
			log.Printf("Batch %s is full. Trying next batch.", batch.ID.String())
			continue
		default:
			return nil, err
		}
	}

	// Create a new batch if no existing batch has space
	newBatchID := uuid.New()
	newBatch := newBatch(newBatchID, mb.batchProcessor, mb.batchSize)
	mb.batches = append(mb.batches, newBatch)

	log.Printf("Job submitted to new batch. BatchID: %s, Batch size: 1", newBatchID.String())
	resultChan, err := newBatch.submitJob(data)
	if err != nil {
		return nil, err
	}

	return resultChan, nil
}

// Function to submit a job and wait for the result.
func (mb *MicroBatcher[T]) SubmitJobAndWait(data T, waitTolerance time.Duration) JobResult[T] {
	resultCh, err := mb.SubmitJob(data)
	if err != nil {
		return JobResult[T]{Error: err}
	}

	// Retry until result is available or timeout is reached.
	timeout := time.After(waitTolerance)
	for {
		select {
		case result := <-resultCh:
			return result
		case <-timeout:
			log.Printf("Timeout waiting for job with data %v to be processed", data)
			return JobResult[T]{Error: errors.New("timeout")}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// StartBatching starts the batching process for all active batches.
func (mb *MicroBatcher[T]) StartBatching() {
	mb.wg.Add(1)
	go func() {
		defer mb.wg.Done()
		ticker := time.NewTicker(mb.batchFrequency)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Process batches on each tick
				for _, batch := range mb.batches {
					batch.processBatch()
				}
			case <-mb.shutdownChannel:
				// Process any remaining jobs on shutdown
				for _, batch := range mb.batches {
					batch.processBatch()
				}
				return
			}
		}
	}()
}

// Shutdown shuts down the MicroBatcher and all active batches.
func (mb *MicroBatcher[T]) Shutdown() {
	log.Println("Shutting down MicroBatcher and all active batches.")
	close(mb.shutdownChannel)
	mb.wg.Wait()
	log.Println("MicroBatcher shutdown complete.")
}

// CountActiveBatches returns a count of active batches.
func (mb *MicroBatcher[T]) CountActiveBatches() int {
	return len(mb.batches)
}

// ErrBatchFull is returned when a job cannot be submitted to a batch because it is full.
var ErrBatchFull = errors.New("batch is full")
