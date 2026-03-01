/**
 * High-Frequency Heartbeat Simulation
 *
 * FR-003: Simulates high-frequency agent heartbeat traffic
 * Tests the registry service's ability to handle thousands of
 * concurrent heartbeat requests without performance degradation
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Gauge, Counter } from 'k6/metrics';
import { REGISTRY_URL, TestData } from '../../lib/config.js';

// Custom metrics
export const heartbeatSuccessRate = new Rate('heartbeat_success_rate');
export const heartbeatDuration = new Trend('heartbeat_duration');
export const heartbeatRate = new Gauge('heartbeat_rate_per_second');
export const activeAgents = new Gauge('active_agents');
export const heartbeatPongs = new Counter('heartbeat_pongs');

// Agent state per VU
const agentStates = new Map();

// Test configuration - high frequency heartbeat simulation
export const options = {
  scenarios: {
    heartbeat: {
      executor: 'constant-vus',
      vus: 1000,
      duration: '5m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'heartbeat_duration': ['p(95)<100', 'p(99)<200'],
    'heartbeat_success_rate': ['rate>0.99'],
    'http_req_duration': ['p(95)<150'],
    'http_req_failed': ['rate<0.01'],
  },
};

const HEARTBEAT_ENDPOINT_TEMPLATE = (id) => `${REGISTRY_URL}/agents/${id}/heartbeat`;

/**
 * Generate or get agent ID for this VU
 */
export function getAgentId(vuId) {
  if (!agentStates.has(vuId)) {
    agentStates.set(vuId, {
      agentId: `loadtest-agent-${vuId}-${Math.floor(Math.random() * 10000)}`,
      status: 'online',
      completedJobs: 0,
      failedJobs: 0,
      startTime: Date.now(),
    });
  }
  return agentStates.get(vuId).agentId;
}

/**
 * Generate heartbeat data
 */
export function generateHeartbeatData(agentState) {
  const statuses = ['online', 'processing', 'idle'];
  const status = agentState.status || 'online';

  return {
    agent_id: agentState.agentId,
    status: status,
    printer_count: Math.floor(Math.random() * 5),
    completed_jobs: agentState.completedJobs || 0,
    failed_jobs: agentState.failedJobs || 0,
    version: '2.0.0',
    capabilities: ['print', 'scan', 'copy'],
  };
}

/**
 * Send heartbeat
 */
export function sendHeartbeat(agentId, data) {
  const payload = JSON.stringify(data);

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'Heartbeat' },
  };

  return http.post(HEARTBEAT_ENDPOINT_TEMPLATE(agentId), payload, params);
}

/**
 * Batch heartbeat for multiple agents
 */
export function batchHeartbeat(heartbeats) {
  const payload = JSON.stringify({ heartbeats });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'BatchHeartbeat' },
  };

  return http.post(`${REGISTRY_URL}/agents/heartbeat/batch`, payload, params);
}

/**
 * Main test scenario
 */
export default function () {
  const vuId = __VU;
  const agentState = agentStates.get(vuId) || {
    agentId: `loadtest-agent-${vuId}-${__ITER}`,
    status: 'online',
    completedJobs: Math.floor(Math.random() * 100),
    failedJobs: Math.floor(Math.random() * 5),
  };

  agentStates.set(vuId, agentState);

  group('Agent Heartbeat', () => {
    const heartbeatData = generateHeartbeatData(agentState);
    const response = sendHeartbeat(agentState.agentId, heartbeatData);

    const success = check(response, {
      'heartbeat status is 200': (r) => r.status === 200,
      'has server_time': (r) => {
        try {
          const body = r.json();
          return body.server_time !== undefined;
        } catch (e) {
          return false;
        }
      },
      'has pending_jobs': (r) => {
        try {
          const body = r.json();
          return body.pending_jobs !== undefined;
        } catch (e) {
          return false;
        }
      },
      'response time < 100ms': (r) => r.timings.duration < 100,
      'response time < 200ms (p99)': (r) => r.timings.duration < 200,
    });

    heartbeatSuccessRate.add(success);
    heartbeatDuration.add(response.timings.duration);

    if (success) {
      heartbeatPongs.add(1);

      // Update agent state based on response
      try {
        const body = response.json();
        if (body.pending_jobs > 0) {
          agentState.status = 'processing';
        } else {
          agentState.status = 'idle';
        }

        // Simulate some job completions
        if (Math.random() > 0.9) {
          agentState.completedJobs += Math.floor(Math.random() * 3);
        }
        if (Math.random() > 0.98) {
          agentState.failedJobs += 1;
        }
      } catch (e) {
        // Response body might be empty for 204 No Content
      }
    }

    // Track active agents
    activeAgents.set(agentStates.size);

    // Calculate heartbeat rate
    if (__ITER % 10 === 0) {
      const elapsed = (Date.now() - agentState.startTime) / 1000;
      const rate = __ITER / elapsed;
      heartbeatRate.set(rate);
    }
  });

  // High-frequency heartbeat - short sleep between requests
  sleep(Math.random() * 0.5 + 0.5);
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('High-Frequency Heartbeat Simulation');
  console.log('========================================');
  console.log(`Target: ${REGISTRY_URL}`);
  console.log(`Concurrent VUs: 1000`);
  console.log(`Duration: 5 minutes`);
  console.log('Heartbeat interval: ~1 second per agent');
  console.log('========================================');

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const endTime = Date.now();
  const duration = (endTime - data.startTime) / 1000;

  console.log('========================================');
  console.log('Heartbeat Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Active agents: ${agentStates.size}`);
  console.log(`Success rate: ${(heartbeatSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P50 duration: ${heartbeatDuration.p('50').toFixed(0)}ms`);
  console.log(`P95 duration: ${heartbeatDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 duration: ${heartbeatDuration.p('99').toFixed(0)}ms`);
  console.log(`Total heartbeats: ${heartbeatPongs.count}`);
  console.log(`Avg heartbeat rate: ${(heartbeatPongs.count / duration).toFixed(2)}/s`);
  console.log('========================================');

  agentStates.clear();
}
