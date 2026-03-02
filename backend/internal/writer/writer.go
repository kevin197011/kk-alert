package writer

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// WriteJob represents a database write operation.
type WriteJob struct {
	Table  string
	Record interface{}
	Done   chan error
}

// AsyncWriter provides asynchronous database writes with batching.
// Optimized for high-concurrency scenarios where data can be queued.
type AsyncWriter struct {
	db            *gorm.DB
	queue         chan WriteJob
	stopChan      chan struct{}
	wg            sync.WaitGroup
	batchSize     int
	flushInterval time.Duration
}

// NewAsyncWriter creates a new async writer.
// batchSize: number of records to batch together
// queueSize: size of the write queue (should be large for 1000+ rules)
func NewAsyncWriter(db *gorm.DB, batchSize, queueSize int) *AsyncWriter {
	if batchSize <= 0 {
		batchSize = 100
	}
	if queueSize <= 0 {
		queueSize = 5000
	}

	w := &AsyncWriter{
		db:            db,
		queue:         make(chan WriteJob, queueSize),
		stopChan:      make(chan struct{}),
		batchSize:     batchSize,
		flushInterval: 5 * time.Second,
	}

	// Start worker goroutines
	for i := 0; i < 4; i++ {
		w.wg.Add(1)
		go w.worker()
	}

	return w
}

// Write queues a record for async writing.
// Returns immediately; check the Done channel for completion.
func (w *AsyncWriter) Write(table string, record interface{}) chan error {
	job := WriteJob{
		Table:  table,
		Record: record,
		Done:   make(chan error, 1),
	}

	select {
	case w.queue <- job:
		// queued successfully
	default:
		// queue full - write synchronously as fallback
		log.Printf("[writer] queue full, writing synchronously to %s", table)
		go func() {
			err := w.db.Create(record).Error
			job.Done <- err
		}()
	}

	return job.Done
}

// WriteAsync queues a record without waiting for result.
// Use for fire-and-forget writes where errors can be logged.
func (w *AsyncWriter) WriteAsync(table string, record interface{}) {
	job := WriteJob{
		Table:  table,
		Record: record,
		Done:   nil, // no one will wait
	}

	select {
	case w.queue <- job:
	default:
		// queue full - drop the write but log it
		log.Printf("[writer] queue full, dropping write to %s", table)
	}
}

// worker processes write jobs with batching.
func (w *AsyncWriter) worker() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]WriteJob, 0, w.batchSize)

	for {
		select {
		case job, ok := <-w.queue:
			if !ok {
				// queue closed, flush remaining
				w.flush(batch)
				return
			}
			batch = append(batch, job)
			if len(batch) >= w.batchSize {
				w.flush(batch)
				batch = make([]WriteJob, 0, w.batchSize)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = make([]WriteJob, 0, w.batchSize)
			}

		case <-w.stopChan:
			w.flush(batch)
			return
		}
	}
}

// flush writes a batch of records to the database.
func (w *AsyncWriter) flush(jobs []WriteJob) {
	if len(jobs) == 0 {
		return
	}

	// Use a transaction for batch insert
	err := w.db.Transaction(func(tx *gorm.DB) error {
		for _, job := range jobs {
			if err := tx.Create(job.Record).Error; err != nil {
				return err
			}
		}
		return nil
	})

	// Notify callers of completion
	for _, job := range jobs {
		if job.Done != nil {
			job.Done <- err
		}
	}

	if err != nil {
		log.Printf("[writer] batch write failed: %v", err)
	}
}

// Stop gracefully shuts down the writer.
func (w *AsyncWriter) Stop() {
	close(w.stopChan)
	w.wg.Wait()
}

// Stats returns current queue statistics.
func (w *AsyncWriter) Stats() (queued int, capacity int) {
	return len(w.queue), cap(w.queue)
}

// Global async writer instance.
var Default *AsyncWriter

// Init initializes the global async writer.
func Init(db *gorm.DB) {
	Default = NewAsyncWriter(db, 100, 5000)
}

// Stop stops the global async writer.
func Stop() {
	if Default != nil {
		Default.Stop()
	}
}
