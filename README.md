# MicroBatching Library

MicroBatching is a library for processing individual tasks in small batches. This can improve throughput by reducing the number of requests made to a downstream system.

### Features

- Submit a single job and get a result.
- Process accepted jobs in batches using a `BatchProcessor`.
- Configure the batching behavior (batch size and batch frequency).
- Graceful shutdown that ensures all previously accepted jobs are processed.

### Installation

To install the MicroBatching library, you can use `go get`:

```sh
go get github.com/yourusername/microbatch
```

### Usage

#### Define a BatchProcessor

First, define a type that implements the BatchProcessor interface. This interface requires a single method, ProcessBatch, that takes a slice of jobs and returns a slice of JobResult.

```go
package main

import (
    "fmt"
    "microbatch"
)

// MyBatchProcessor is a simple implementation of the BatchProcessor interface.
type MyBatchProcessor struct{}

// ProcessBatch processes a batch of jobs and returns their results.
func (bp *MyBatchProcessor) ProcessBatch(jobs []string) []microbatch.JobResult[string] {
    results := make([]microbatch.JobResult[string], len(jobs))
    for i, job := range jobs {
        results[i] = microbatch.JobResult[string]{JobID: i, Data: fmt.Sprintf("Processed: %v", job), Error: nil}
    }
    return results
}
```

#### Create a MicroBatcher

Create an instance of `MicroBatcher` with the desired batch size, batch frequency, and the batch processor.

```go
package main

import (
    "fmt"
    "time"
    "microbatch"
)

func main() {
    processor := &MyBatchProcessor{}
    batcher := microbatch.NewMicroBatcher[string](5, 2*time.Second, processor)

    // Submit jobs to the batcher
    for i := 0; i < 20; i++ {
        job := microbatch.Job[string]{ID: i, Data: fmt.Sprintf("JobData %d", i)}
        resultChan := batcher.SubmitJob(job)
        go func(jobID int, resultChan chan microbatch.JobResult[string]) {
            result := <-resultChan
            fmt.Printf("Job ID: %d, Result: %v\n", result.JobID, result.Data)
        }(i, resultChan)
    }

    // Allow some time for processing
    time.Sleep(10 * time.Second)
    // Shutdown the batcher and wait for all jobs to be processed
    batcher.Shutdown()
}
```

## API Reference

### Job[T any]
Represents a unit of work to be processed.

- ID: Unique identifier for the job.
- Data: Data associated with the job.
- Result: Channel to send the result of the job.

### JobResult[T any]

Represents the result of a processed job.

- JobID: ID of the job this result corresponds to.
- Data: Processed data from the job.
- Error: Error encountered during processing, if any.

### BatchProcessor[T any]

Interface for processing batches of jobs.

- `ProcessBatch(jobs []T) []JobResult[T]`: Method to process a batch of jobs and return their results.

### MicroBatcher[T any]

Main struct that manages job submission and batch processing.

- `NewMicroBatcher(batchSize int, batchFrequency time.Duration, processor BatchProcessor[T]) *MicroBatcher[T]`: Creates a new MicroBatcher.
- `SubmitJob(job Job[T]) chan JobResult[T]`: Submits a single job to be processed.
- `Shutdown()`: Shuts down the MicroBatcher after processing all remaining jobs.