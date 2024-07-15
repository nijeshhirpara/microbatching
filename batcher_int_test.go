package microbatching_test

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nijeshhirpara/microbatching"
	"github.com/stretchr/testify/assert"
)

const jobWaitTolerance = 15 * time.Second // Wait tolerance for job result

// TestBatchProcessor is a simple implementation of the BatchProcessor interface for testing.
type TestBatchProcessor struct{}

// ProcessBatch processes a batch of jobs and returns their results.
func (bp *TestBatchProcessor) ProcessBatch(id uuid.UUID, jobIDs []uuid.UUID, jobs []int) []microbatching.JobResult[int] {
	results := make([]microbatching.JobResult[int], len(jobs))
	for i, job := range jobs {
		results[i] = microbatching.JobResult[int]{JobID: jobIDs[i], Data: job * 2}
	}

	// Added delay
	time.Sleep(1 * time.Second)

	log.Printf("Processed Batch ID: %s, size: %d", id, len(jobs))
	return results
}

func TestMicroBatcherWithSubmitJobAndWait(t *testing.T) {
	// Create a new microbatcher with a small batch size and short frequency for testing.
	batchProcessor := &TestBatchProcessor{}
	batcher := microbatching.NewMicroBatcher[int](2, 50*time.Millisecond, batchProcessor)

	// Submit two jobs and wait for their results.
	var wg sync.WaitGroup
	jobs := []int{10, 20, 30}
	results := make([]microbatching.JobResult[int], len(jobs))

	for i, v := range jobs {
		wg.Add(1)
		go func(i int, jobData int) {
			defer wg.Done()
			results[i] = batcher.SubmitJobAndWait(jobData, jobWaitTolerance)
		}(i, v)
	}

	wg.Wait()

	expectedDataValues := []int{20, 40, 60}

	// Verify the results.
	for i, result := range results {
		expectedResult := microbatching.JobResult[int]{JobID: result.JobID, Data: expectedDataValues[i]}

		if result.JobID != expectedResult.JobID || result.Data != expectedResult.Data {
			t.Errorf("Unexpected result for job %d: got %+v, want %+v", result.JobID, result.Data, expectedResult)
		}
	}
}

func TestEmptyMicroBatcher(t *testing.T) {
	// Create a new MockBatchProcessor
	processor := &TestBatchProcessor{}

	// Create a new MicroBatcher with a small batch size and frequency for testing
	batchSize := 5
	batchFrequency := 100 * time.Millisecond
	batcher := microbatching.NewMicroBatcher[int](batchSize, batchFrequency, processor)

	// Ensure there are no active batches initially
	assert.Zero(t, batcher.CountActiveBatches(), "Expected no active batches initially")

	// Sleep for a duration greater than the batch frequency to allow potential processing
	time.Sleep(batchFrequency * 2)

	// Assert that there are still no active batches
	assert.Zero(t, batcher.CountActiveBatches(), "Expected no active batches after waiting")

	// Shutdown the MicroBatcher
	batcher.Shutdown()
}

func TestImmediateProcessing(t *testing.T) {
	processor := &TestBatchProcessor{}
	batcher := microbatching.NewMicroBatcher[int](5, 500*time.Millisecond, processor)

	// Submit jobs and verify that they are processed at the expected frequency
	jobCount := 10
	resultChans := make([]chan microbatching.JobResult[int], jobCount)
	var err error
	for i := 0; i < jobCount; i++ {
		resultChans[i], err = batcher.SubmitJob(i)
		if err != nil {
			t.Errorf("Error submitting a job with data %d, %v", i, err)
		}
	}

	// Verify results
	for i := 0; i < jobCount; i++ {
		select {
		case result := <-resultChans[i]:
			if result.Data != i*2 {
				t.Errorf("Expected result for job ID %d to be %d, got %v", i, i, result.Data)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for job %d to be processed", i)
		}
	}

	// Shutdown the batcher and wait for all jobs to be processed
	batcher.Shutdown()
}

func TestBatchFrequency(t *testing.T) {
	processor := &TestBatchProcessor{}
	batcher := microbatching.NewMicroBatcher[int](1000, 500*time.Millisecond, processor)

	// Submit jobs and verify that they are processed at the expected frequency
	jobCount := 10
	resultChans := make([]chan microbatching.JobResult[int], jobCount)
	var err error
	for i := 0; i < jobCount; i++ {
		resultChans[i], err = batcher.SubmitJob(i)
		if err != nil {
			t.Errorf("Error submitting a job with data %d, %v", i, err)
		}
	}

	// Verify results
	for i := 0; i < jobCount; i++ {
		select {
		case result := <-resultChans[i]:
			if result.Data != i*2 {
				t.Errorf("Expected result for job ID %d to be %d, got %v", i, i, result.Data)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for job %d to be processed", i)
		}
	}

	// Shutdown the batcher and wait for all jobs to be processed
	batcher.Shutdown()
}

// TestMicroBatcherResultChannelBlocking tests channel blocking scenario in MicroBatcher.
// It handles processing delay without any blockers
func TestMicroBatcherResultChannelBlocking(t *testing.T) {
	processor := &TestBatchProcessor{}
	batcher := microbatching.NewMicroBatcher[int](3, 10*time.Second, processor)

	// Submit 50 jobs, which exceeds the buffer size of 3
	jobCount := 50
	resultChans := make([]chan microbatching.JobResult[int], jobCount)
	var err error
	for i := 0; i < jobCount; i++ {
		resultChans[i], err = batcher.SubmitJob(i)
		if err != nil {
			t.Errorf("Error submitting a job with data %d, %v", i, err)
		}
	}

	// Shutdown the batcher and wait for all jobs to be processed
	batcher.Shutdown()

	// Verify results
	for i := 0; i < jobCount; i++ {
		select {
		case result := <-resultChans[i]:
			if result.Data != i*2 {
				t.Errorf("Expected result for job ID %d to be %d, got %v", i, i, result.Data)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for job %d to be processed", i)
		}
	}
}
