# Test Runner Service

## How it works

This service receives test jobs from Redis and processes them concurrently using a worker pool.

**Flow:**
1. Redis sends test jobs to a Subscriber
2. Subscriber puts jobs into a channel (`jobChan`)
3. Multiple workers (goroutines) compete to grab jobs from the channel
4. Each worker processes its job independently using the Test Runner
5. Workers send results to another channel (`resultsChan`)
6. A result processor reads results and sends them to an API

**Concurrency:** If you have 5 workers, 5 tests can run at the same time. When a worker finishes, it immediately grabs the next available job.

**Shutdown:** When you restart the service, it stops accepting new jobs, lets workers finish their current jobs, then exits cleanly.

**Configuration:** Set `maxWorkers` to control how many tests run concurrently.