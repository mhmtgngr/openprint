/**
 * Job Assignment Load Test
 *
 * Tests the job assignment and agent polling endpoints
 * under concurrent load
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { JOB_URL } from '../../lib/config.js';

// Custom metrics
export const assignmentSuccessRate = new Rate('assignment_success_rate');
export const assignmentDuration = new Trend('assignment_duration');
export const pollSuccessRate = new Rate('poll_success_rate');
export const pollDuration = new Trend('poll_duration');
export const jobsAssigned = new Counter('jobs_assigned');
export const jobsPolled = new Counter('jobs_polled');
export const activeAgentsPolling = new Gauge('active_agents_polling');

// Agent states per VU
const agentStates = new Map();

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m', target: 200 },
    { duration: '1m', target: 200 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'assignment_duration': ['p(95)<400', 'p(99)<800'],
    'poll_duration': ['p(95)<300', 'p(99)<600'],
    'assignment_success_rate': ['rate>0.95'],
    'poll_success_rate': ['rate>0.98'],
    'http_req_failed': ['rate<0.02'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
};

/**
 * Get or create agent ID for this VU
 */
export function getAgentId(vuId) {
  if (!agentStates.has(vuId)) {
    agentStates.set(vuId, {
      agentId: `loadtest-agent-${vuId}`,
      jobsPolled: 0,
      startTime: Date.now(),
    });
  }
  return agentStates.get(vuId).agentId;
}

/**
 * Agent poll for available jobs
 */
export function agentPoll(agentId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'AgentPoll' },
  };

  return http.get(`${JOB_URL}/agents/${agentId}/poll`, params);
}

/**
 * Update job status (agent completing assignment)
 */
export function updateJobStatus(jobId, status, agentId) {
  const payload = JSON.stringify({
    status,
    agent_id: agentId,
    message: `Job ${status} by load test`,
    pages: status === 'completed' ? Math.floor(Math.random() * 50) + 1 : 0,
  });

  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'UpdateJobStatus' },
  };

  return http.put(`${JOB_URL}/jobs/${jobId}/status`, payload, params);
}

/**
 * Get queue statistics
 */
export function getQueueStats() {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'QueueStats' },
  };

  return http.get(`${JOB_URL}/jobs/queue/stats`, params);
}

/**
 * Main test scenario
 */
export default function () {
  const vuId = __VU;
  const agentId = getAgentId(vuId);
  const agentState = agentStates.get(vuId);

  group('Job Assignment', () => {
    const op = __ITER % 10;

    let response;
    let success;

    if (op < 7) {
      // 70% - Agent poll for jobs
      response = agentPoll(agentId);

      success = check(response, {
        'poll status is 200': (r) => r.status === 200,
        'has jobs array': (r) => {
          try {
            const body = r.json();
            return Array.isArray(body.jobs) || Array.isArray(body.data);
          } catch (e) {
            return false;
          }
        },
        'response time < 300ms': (r) => r.timings.duration < 300,
      });

      pollSuccessRate.add(success);
      pollDuration.add(response.timings.duration);

      if (success) {
        jobsPolled.add(1);
        agentState.jobsPolled++;

        // Process any assigned jobs
        try {
          const body = response.json();
          const jobs = body.jobs || body.data || [];

          if (jobs.length > 0) {
            jobsAssigned.add(jobs.length);

            // Simulate processing first job
            const job = jobs[0];
            const jobId = job.job_id || job.id;

            // Randomly complete or fail the job
            const finalStatus = Math.random() > 0.05 ? 'completed' : 'failed';
            const statusResponse = updateJobStatus(jobId, finalStatus, agentId);

            assignmentSuccessRate.add(statusResponse.status === 200);
            assignmentDuration.add(statusResponse.timings.duration);
          }
        } catch (e) {}
      }

    } else if (op < 9) {
      // 20% - Update job status directly
      const jobId = `00000000-0000-4000-8000-000000000${(__ITER % 100).toString().padStart(3, '0')}`;
      const status = Math.random() > 0.1 ? 'processing' : 'completed';

      response = updateJobStatus(jobId, status, agentId);

      success = check(response, {
        'update status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      });

      assignmentSuccessRate.add(success);
      assignmentDuration.add(response.timings.duration);

    } else {
      // 10% - Get queue stats
      response = getQueueStats();

      success = check(response, {
        'queue stats status is 200': (r) => r.status === 200,
        'has stats data': (r) => {
          try {
            const body = r.json();
            return body.queued !== undefined || body.total !== undefined;
          } catch (e) {
            return false;
          }
        },
      });
    }

    activeAgentsPolling.set(agentStates.size);

    // Simulate agent polling interval
    sleep(Math.random() * 2 + 1);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting job assignment load test');
  console.log(`Target endpoint: ${JOB_URL}`);
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('Job assignment test completed');
  console.log(`Jobs assigned: ${jobsAssigned.count}`);
  console.log(`Jobs polled: ${jobsPolled.count}`);
  console.log(`Assignment success rate: ${(assignmentSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`Poll success rate: ${(pollSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 poll duration: ${pollDuration.p('95').toFixed(0)}ms`);

  agentStates.clear();
}
