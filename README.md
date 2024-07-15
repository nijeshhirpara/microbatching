# MicroBatching Library

The `microbatching` package provides a framework for managing and processing jobs in batches. This is useful for systems that need to process a high volume of jobs efficiently by grouping them into batches and processing each batch at a defined frequency or when a batch size is reached.

### Features

- Configure the batching behavior (batch size and batch frequency).
- Submit a single job and get a result.
- Create batches either when a specified batch size is reached or after a fixed interval.
- Process jobs in batches using a `BatchProcessor`.
- Ensure unique job IDs to avoid conflicts.
- Ensure uniqueness in batch processing to handle concurrency
- Graceful shutdown that ensures all previously accepted jobs are processed.
- Logging to track job submission and batch processing.

### Installation

To install the MicroBatching library, you can use `go get`:

```sh
go get github.com/yourusername/microbatch
```

### Usage

#### Creating a Batch Processor

First, define a type that implements the `BatchProcessor` interface. 

```go
package main

import (
	"github.com/nijeshhirpara/microbatching"
	"github.com/google/uuid"
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

```

#### Create a MicroBatcher

Create an instance of `MicroBatcher` with the desired batch size, batch frequency, and the batch processor.

```go
package main

import (
	"log"
	"time"

	"github.com/nijeshhirpara/microbatching"
)

func main() {
	processor := &StringProcessor{}
	batchSize := 5
	batchFrequency := 2 * time.Second

	batcher := microbatching.NewMicroBatcher[string](batchSize, batchFrequency, processor)

	// Submit jobs
	for i := 0; i < 10; i++ {
		resultChan, err := batcher.SubmitJob("job data")
		if err != nil {
			log.Fatalf("Failed to submit job: %v", err)
		}
		go func() {
			result := <-resultChan
			if result.Error != nil {
				log.Printf("Job failed: %v", result.Error)
			} else {
				log.Printf("Job completed: %v", result.Data)
			}
		}()
	}

	// Shutdown the batcher after some time
	time.Sleep(10 * time.Second)
	batcher.Shutdown()
}
```

## API Reference

### Job[T any]
Represents a unit of work to be processed.

```go
type Job[T any] struct {
	ID     uuid.UUID         // Unique identifier for the job.
	Data   T                 // Data associated with the job.
	Result chan JobResult[T] // Channel to send the result of the job.
}
```

### JobResult[T any]

Represents the result of a processed job.

```go
type JobResult[T any] struct {
	JobID uuid.UUID // ID of the job this result corresponds to.
	Data  T         // Processed data from the job.
	Error error     // Error encountered during processing, if any.
}
```

### BatchProcessor[T any]

Defines the interface for processing batches of jobs.

```go
type BatchProcessor[T any] interface {
	ProcessBatch(batchID uuid.UUID, jobIDs []uuid.UUID, jobs []T) []JobResult[T]
}
```

### MicroBatcher[T any]

Manages job submission and batch processing.

#### `NewMicroBatcher`

Creates a new MicroBatcher.

```go
func NewMicroBatcher[T any](batchSize int, batchFrequency time.Duration, processor BatchProcessor[T]) *MicroBatcher[T]
```
- `batchSize`: Maximum number of jobs to be included in a batch.
- `batchFrequency`: Frequency at which batches are processed.
- `processor`: Processor that handles the batch processing.

#### `SubmitJob`

Submits a single job to be processed.

```go
func (mb *MicroBatcher[T]) SubmitJob(data T) (chan JobResult[T], error)
```
- `data`: The data associated with the job.
- Returns a channel to receive the job result and an error if the job could not be submitted.

#### `SubmitJobAndWait`

Submits a single job to be processed and wait for the results. It will timeout after passing wait tolerance.
```go
func (mb *MicroBatcher[T]) SubmitJobAndWait(data T, waitTolerance time.Duration) JobResult[T]
```
- `data`: The data associated with the job.
- Returns job result which include an error if the job could not be submitted or timeout.

#### `CountActiveBatches`

Returns a count of active batches. It will help polling status of longer running batches.

```go
func (mb *MicroBatcher[T]) CountActiveBatches() int
```

#### `Shutdown`

Shuts down the MicroBatcher after processing all remaining jobs.

```go
func (mb *MicroBatcher[T]) Shutdown()
```