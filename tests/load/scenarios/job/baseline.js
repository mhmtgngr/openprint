/**
 * Job Service Performance Baseline
 *
 * Establishes performance baseline for job service
 * Covers job submission, status queries, assignment, and queue operations
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { JOB_URL, TestData } from '../../lib/config.js';
import { createPrintJobData } from '../../lib/helpers.js';

// Custom metrics for baseline
export const baselineMetrics = {
  submit: new Trend('baseline_job_submit'),
  status: new Trend('baseline_job_status'),
  assignment: new Trend('baseline_job_assignment'),
  queue: new Trend('baseline_job_queue'),
  all: new Trend('baseline_job_all'),
};

export const operationRates = {
  submit: new Rate('baseline_job_submit_rate'),
  status: new Rate('baseline_job_status_rate'),
  assignment: new Rate('baseline_job_assignment_rate'),
  queue: new Rate('baseline_job_queue_rate'),
};

// Test configuration
export const options = {
  scenarios: {
    job_operations: {
      executor: 'constant-arrival-rate',
      rate: 50,
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 150,
    },
  },
  thresholds: {
    'baseline_job_submit': ['p(95)<500', 'p(99)<1000'],
    'baseline_job_status': ['p(95)<200', 'p(99)<400'],
    'baseline_job_assignment': ['p(95)<400', 'p(99)<800'],
    'baseline_job_queue': ['p(95)<300', 'p(99)<600'],
    'baseline_job_all': ['p(95)<400', 'p(99)<800'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-baseline-test/1.0',
};

// Track job IDs for status queries
const jobIds = [];
for (let i = 0; i < 100; i++) {
  jobIds.push(`00000000-0000-4000-8000-000000000${i.toString().padStart(3, '0')}`);
}

/**
 * Perform job submission
 */
function performSubmit() {
  const jobData = createPrintJobData();
  const payload = JSON.stringify(jobData);

  const startTime = Date.now();
  const response = http.post(`${JOB_URL}/jobs`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'SubmitJob' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'submit status is 201 or 200': (r) => r.status === 201 || r.status === 200,
    'has job_id': (r) => {
      try {
        const body = r.json();
        return body.job_id !== undefined || body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  baselineMetrics.submit.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.submit.add(success);

  return { success, duration };
}

/**
 * Perform job status query
 */
function performStatus() {
  const jobId = jobIds[__ITER % jobIds.length];

  const startTime = Date.now();
  const response = http.get(`${JOB_URL}/jobs/${jobId}`, {
    headers: BASE_HEADERS,
    tags: { name: 'GetJobStatus' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'status query is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  baselineMetrics.status.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.status.add(success);

  return { success, duration };
}

/**
 * Perform job assignment (agent poll)
 */
function performAssignment() {
  const agentId = `baseline-agent-${__VU}`;

  const startTime = Date.now();
  const response = http.get(`${JOB_URL}/agents/${agentId}/poll`, {
    headers: BASE_HEADERS,
    tags: { name: 'AgentPoll' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'poll status is 200': (r) => r.status === 200,
    'has jobs array': (r) => {
      try {
        const body = r.json();
        return Array.isArray(body.jobs) || Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    },
  });

  baselineMetrics.assignment.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.assignment.add(success);

  return { success, duration };
}

/**
 * Perform queue stats query
 */
function performQueueStats() {
  const startTime = Date.now();
  const response = http.get(`${JOB_URL}/jobs/queue/stats`, {
    headers: BASE_HEADERS,
    tags: { name: 'QueueStats' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'queue stats status is 200': (r) => r.status === 200,
  });

  baselineMetrics.queue.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.queue.add(success);

  return { success, duration };
}

/**
 * Main test scenario
 */
export default function () {
  group('Job Baseline', () => {
    // Weighted operation distribution
    // 40% submit, 30% status, 20% assignment, 10% queue stats
    const op = __ITER % 20;

    if (op < 8) {
      performSubmit();
    } else if (op < 14) {
      performStatus();
    } else if (op < 18) {
      performAssignment();
    } else {
      performQueueStats();
    }
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Job Service Baseline Test');
  console.log('========================================');
  console.log(`Target: ${JOB_URL}`);
  console.log('Duration: 2 minutes');
  console.log('Rate: 50 ops/sec');
  console.log('========================================');

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown - export baseline results
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('========================================');
  console.log('Job Service Baseline Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);

  const results = {
    timestamp: new Date().toISOString(),
    duration,
    operations: {
      submit: {
        p50: baselineMetrics.submit.p('50'),
        p95: baselineMetrics.submit.p('95'),
        p99: baselineMetrics.submit.p('99'),
        avg: baselineMetrics.submit.avg,
        min: baselineMetrics.submit.min,
        max: baselineMetrics.submit.max,
        success_rate: operationRates.submit.rate,
      },
      status: {
        p50: baselineMetrics.status.p('50'),
        p95: baselineMetrics.status.p('95'),
        p99: baselineMetrics.status.p('99'),
        avg: baselineMetrics.status.avg,
        min: baselineMetrics.status.min,
        max: baselineMetrics.status.max,
        success_rate: operationRates.status.rate,
      },
      assignment: {
        p50: baselineMetrics.assignment.p('50'),
        p95: baselineMetrics.assignment.p('95'),
        p99: baselineMetrics.assignment.p('99'),
        avg: baselineMetrics.assignment.avg,
        min: baselineMetrics.assignment.min,
        max: baselineMetrics.assignment.max,
        success_rate: operationRates.assignment.rate,
      },
      queue: {
        p50: baselineMetrics.queue.p('50'),
        p95: baselineMetrics.queue.p('95'),
        p99: baselineMetrics.queue.p('99'),
        avg: baselineMetrics.queue.avg,
        min: baselineMetrics.queue.min,
        max: baselineMetrics.queue.max,
        success_rate: operationRates.queue.rate,
      },
      all: {
        p95: baselineMetrics.all.p('95'),
        p99: baselineMetrics.all.p('99'),
        avg: baselineMetrics.all.avg,
        success_rate: (
          operationRates.submit.rate +
          operationRates.status.rate +
          operationRates.assignment.rate +
          operationRates.queue.rate
        ) / 4,
      },
    },
  };

  console.log(JSON.stringify(results, null, 2));
  console.log('========================================');
}
