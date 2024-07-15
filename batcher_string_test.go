package microbatching_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nijeshhirpara/microbatching"
	"github.com/stretchr/testify/assert"
)

type StringProcessor struct{}

func (sp *StringProcessor) ProcessBatch(batchID uuid.UUID, jobIDs []uuid.UUID, jobs []string) []microbatching.JobResult[string] {
	results := make([]microbatching.JobResult[string], len(jobs))
	for i, jobID := range jobIDs {
		results[i] = microbatching.JobResult[string]{
			JobID: jobID,
			Data:  jobs[i], // Processed data (mock implementation)
			Error: nil,     // Assume no errors
		}
	}
	return results
}

func TestEmptyStringMicroBatcher(t *testing.T) {
	// Create a new MockBatchProcessor
	processor := &StringProcessor{}

	// Create a new MicroBatcher with a small batch size and frequency for testing
	batchSize := 5
	batchFrequency := 100 * time.Millisecond
	batcher := microbatching.NewMicroBatcher[string](batchSize, batchFrequency, processor)

	// Ensure there are no active batches initially
	assert.Zero(t, batcher.CountActiveBatches(), "Expected no active batches initially")

	// Sleep for a duration greater than the batch frequency to allow potential processing
	time.Sleep(batchFrequency * 2)

	// Assert that there are still no active batches
	assert.Zero(t, batcher.CountActiveBatches(), "Expected no active batches after waiting")

	// Shutdown the MicroBatcher
	batcher.Shutdown()
}
