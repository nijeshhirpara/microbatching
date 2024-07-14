package microbatching

import (
	"sync"
	"time"
)

// Job represents a unit of work to be processed.
type Job struct {
	ID   int         // Unique identifier for the job.
	Data interface{} // Data associated with the job.
}

// JobResult represents the result of a processed job.
type JobResult struct {
	JobID int         // ID of the job this result corresponds to.
	Data  interface{} // Processed data from the job.
	Error error       // Error encountered during processing, if any.
}

// BatchProcessor defines the interface for processing batches of jobs.
type BatchProcessor interface {
	ProcessBatch(jobs []Job) []JobResult // Method to process a batch of jobs and return their results.
}

// MicroBatcher is the main struct that manages job submission and batch processing.
type MicroBatcher struct {
	batchSize       int            // Maximum number of jobs to be included in a batch.
	batchFrequency  time.Duration  // Frequency at which batches are processed.
	batchProcessor  BatchProcessor // Processor that handles the batch processing.
	jobs            []Job          // Slice to store submitted jobs.
	results         []JobResult    // Slice to store results of processed jobs.
	jobsMutex       sync.Mutex     // Mutex to protect access to the jobs slice.
	resultsMutex    sync.Mutex     // Mutex to protect access to the results slice.
	shutdownChannel chan struct{}  // Channel to signal shutdown.
	wg              sync.WaitGroup // WaitGroup to wait for all goroutines to finish.
}

// NewMicroBatcher creates a new MicroBatcher.
func NewMicroBatcher(batchSize int, batchFrequency time.Duration, processor BatchProcessor) *MicroBatcher {
	mb := &MicroBatcher{
		batchSize:       batchSize,
		batchFrequency:  batchFrequency,
		batchProcessor:  processor,
		jobs:            make([]Job, 0, batchSize),
		results:         make([]JobResult, 0),
		shutdownChannel: make(chan struct{}),
	}

	// Start the batching goroutine
	mb.wg.Add(1)
	go mb.startBatching()
	return mb
}

// SubmitJob submits a single job to be processed.
func (mb *MicroBatcher) SubmitJob(job Job) JobResult {
	// Lock the jobs slice to ensure thread-safe access
	mb.jobsMutex.Lock()
	mb.jobs = append(mb.jobs, job)
	jobCount := len(mb.jobs)
	mb.jobsMutex.Unlock()

	// If the batch size is reached, process the batch immediately
	if jobCount >= mb.batchSize {
		go mb.processBatch()
	}

	return JobResult{JobID: job.ID, Data: nil, Error: nil}
}

// startBatching starts the batching process.
func (mb *MicroBatcher) startBatching() {
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
func (mb *MicroBatcher) processBatch() {
	mb.jobsMutex.Lock()
	if len(mb.jobs) == 0 {
		mb.jobsMutex.Unlock()
		return
	}
	// Copy the current jobs to process
	jobsToProcess := mb.jobs
	mb.jobs = make([]Job, 0, mb.batchSize)
	mb.jobsMutex.Unlock()

	// Process the batch and get results
	results := mb.batchProcessor.ProcessBatch(jobsToProcess)

	// Store the results
	mb.resultsMutex.Lock()
	mb.results = append(mb.results, results...)
	mb.resultsMutex.Unlock()
}

// Shutdown shuts down the MicroBatcher after processing all remaining jobs.
func (mb *MicroBatcher) Shutdown() {
	// Signal the shutdown
	close(mb.shutdownChannel)
	// Wait for the batching goroutine to finish
	mb.wg.Wait()
}

// GetResults returns the processed job results.
func (mb *MicroBatcher) GetResults() []JobResult {
	mb.resultsMutex.Lock()
	defer mb.resultsMutex.Unlock()
	return mb.results
}
