package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// RequestQueue manages priority request queuing when rate limits are exceeded.
type RequestQueue struct {
	redis *RedisClient
	mu    sync.RWMutex

	// In-memory queues for fast access
	queues map[string]*priorityQueue

	// Default configuration
	defaultMaxSize    int
	defaultMaxWait    time.Duration
	defaultProcessing int // requests per interval when dequeuing
}

// priorityQueue implements a priority queue for rate limit requests.
type priorityQueue struct {
	items      []*queuedRequest
	maxSize    int
	maxWait    time.Duration
	mu         sync.Mutex
	notifyCh   chan struct{}
	lastUpdate time.Time
}

// queuedRequest represents a queued request.
type queuedRequest struct {
	ID        string      `json:"id"`
	Request   *Request    `json:"request"`
	Priority  int         `json:"priority"`
	QueuedAt  time.Time   `json:"queued_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// NewRequestQueue creates a new request queue.
func NewRequestQueue(redis *RedisClient) *RequestQueue {
	rq := &RequestQueue{
		redis:             redis,
		queues:            make(map[string]*priorityQueue),
		defaultMaxSize:    100,
		defaultMaxWait:    5 * time.Minute,
		defaultProcessing: 10,
	}

	// Start processing goroutine
	go rq.processQueues()

	return rq
}

// Enqueue adds a request to the queue.
// Returns (queued, position, estimatedWait).
func (rq *RequestQueue) Enqueue(ctx context.Context, req *Request, policy *Policy) (bool, int, time.Duration) {
	if policy.MaxQueueSize <= 0 {
		return false, 0, 0
	}

	queueKey := rq.getQueueKey(req)
	queue := rq.getOrCreateQueue(queueKey, policy)

	queue.mu.Lock()
	defer queue.mu.Unlock()

	// Check if queue is full
	if len(queue.items) >= queue.maxSize {
		return false, 0, 0
	}

	// Create queued request
	queued := &queuedRequest{
		ID:        generateID(),
		Request:   req,
		Priority:  req.Priority,
		QueuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(queue.maxWait),
	}

	// Add to queue (insert in priority order)
	position := rq.insertByPriority(queue.items, queued)

	// Notify processor
	select {
	case queue.notifyCh <- struct{}{}:
	default:
	}

	// Calculate estimated wait time
	estimatedWait := rq.estimateWait(queue, position)

	return true, position + 1, estimatedWait
}

// insertByPriority inserts a request in priority order (higher priority first).
// Returns the position (0-indexed).
func (rq *RequestQueue) insertByPriority(items []*queuedRequest, item *queuedRequest) int {
	// Binary search for insertion point
	low, high := 0, len(items)

	for low < high {
		mid := (low + high) / 2
		if items[mid].Priority >= item.Priority {
			low = mid + 1
		} else {
			high = mid
		}
	}

	// Insert at found position
	items = append(items, nil)
	copy(items[low+1:], items[low:])
	items[low] = item

	return low
}

// getOrCreateQueue gets or creates a queue for a key.
func (rq *RequestQueue) getOrCreateQueue(key string, policy *Policy) *priorityQueue {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if queue, ok := rq.queues[key]; ok {
		return queue
	}

	maxSize := policy.MaxQueueSize
	if maxSize <= 0 {
		maxSize = rq.defaultMaxSize
	}

	queue := &priorityQueue{
		items:      make([]*queuedRequest, 0, maxSize),
		maxSize:    maxSize,
		maxWait:    rq.defaultMaxWait,
		notifyCh:   make(chan struct{}, 1),
		lastUpdate: time.Now(),
	}

	rq.queues[key] = queue
	return queue
}

// getQueueKey generates a queue key for a request.
func (rq *RequestQueue) getQueueKey(req *Request) string {
	return fmt.Sprintf("queue:%s:%s", req.Type, req.Identifier)
}

// Dequeue removes and returns the highest priority request from the queue.
func (rq *RequestQueue) Dequeue(ctx context.Context, req *Request) (*queuedRequest, bool) {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return nil, false
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if len(queue.items) == 0 {
		return nil, false
	}

	// Remove expired requests first
	rq.removeExpired(queue)

	if len(queue.items) == 0 {
		return nil, false
	}

	// Get highest priority item (first in sorted list)
	item := queue.items[0]
	queue.items = queue.items[1:]

	return item, true
}

// removeExpired removes expired requests from the queue.
func (rq *RequestQueue) removeExpired(queue *priorityQueue) {
	now := time.Now()
	i := 0

	for _, item := range queue.items {
		if item.ExpiresAt.After(now) {
			queue.items[i] = item
			i++
		}
	}

	// Truncate to new length
	for j := i; j < len(queue.items); j++ {
		queue.items[j] = nil
	}
	queue.items = queue.items[:i]
}

// estimateWait estimates the wait time for a position in the queue.
func (rq *RequestQueue) estimateWait(queue *priorityQueue, position int) time.Duration {
	if position == 0 {
		return 0
	}

	// Estimate based on average processing time
	// Assume ~100ms per request
	estimatedTime := time.Duration(position) * 100 * time.Millisecond

	// Cap at max wait
	if estimatedTime > queue.maxWait {
		estimatedTime = queue.maxWait
	}

	return estimatedTime
}

// processQueues processes queued requests periodically.
func (rq *RequestQueue) processQueues() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		rq.processAllQueues()
	}
}

// processAllQueues processes all queues, removing expired items.
func (rq *RequestQueue) processAllQueues() {
	rq.mu.RLock()
	keys := make([]string, 0, len(rq.queues))
	for key := range rq.queues {
		keys = append(keys, key)
	}
	rq.mu.RUnlock()

	now := time.Now()

	for _, key := range keys {
		rq.mu.RLock()
		queue, ok := rq.queues[key]
		rq.mu.RUnlock()

		if !ok {
			continue
		}

		queue.mu.Lock()
		rq.removeExpired(queue)

		// If queue is empty and unused for a while, remove it
		if len(queue.items) == 0 && now.Sub(queue.lastUpdate) > 5*time.Minute {
			rq.mu.Lock()
			delete(rq.queues, key)
			rq.mu.Unlock()
		}

		queue.lastUpdate = now
		queue.mu.Unlock()
	}
}

// GetQueueStatus returns the current status of a queue.
func (rq *RequestQueue) GetQueueStatus(req *Request) *QueueStatus {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return &QueueStatus{
			Key:       queueKey,
			QueueSize: 0,
		}
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	// Calculate statistics
	totalWait := time.Duration(0)
	maxWait := time.Duration(0)
	count := 0

	now := time.Now()
	for _, item := range queue.items {
		wait := now.Sub(item.QueuedAt)
		totalWait += wait
		if wait > maxWait {
			maxWait = wait
		}
		count++
	}

	avgWait := time.Duration(0)
	if count > 0 {
		avgWait = totalWait / time.Duration(count)
	}

	return &QueueStatus{
		Key:          queueKey,
		QueueSize:    count,
		MaxQueueSize: queue.maxSize,
		AvgWaitTime:  avgWait,
		MaxWaitTime:  maxWait,
	}
}

// QueueStatus represents the status of a request queue.
type QueueStatus struct {
	Key          string        `json:"key"`
	QueueSize    int           `json:"queue_size"`
	MaxQueueSize int           `json:"max_queue_size"`
	AvgWaitTime  time.Duration `json:"avg_wait_time"`
	MaxWaitTime  time.Duration `json:"max_wait_time"`
}

// GetPosition returns the current position of a request in the queue.
func (rq *RequestQueue) GetPosition(req *Request, requestID string) int {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return -1
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	for i, item := range queue.items {
		if item.ID == requestID {
			return i + 1
		}
	}

	return -1
}

// Remove removes a request from the queue.
func (rq *RequestQueue) Remove(req *Request, requestID string) bool {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return false
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	for i, item := range queue.items {
		if item.ID == requestID {
			// Remove by swapping with last and truncating
			queue.items[i] = queue.items[len(queue.items)-1]
			queue.items[len(queue.items)-1] = nil
			queue.items = queue.items[:len(queue.items)-1]
			return true
		}
	}

	return false
}

// Clear removes all items from a queue.
func (rq *RequestQueue) Clear(req *Request) int {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return 0
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	count := len(queue.items)
	queue.items = queue.items[:0]

	return count
}

// GetAllQueueStatus returns the status of all queues.
func (rq *RequestQueue) GetAllQueueStatus() map[string]*QueueStatus {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	statuses := make(map[string]*QueueStatus)

	for key, queue := range rq.queues {
		queue.mu.Lock()

		totalWait := time.Duration(0)
		maxWait := time.Duration(0)
		count := len(queue.items)

		now := time.Now()
		for _, item := range queue.items {
			wait := now.Sub(item.QueuedAt)
			totalWait += wait
			if wait > maxWait {
				maxWait = wait
			}
		}

		avgWait := time.Duration(0)
		if count > 0 {
			avgWait = totalWait / time.Duration(count)
		}

		statuses[key] = &QueueStatus{
			Key:          key,
			QueueSize:    count,
			MaxQueueSize: queue.maxSize,
			AvgWaitTime:  avgWait,
			MaxWaitTime:  maxWait,
		}

		queue.mu.Unlock()
	}

	return statuses
}

// SetMaxWait sets the maximum wait time for new queues.
func (rq *RequestQueue) SetMaxWait(maxWait time.Duration) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.defaultMaxWait = maxWait
}

// SetMaxSize sets the maximum queue size for new queues.
func (rq *RequestQueue) SetMaxSize(maxSize int) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.defaultMaxSize = maxSize
}

// ExportToRedis exports a queue to Redis for persistence.
func (rq *RequestQueue) ExportToRedis(ctx context.Context, req *Request) error {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return fmt.Errorf("queue not found")
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	data, err := json.Marshal(queue.items)
	if err != nil {
		return err
	}

	redisKey := fmt.Sprintf("ratelimit:queue:%s", queueKey)
	return rq.redis.SetJSON(ctx, redisKey, data, 10*time.Minute)
}

// ImportFromRedis imports a queue from Redis.
func (rq *RequestQueue) ImportFromRedis(ctx context.Context, req *Request) error {
	queueKey := rq.getQueueKey(req)
	redisKey := fmt.Sprintf("ratelimit:queue:%s", queueKey)

	var items []*queuedRequest
	if err := rq.redis.GetJSON(ctx, redisKey, &items); err != nil {
		return err
	}

	queue := rq.getOrCreateQueue(queueKey, &Policy{
		MaxQueueSize: rq.defaultMaxSize,
	})

	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.items = items
	return nil
}

// GetQueuedRequest retrieves a specific queued request by ID.
func (rq *RequestQueue) GetQueuedRequest(req *Request, requestID string) (*queuedRequest, bool) {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return nil, false
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	for _, item := range queue.items {
		if item.ID == requestID {
			return item, true
		}
	}

	return nil, false
}

// UpdatePriority updates the priority of a queued request.
func (rq *RequestQueue) UpdatePriority(req *Request, requestID string, newPriority int) bool {
	queueKey := rq.getQueueKey(req)

	rq.mu.RLock()
	queue, ok := rq.queues[queueKey]
	rq.mu.RUnlock()

	if !ok {
		return false
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	// Find and remove the item
	for i, item := range queue.items {
		if item.ID == requestID {
			// Remove from current position
			queue.items = append(queue.items[:i], queue.items[i+1:]...)
			// Update priority and re-insert
			item.Priority = newPriority
			rq.insertByPriority(queue.items, item)
			return true
		}
	}

	return false
}
