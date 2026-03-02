/**
 * Parallel Job Submission Test
 *
 * FR-002: Tests the job service's ability to handle
 * concurrent print job submissions without performance degradation
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { JOB_URL, TestData } from '../../lib/config.js';
import { createPrintJobData } from '../../lib/helpers.js';

// Custom metrics
export const jobSubmitSuccessRate = new Rate('job_submit_success_rate');
export const jobSubmitDuration = new Trend('job_submit_duration');
export const jobsSubmitted = new Counter('jobs_submitted');
export const jobsQueued = new Gauge('jobs_queued');

// Track submitted jobs for cleanup/validation
const submittedJobs = [];

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 100 },
    { duration: '2m', target: 500 },
    { duration: '1m', target: 500 },
    { duration: '30s', target: 10 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'job_submit_duration': ['p(95)<500', 'p(99)<1000'],
    'job_submit_success_rate': ['rate>0.95'],
    'http_req_duration': ['p(95)<800'],
    'http_req_failed': ['rate<0.05'],
  },
};

const SUBMIT_ENDPOINT = `${JOB_URL}/jobs`;

/**
 * Generate realistic print job data
 */
export function generateJobData() {
  return createPrintJobData();
}

/**
 * Submit a print job
 */
export function submitJob(jobData, authToken = null) {
  const payload = JSON.stringify(jobData);

  const headers = {
    'Content-Type': 'application/json',
  };

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }

  const params = {
    headers,
    tags: { name: 'SubmitJob' },
  };

  return http.post(SUBMIT_ENDPOINT, payload, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Job Submission', () => {
    const jobData = generateJobData();
    const response = submitJob(jobData);

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
      'has status': (r) => {
        try {
          const body = r.json();
          return body.status !== undefined;
        } catch (e) {
          return false;
        }
      },
      'status is queued': (r) => {
        try {
          const body = r.json();
          return body.status === 'queued';
        } catch (e) {
          return false;
        }
      },
      'response time < 1s': (r) => r.timings.duration < 1000,
    });

    jobSubmitSuccessRate.add(success);
    jobSubmitDuration.add(response.timings.duration);

    if (success) {
      jobsSubmitted.add(1);

      try {
        const body = response.json();
        const jobId = body.job_id || body.id;
        submittedJobs.push({
          id: jobId,
          submittedAt: Date.now(),
          data: jobData,
        });

        // Track queue size
        jobsQueued.set(submittedJobs.length);
      } catch (e) {}
    }

    // Cleanup old job references to prevent memory issues
    if (submittedJobs.length > 10000) {
      submittedJobs.splice(0, 5000);
    }

    // Simulate realistic user think time between submissions
    sleep(Math.random() * 2 + 1);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting parallel job submission test');
  console.log(`Target endpoint: ${SUBMIT_ENDPOINT}`);
  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('Job submission test completed');
  console.log(`Total jobs submitted: ${jobsSubmitted.count}`);
  console.log(`Submit rate: ${(jobsSubmitted.count / duration).toFixed(2)} jobs/sec`);
  console.log(`Success rate: ${(jobSubmitSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 duration: ${jobSubmitDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 duration: ${jobSubmitDuration.p('99').toFixed(0)}ms`);
}
