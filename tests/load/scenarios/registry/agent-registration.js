/**
 * Agent Registration Load Test
 *
 * Tests the agent registration endpoint's ability to handle
 * multiple concurrent agent registrations
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { REGISTRY_URL, TestData } from '../../lib/config.js';

// Custom metrics
export const registrationSuccessRate = new Rate('registration_success_rate');
export const registrationDuration = new Trend('registration_duration');
export const agentsRegistered = new Counter('agents_registered');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '1m', target: 100 },
    { duration: '30s', target: 10 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'registration_duration': ['p(95)<500', 'p(99)<1000'],
    'registration_success_rate': ['rate>0.95'],
    'http_req_failed': ['rate<0.05'],
  },
};

const REGISTER_ENDPOINT = `${REGISTRY_URL}/agents/register`;

/**
 * Generate agent registration data
 */
export function generateAgentData() {
  return {
    name: `loadtest-agent-${TestData.agentId()}`,
    type: Math.random() > 0.5 ? 'desktop' : 'mobile',
    version: '2.0.0',
    os: randomChoice(['Windows 11', 'macOS 14', 'Ubuntu 22.04', 'iOS 17', 'Android 14']),
    hostname: `host-${Math.floor(Math.random() * 1000)}`,
  };
}

function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

/**
 * Register an agent
 */
export function registerAgent(agentData) {
  const payload = JSON.stringify(agentData);

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'RegisterAgent' },
  };

  return http.post(REGISTER_ENDPOINT, payload, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Agent Registration', () => {
    const agentData = generateAgentData();
    const response = registerAgent(agentData);

    const success = check(response, {
      'registration status is 201 or 200': (r) => r.status === 201 || r.status === 200,
      'has agent_id': (r) => {
        try {
          const body = r.json();
          return body.agent_id !== undefined || body.id !== undefined;
        } catch (e) {
          return false;
        }
      },
      'has enrollment_token': (r) => {
        try {
          const body = r.json();
          return body.enrollment_token !== undefined;
        } catch (e) {
          return false;
        }
      },
      'response time < 1s': (r) => r.timings.duration < 1000,
    });

    registrationSuccessRate.add(success);
    registrationDuration.add(response.timings.duration);

    if (success) {
      agentsRegistered.add(1);
    }

    sleep(Math.random() * 2 + 1);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting agent registration load test');
  console.log(`Target endpoint: ${REGISTER_ENDPOINT}`);
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  console.log('Agent registration test completed');
  console.log(`Agents registered: ${agentsRegistered.count}`);
  console.log(`Success rate: ${(registrationSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 duration: ${registrationDuration.p('95').toFixed(0)}ms`);
}
