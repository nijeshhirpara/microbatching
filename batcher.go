package microbatching

import (
	"sync"
	"time"
)

// Job represents a unit of work to be processed.
type Job[T any] struct {
	ID     int               // Unique identifier for the job.
	Data   T                 // Data associated with the job.
	Result chan JobResult[T] // Channel to send the result of the job.
}

// JobResult represents the result of a processed job.
type JobResult[T any] struct {
	JobID int   // ID of the job this result corresponds to.
	Data  T     // Processed data from the job.
	Error error // Error encountered during processing, if any.
}

// BatchProcessor defines the interface for processing batches of jobs.
type BatchProcessor[T any] interface {
	ProcessBatch(jobs []T) []JobResult[T] // Method to process a batch of jobs and return their results.
}

// MicroBatcher is the main struct that manages job submission and batch processing.
type MicroBatcher[T any] struct {
	batchSize       int               // Maximum number of jobs to be included in a batch.
	batchFrequency  time.Duration     // Frequency at which batches are processed.
	batchProcessor  BatchProcessor[T] // Processor that handles the batch processing.
	jobs            []Job[T]          // Slice to store submitted jobs.
	jobsMutex       sync.Mutex        // Mutex to protect access to the jobs slice.
	shutdownChannel chan struct{}     // Channel to signal shutdown.
	wg              sync.WaitGroup    // WaitGroup to wait for all goroutines to finish.
}

// NewMicroBatcher creates a new MicroBatcher.
func NewMicroBatcher[T any](batchSize int, batchFrequency time.Duration, processor BatchProcessor[T]) *MicroBatcher[T] {
	mb := &MicroBatcher[T]{
		batchSize:       batchSize,
		batchFrequency:  batchFrequency,
		batchProcessor:  processor,
		jobs:            make([]Job[T], 0, batchSize),
		shutdownChannel: make(chan struct{}),
	}

	// Start the batching goroutine
	mb.wg.Add(1)
	go mb.startBatching()
	return mb
}

// SubmitJob submits a single job to be processed.
func (mb *MicroBatcher[T]) SubmitJob(job Job[T]) chan JobResult[T] {
	// Create a result channel for this job
	job.Result = make(chan JobResult[T], 1)

	// Lock the jobs slice to ensure thread-safe access
	mb.jobsMutex.Lock()
	mb.jobs = append(mb.jobs, job)
	jobCount := len(mb.jobs)
	mb.jobsMutex.Unlock()

	// If the batch size is reached, process the batch immediately
	if jobCount >= mb.batchSize {
		go mb.processBatch()
	}

	return job.Result
}

// startBatching starts the batching process.
func (mb *MicroBatcher[T]) startBatching() {
	defer mb.wg.Done()
	ticker := time.NewTicker(mb.batchFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Process the batch on each tick
			mb.processBatch()
		case <-mb.shutdownChannel:
			// Process any remaining jobs on shutdown
			mb.processBatch()
			return
		}
	}
}

// processBatch processes the current batch of jobs.
func (mb *MicroBatcher[T]) processBatch() {
	mb.jobsMutex.Lock()
	if len(mb.jobs) == 0 {
		mb.jobsMutex.Unlock()
		return
	}
	// Copy the current jobs to process
	jobsToProcess := mb.jobs
	mb.jobs = make([]Job[T], 0, mb.batchSize)
	mb.jobsMutex.Unlock()

	// Extract job data for processing
	jobData := make([]T, len(jobsToProcess))
	for i, job := range jobsToProcess {
		jobData[i] = job.Data
	}

	// Process the batch and get results
	results := mb.batchProcessor.ProcessBatch(jobData)

	// Send the results back through their respective channels
	for _, result := range results {
		for _, job := range jobsToProcess {
			if job.ID == result.JobID {
				job.Result <- result
				close(job.Result)
				break
			}
		}
	}
}

// Shutdown shuts down the MicroBatcher after processing all remaining jobs.
func (mb *MicroBatcher[T]) Shutdown() {
	// Signal the shutdown
	close(mb.shutdownChannel)
	// Wait for the batching goroutine to finish
	mb.wg.Wait()
}
