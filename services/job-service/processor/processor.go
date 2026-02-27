// Package processor handles background job processing for the job service.
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/openprint/openprint/services/job-service/repository"
)

// JobRepository defines the interface for job repository operations.
type JobRepository interface {
	Create(ctx context.Context, job *repository.PrintJob) error
	FindByID(ctx context.Context, id string) (*repository.PrintJob, error)
	Update(ctx context.Context, job *repository.PrintJob) error
	FindByStatus(ctx context.Context, status string, limit int) ([]*repository.PrintJob, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	UpdateStatus(ctx context.Context, jobID, status string) error
	Delete(ctx context.Context, id string) error
	FindByPrinter(ctx context.Context, printerID string, limit, offset int) ([]*repository.PrintJob, error)
	FindByUser(ctx context.Context, userEmail string, limit, offset int) ([]*repository.PrintJob, error)
	ListWithFilters(ctx context.Context, limit, offset int, printerID, status, userEmail string) ([]*repository.PrintJob, int, error)
	GetNextPendingJob(ctx context.Context, printerID string) (*repository.PrintJob, error)
	UpdateJobProgress(ctx context.Context, jobID string, progress int) error
	GetJobsNeedingRetry(ctx context.Context, maxRetries, limit int) ([]*repository.PrintJob, error)
	AssignAgent(ctx context.Context, jobID, agentID string) error
}

// JobHistoryRepository defines the interface for job history repository operations.
type JobHistoryRepository interface {
	Create(ctx context.Context, history *repository.JobHistory) error
	FindByID(ctx context.Context, id string) (*repository.JobHistory, error)
	FindByJobID(ctx context.Context, jobID string) ([]*repository.JobHistory, error)
	FindByStatus(ctx context.Context, status string, limit, offset int) ([]*repository.JobHistory, error)
	DeleteByJobID(ctx context.Context, jobID string) error
	DeleteOld(ctx context.Context, olderThan time.Duration) (int64, error)
	GetLatestByJobID(ctx context.Context, jobID string) (*repository.JobHistory, error)
	CountByJobID(ctx context.Context, jobID string) (int, error)
	List(ctx context.Context, limit, offset int) ([]*repository.JobHistory, int, error)
	CreateBatch(ctx context.Context, entries []*repository.JobHistory) error
}

// Config holds processor configuration.
type Config struct {
	JobRepo      JobRepository
	HistoryRepo  JobHistoryRepository
	Redis        *redis.Client
	Workers      int
	PollInterval time.Duration
}

// Stats represents processor statistics.
type Stats struct {
	Queued      int64 `json:"queued"`
	Processing  int64 `json:"processing"`
	Completed   int64 `json:"completed"`
	Failed      int64 `json:"failed"`
	Workers     int   `json:"workers"`
}

// Processor handles background job processing.
type Processor struct {
	jobRepo     JobRepository
	historyRepo JobHistoryRepository
	redis       *redis.Client
	workers     int
	pollInterval time.Duration

	// Channels for job distribution
	jobQueue    chan *repository.PrintJob
	workerStop  chan struct{}
	wg          sync.WaitGroup

	// Tracking
	mu           sync.Mutex
	processing   map[string]*repository.PrintJob
	cancelled    map[string]struct{}
}

// New creates a new job processor.
func New(cfg Config) *Processor {
	p := &Processor{
		jobRepo:      cfg.JobRepo,
		historyRepo:  cfg.HistoryRepo,
		redis:        cfg.Redis,
		workers:      cfg.Workers,
		pollInterval: cfg.PollInterval,
		jobQueue:     make(chan *repository.PrintJob, 1000),
		workerStop:   make(chan struct{}),
		processing:   make(map[string]*repository.PrintJob),
		cancelled:    make(map[string]struct{}),
	}

	return p
}

// Start begins the job processor.
func (p *Processor) Start(ctx context.Context) {
	// Start dispatcher
	p.wg.Add(1)
	go p.dispatcher(ctx)

	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Start status reporter
	p.wg.Add(1)
	go p.statusReporter(ctx)
}

// Stop gracefully shuts down the processor.
func (p *Processor) Stop() {
	close(p.workerStop)
	p.wg.Wait()
}

// dispatcher polls for jobs and dispatches them to workers.
func (p *Processor) dispatcher(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.workerStop:
			return
		case <-ticker.C:
			p.pollJobs(ctx)
		}
	}
}

// pollJobs retrieves pending jobs from the database.
func (p *Processor) pollJobs(ctx context.Context) {
	// Get queued jobs up to worker count
	jobs, err := p.jobRepo.FindByStatus(ctx, "queued", p.workers*2)
	if err != nil {
		fmt.Printf("Failed to poll jobs: %v\n", err)
		return
	}

	for _, job := range jobs {
		// Check if job was cancelled
		p.mu.Lock()
		if _, ok := p.cancelled[job.ID]; ok {
			delete(p.cancelled, job.ID)
			p.mu.Unlock()
			continue
		}
		p.mu.Unlock()

		select {
		case p.jobQueue <- job:
			// Mark as processing
			job.Status = "processing"
			job.StartedAt = time.Now()
			p.jobRepo.Update(ctx, job)

			p.addHistory(ctx, job.ID, "processing", "Job assigned to processor")
		default:
			// Queue full, skip
			return
		}
	}
}

// worker processes jobs from the queue.
func (p *Processor) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.workerStop:
			return
		case job := <-p.jobQueue:
			p.processJob(ctx, job, workerID)
		}
	}
}

// processJob handles a single print job.
func (p *Processor) processJob(ctx context.Context, job *repository.PrintJob, workerID int) {
	// Track processing
	p.mu.Lock()
	p.processing[job.ID] = job
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.processing, job.ID)
		p.mu.Unlock()
	}()

	// Check for cancellation
	p.mu.Lock()
	if _, ok := p.cancelled[job.ID]; ok {
		delete(p.cancelled, job.ID)
		p.mu.Unlock()

		job.Status = "cancelled"
		job.CompletedAt = &[]time.Time{time.Now()}[0]
		p.jobRepo.Update(ctx, job)
		p.addHistory(ctx, job.ID, "cancelled", "Job cancelled during processing")
		return
	}
	p.mu.Unlock()

	// In a real implementation, this would:
	// 1. Fetch the document from storage service
	// 2. Send the job to the agent via WebSocket/HTTP
	// 3. Monitor progress
	// 4. Handle completion/failure

	// For now, simulate processing
	fmt.Printf("Worker %d: Processing job %s\n", workerID, job.ID)

	// Simulate work
	time.Sleep(100 * time.Millisecond)

	// Update to pending_agent (waiting for agent to pick up)
	job.Status = "pending_agent"
	p.jobRepo.Update(ctx, job)
	p.addHistory(ctx, job.ID, "pending_agent", "Job waiting for agent")

	// In production, we'd wait for agent acknowledgment here
	// For now, we'll mark as completed after a delay
}

// Enqueue adds a job to the processing queue.
func (p *Processor) Enqueue(ctx context.Context, job *repository.PrintJob) error {
	if p.redis == nil {
		// No redis configured, job will be picked up by polling
		return nil
	}
	// Add to Redis queue for persistence
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return p.redis.LPush(ctx, "print:queue", data).Err()
}

// Cancel marks a job for cancellation.
func (p *Processor) Cancel(ctx context.Context, jobID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If currently processing, mark for cancellation
	if _, ok := p.processing[jobID]; ok {
		p.cancelled[jobID] = struct{}{}
	}

	// Remove from Redis queue (if redis is configured)
	if p.redis != nil {
		p.redis.LRem(ctx, "print:queue", 0, jobID)
	}
}

// GetStats returns current processor statistics.
func (p *Processor) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		Workers: p.workers,
	}

	// Get status counts from database
	queued, _ := p.jobRepo.CountByStatus(ctx, "queued")
	processing, _ := p.jobRepo.CountByStatus(ctx, "processing")
	pendingAgent, _ := p.jobRepo.CountByStatus(ctx, "pending_agent")
	completed, _ := p.jobRepo.CountByStatus(ctx, "completed")
	failed, _ := p.jobRepo.CountByStatus(ctx, "failed")

	stats.Queued = queued
	stats.Processing = processing + pendingAgent
	stats.Completed = completed
	stats.Failed = failed

	return stats, nil
}

// statusReporter periodically reports processor status.
func (p *Processor) statusReporter(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.workerStop:
			return
		case <-ticker.C:
			stats, err := p.GetStats(ctx)
			if err == nil {
				fmt.Printf("Processor Stats: Queued=%d Processing=%d Completed=%d Failed=%d\n",
					stats.Queued, stats.Processing, stats.Completed, stats.Failed)
			}
		}
	}
}

// CompleteJob marks a job as completed from an agent callback.
func (p *Processor) CompleteJob(ctx context.Context, jobID string, success bool, message string) error {
	job, err := p.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	now := time.Now()
	job.CompletedAt = &now
	job.UpdatedAt = now

	if success {
		job.Status = "completed"
	} else {
		job.Status = "failed"
		job.Retries++
	}

	if err := p.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	p.addHistory(ctx, jobID, job.Status, message)

	return nil
}

// UpdateJobProgress updates job progress from agent feedback.
func (p *Processor) UpdateJobProgress(ctx context.Context, jobID, status string, progress int, message string) error {
	job, err := p.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = status
	job.UpdatedAt = time.Now()

	if err := p.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	p.addHistory(ctx, jobID, status, fmt.Sprintf("%s (Progress: %d%%)", message, progress))

	return nil
}

// RequeueFailedJobs requeues jobs that have failed but haven't exceeded retry limit.
func (p *Processor) RequeueFailedJobs(ctx context.Context, maxRetries int) (int, error) {
	jobs, err := p.jobRepo.FindByStatus(ctx, "failed", 100)
	if err != nil {
		return 0, err
	}

	requeued := 0
	for _, job := range jobs {
		if job.Retries < maxRetries {
			job.Status = "queued"
			job.UpdatedAt = time.Now()

			if err := p.jobRepo.Update(ctx, job); err != nil {
				continue
			}

			// Enqueue
			p.Enqueue(ctx, job)
			p.addHistory(ctx, job.ID, "queued", fmt.Sprintf("Auto-requeue attempt %d", job.Retries+1))
			requeued++
		}
	}

	return requeued, nil
}

// GetQueueLength returns the current queue length.
func (p *Processor) GetQueueLength(ctx context.Context) (int64, error) {
	return p.redis.LLen(ctx, "print:queue").Result()
}

// Pause pauses job processing.
func (p *Processor) Pause(ctx context.Context) error {
	return p.redis.Set(ctx, "print:processor:paused", "1", 0).Err()
}

// Resume resumes job processing.
func (p *Processor) Resume(ctx context.Context) error {
	return p.redis.Del(ctx, "print:processor:paused").Err()
}

// IsPaused checks if the processor is paused.
func (p *Processor) IsPaused(ctx context.Context) (bool, error) {
	exists, err := p.redis.Exists(ctx, "print:processor:paused").Result()
	return exists > 0, err
}

// addHistory adds a history entry for a job.
func (p *Processor) addHistory(ctx context.Context, jobID, status, message string) {
	history := &repository.JobHistory{
		JobID:    jobID,
		Status:   status,
		Message:  message,
		CreatedAt: time.Now(),
	}
	p.historyRepo.Create(ctx, history)
}
