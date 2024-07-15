package microbatching

import (
	"log"
	"sync"

	"github.com/google/uuid"
)

// batch manages a batch of jobs and their processing.
type batch[T any] struct {
	ID          uuid.UUID                       // Unique identifier for the batch.
	jobs        map[uuid.UUID]Job[T]            // Map to store jobs by their IDs.
	jobsMutex   sync.Mutex                      // Mutex to protect concurrent access to jobs map.
	processor   BatchProcessor[T]               // Processor that handles the batch processing.
	resultChans map[uuid.UUID]chan JobResult[T] // Channels to send results back to the respective jobs.
	batchSize   int                             // Maximum number of jobs to be included in a batch.
	processing  bool                            // Flag to indicate if the batch is currently being processed.
}

// newBatch creates a new Batch instance.
func newBatch[T any](batchID uuid.UUID, processor BatchProcessor[T], batchSize int) *batch[T] {
	return &batch[T]{
		ID:          batchID,
		jobs:        make(map[uuid.UUID]Job[T]),
		jobsMutex:   sync.Mutex{},
		processor:   processor,
		resultChans: make(map[uuid.UUID]chan JobResult[T]),
		batchSize:   batchSize,
		processing:  false,
	}
}

// submitJob submits a job to the batch.
func (b *batch[T]) submitJob(data T) (chan JobResult[T], error) {
	b.jobsMutex.Lock()
	defer b.jobsMutex.Unlock()

	if len(b.jobs) >= b.batchSize {
		return nil, ErrBatchFull
	}

	jobID := uuid.New()
	resultChan := make(chan JobResult[T], 1)

	b.jobs[jobID] = Job[T]{ID: jobID, Data: data, Result: resultChan}
	b.resultChans[jobID] = resultChan

	log.Printf("Job submitted to Batch. BatchID: %s, JobID: %s", b.ID.String(), jobID.String())

	// Check if batch size is reached and process batch immediately
	if len(b.jobs) >= b.batchSize && !b.processing {
		go b.processBatch()
	}

	return resultChan, nil
}

// processBatch processes the current batch of jobs.
func (b *batch[T]) processBatch() {
	b.jobsMutex.Lock()
	defer b.jobsMutex.Unlock()

	// Terminate if no jobs in a batch or it's already processing
	if len(b.jobs) == 0 || b.processing {
		return
	}

	b.processing = true
	log.Printf("Processing batch. BatchID: %s, Batch size: %d", b.ID.String(), len(b.jobs))

	// Copy the current jobs to process
	jobIDsToProcess := make([]uuid.UUID, 0, len(b.jobs))
	jobData := make([]T, 0, len(b.jobs))
	for jobID, job := range b.jobs {
		jobIDsToProcess = append(jobIDsToProcess, jobID)
		jobData = append(jobData, job.Data)
	}

	// Process the batch and get results
	results := b.processor.ProcessBatch(b.ID, jobIDsToProcess, jobData)

	// Send the results back through their respective channels
	for _, result := range results {
		if resultChan, ok := b.resultChans[result.JobID]; ok {
			resultChan <- result
			close(resultChan)
			delete(b.jobs, result.JobID)
			delete(b.resultChans, result.JobID)
			log.Printf("Result sent for JobID: %s. BatchID: %s", result.JobID.String(), b.ID.String())
		}
	}

	b.processing = false
	log.Printf("Batch processing complete. BatchID: %s", b.ID.String())
}
