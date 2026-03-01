/**
 * Job Queue Processing Load Test
 *
 * Tests the job queue processing and state transitions
 * under heavy load
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { JOB_URL, TestData } from '../../lib/config.js';
import { createPrintJobData } from '../../lib/helpers.js';

// Custom metrics for queue processing
export const queueProcessingRate = new Rate('queue_processing_rate');
export const queueProcessingDuration = new Trend('queue_processing_duration');
export const jobsCreated = new Counter('jobs_created');
export const jobsProcessing = new Counter('jobs_processing');
export const jobsCompleted = new Counter('jobs_completed');
export const jobsFailed = new Counter('jobs_failed');
export const jobsCancelled = new Counter('jobs_cancelled');
export const queueDepth = new Gauge('queue_depth');

// Track job lifecycle
const jobLifecycles = new Map();

// Test configuration
export const options = {
  scenarios: {
    job_lifecycle: {
      executor: 'constant-vus',
      vus: 100,
      duration: '3m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'queue_processing_duration': ['p(95)<1000', 'p(99)<2000'],
    'queue_processing_rate': ['rate>0.90'],
    'http_req_failed': ['rate<0.05'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
};

/**
 * Create a new job
 */
export function createJob() {
  const jobData = createPrintJobData();
  const payload = JSON.stringify(jobData);

  const response = http.post(`${JOB_URL}/jobs`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'CreateJob' },
  });

  if (response.status === 201 || response.status === 200) {
    try {
      const body = response.json();
      const jobId = body.job_id || body.id;

      jobLifecycles.set(jobId, {
        id: jobId,
        status: 'queued',
        createdAt: Date.now(),
        transitions: ['queued'],
      });

      jobsCreated.add(1);
      return jobId;
    } catch (e) {}
  }

  return null;
}

/**
 * Get job status
 */
export function getJobStatus(jobId) {
  return http.get(`${JOB_URL}/jobs/${jobId}`, {
    headers: BASE_HEADERS,
    tags: { name: 'GetJobStatus' },
  });
}

/**
 * Update job to processing
 */
export function updateJobProcessing(jobId, agentId) {
  const payload = JSON.stringify({
    status: 'processing',
    agent_id: agentId,
    message: 'Job picked up by agent',
  });

  const response = http.put(`${JOB_URL}/jobs/${jobId}/status`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'UpdateProcessing' },
  });

  if (response.status === 200) {
    const lifecycle = jobLifecycles.get(jobId);
    if (lifecycle) {
      lifecycle.status = 'processing';
      lifecycle.transitions.push('processing');
      jobsProcessing.add(1);
    }
  }

  return response;
}

/**
 * Complete job
 */
export function completeJob(jobId, pages = 1) {
  const payload = JSON.stringify({
    status: 'completed',
    message: 'Job completed successfully',
    pages: pages,
  });

  const response = http.put(`${JOB_URL}/jobs/${jobId}/status`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'CompleteJob' },
  });

  if (response.status === 200) {
    const lifecycle = jobLifecycles.get(jobId);
    if (lifecycle) {
      lifecycle.status = 'completed';
      lifecycle.transitions.push('completed');
      lifecycle.completedAt = Date.now();
      jobsCompleted.add(1);
    }
  }

  return response;
}

/**
 * Fail job
 */
export function failJob(jobId) {
  const payload = JSON.stringify({
    status: 'failed',
    message: 'Job failed: test error',
  });

  const response = http.put(`${JOB_URL}/jobs/${jobId}/status`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'FailJob' },
  });

  if (response.status === 200) {
    const lifecycle = jobLifecycles.get(jobId);
    if (lifecycle) {
      lifecycle.status = 'failed';
      lifecycle.transitions.push('failed');
      jobsFailed.add(1);
    }
  }

  return response;
}

/**
 * Cancel job
 */
export function cancelJob(jobId) {
  const response = http.delete(`${JOB_URL}/jobs/${jobId}`, {
    headers: BASE_HEADERS,
    tags: { name: 'CancelJob' },
  });

  if (response.status === 200) {
    const lifecycle = jobLifecycles.get(jobId);
    if (lifecycle) {
      lifecycle.status = 'cancelled';
      lifecycle.transitions.push('cancelled');
      jobsCancelled.add(1);
    }
  }

  return response;
}

/**
 * Get queue stats
 */
export function getQueueStats() {
  return http.get(`${JOB_URL}/jobs/queue/stats`, {
    headers: BASE_HEADERS,
    tags: { name: 'QueueStats' },
  });
}

/**
 * Main test scenario - simulates complete job lifecycle
 */
export default function () {
  const agentId = `loadtest-agent-${__VU}`;
  const lifecycleKey = `${__VU}_${__ITER}`;

  group('Queue Processing', () => {
    const startTime = Date.now();
    let success = true;
    let finalStatus = 'unknown';

    // Step 1: Create job
    const jobId = createJob();

    if (!jobId) {
      console.error('Failed to create job');
      sleep(1);
      return;
    }

    // Step 2: Move to processing
    sleep(Math.random() * 0.5 + 0.1);
    const processingResponse = updateJobProcessing(jobId, agentId);

    if (processingResponse.status !== 200) {
      success = false;
    }

    // Step 3: Complete or fail the job
    sleep(Math.random() * 1 + 0.5);

    const outcome = Math.random();
    if (outcome > 0.95) {
      // 5% - Job fails
      const failResponse = failJob(jobId);
      finalStatus = 'failed';
      if (failResponse.status !== 200) success = false;
    } else if (outcome > 0.98) {
      // 2% - Job is cancelled
      const cancelResponse = cancelJob(jobId);
      finalStatus = 'cancelled';
      if (cancelResponse.status !== 200) success = false;
    } else {
      // 93% - Job completes successfully
      const pages = Math.floor(Math.random() * 50) + 1;
      const completeResponse = completeJob(jobId, pages);
      finalStatus = 'completed';
      if (completeResponse.status !== 200) success = false;
    }

    const duration = Date.now() - startTime;

    queueProcessingRate.add(success);
    queueProcessingDuration.add(duration);

    // Update queue depth metric
    if (__ITER % 10 === 0) {
      const statsResponse = getQueueStats();
      if (statsResponse.status === 200) {
        try {
          const body = statsResponse.json();
          const depth = body.queued || body.pending || 0;
          queueDepth.set(depth);
        } catch (e) {}
      }
    }

    // Cleanup old lifecycles
    if (jobLifecycles.size > 1000) {
      const entries = Array.from(jobLifecycles.entries());
      entries.slice(0, 500).forEach(([key]) => jobLifecycles.delete(key));
    }

    // Simulate processing time
    sleep(Math.random() * 2 + 1);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting job queue processing test');
  console.log(`Target endpoint: ${JOB_URL}`);
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('========================================');
  console.log('Queue Processing Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Jobs created: ${jobsCreated.count}`);
  console.log(`Jobs processed: ${jobsProcessing.count}`);
  console.log(`Jobs completed: ${jobsCompleted.count}`);
  console.log(`Jobs failed: ${jobsFailed.count}`);
  console.log(`Jobs cancelled: ${jobsCancelled.count}`);
  console.log(`Processing success rate: ${(queueProcessingRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 processing duration: ${queueProcessingDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 processing duration: ${queueProcessingDuration.p('99').toFixed(0)}ms`);
  console.log(`Throughput: ${(jobsCreated.count / duration).toFixed(2)} jobs/sec`);
  console.log('========================================');

  jobLifecycles.clear();
}
