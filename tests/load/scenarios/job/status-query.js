/**
 * Job Status Query Load Test
 *
 * Tests concurrent job status queries
 * Simulates users checking print job progress
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Gauge } from 'k6/metrics';
import { JOB_URL } from '../../lib/config.js';

// Custom metrics
export const statusQuerySuccessRate = new Rate('status_query_success_rate');
export const statusQueryDuration = new Trend('status_query_duration');
export const activeJobQueries = new Gauge('active_job_queries');

// Test configuration
export const options = {
  stages: [
    { duration: '20s', target: 50 },
    { duration: '1m', target: 200 },
    { duration: '1m', target: 200 },
    { duration: '20s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'status_query_duration': ['p(95)<200', 'p(99)<400'],
    'status_query_success_rate': ['rate>0.98'],
    'http_req_duration': ['p(95)<300'],
    'http_req_failed': ['rate<0.02'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
};

// Predefined job IDs for testing (in real scenario, these would come from setup)
const testJobIds = [];
for (let i = 0; i < 1000; i++) {
  testJobIds.push(`00000000-0000-4000-8000-000000000${i.toString().padStart(3, '0')}`);
}

/**
 * Query job status
 */
export function queryJobStatus(jobId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'QueryJobStatus' },
  };

  return http.get(`${JOB_URL}/jobs/${jobId}`, params);
}

/**
 * List jobs with filters
 */
export function listJobs(status = null, limit = 50) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'ListJobs' },
  };

  let url = `${JOB_URL}/jobs?limit=${limit}`;
  if (status) {
    url += `&status=${status}`;
  }

  return http.get(url, params);
}

/**
 * Query job history
 */
export function queryJobHistory(jobId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'JobHistory' },
  };

  return http.get(`${JOB_URL}/jobs/history?job_id=${jobId}`, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Job Status Query', () => {
    // Mix of query operations
    const op = __ITER % 10;

    let response;
    let success;

    if (op < 6) {
      // 60% - Query specific job status
      const jobId = testJobIds[__ITER % testJobIds.length];
      response = queryJobStatus(jobId);

      success = check(response, {
        'status query returns 200 or 404': (r) => r.status === 200 || r.status === 404,
        'has job data when found': (r) => {
          if (r.status !== 200) return true;
          try {
            const body = r.json();
            return body.job_id !== undefined || body.id !== undefined;
          } catch (e) {
            return false;
          }
        },
        'response time < 200ms': (r) => r.timings.duration < 200,
      });

    } else if (op < 9) {
      // 30% - List jobs with various statuses
      const statuses = ['queued', 'processing', 'completed', 'failed'];
      const status = statuses[__ITER % statuses.length];
      response = listJobs(status, 50);

      success = check(response, {
        'list jobs status is 200': (r) => r.status === 200,
        'has jobs array': (r) => {
          try {
            const body = r.json();
            return Array.isArray(body.data) || Array.isArray(body.jobs);
          } catch (e) {
            return false;
          }
        },
      });

    } else {
      // 10% - Query job history
      const jobId = testJobIds[__ITER % testJobIds.length];
      response = queryJobHistory(jobId);

      success = check(response, {
        'history status is 200 or 404': (r) => r.status === 200 || r.status === 404,
        'has history when found': (r) => {
          if (r.status !== 200) return true;
          try {
            const body = r.json();
            return Array.isArray(body.history) || Array.isArray(body.data);
          } catch (e) {
            return false;
          }
        },
      });
    }

    statusQuerySuccessRate.add(success);
    statusQueryDuration.add(response.timings.duration);
    activeJobQueries.add(__VU);

    // Simulate user polling behavior
    sleep(Math.random() * 1 + 0.5);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting job status query load test');
  console.log(`Target endpoint: ${JOB_URL}`);
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('Job status query test completed');
  console.log(`Success rate: ${(statusQuerySuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 duration: ${statusQueryDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 duration: ${statusQueryDuration.p('99').toFixed(0)}ms`);
}
