package microbatching_test

import (
	"testing"
	"time"

	"github.com/nijeshhirpara/microbatching"
)

// TestBatchProcessor is a simple implementation of the BatchProcessor interface for testing.
type TestBatchProcessor struct{}

// ProcessBatch processes a batch of jobs and returns their results.
func (bp *TestBatchProcessor) ProcessBatch(jobs []microbatching.Job) []microbatching.JobResult {
	results := make([]microbatching.JobResult, len(jobs))
	for i, job := range jobs {
		results[i] = microbatching.JobResult{JobID: job.ID, Data: job.Data, Error: nil}
	}
	return results
}

func TestMicroBatcher(t *testing.T) {
	processor := &TestBatchProcessor{}
	batcher := microbatching.NewMicroBatcher(5, 1*time.Second, processor)

	jobCount := 10
	// Submit jobs to the batcher
	for i := 0; i < jobCount; i++ {
		job := microbatching.Job{ID: i, Data: i}
		batcher.SubmitJob(job)
	}

	// Allow some time for processing
	time.Sleep(2 * time.Second)
	// Shutdown the batcher and wait for all jobs to be processed
	batcher.Shutdown()

	// Retrieve the results
	results := batcher.GetResults()
	if len(results) != jobCount {
		t.Errorf("Expected %d results, got %d", jobCount, len(results))
	}

	// Verify the results
	for i, result := range results {
		if result.JobID != i || result.Data != i {
			t.Errorf("Expected result for job ID %d to be %d, got %v", i, i, result.Data)
		}
	}
}
