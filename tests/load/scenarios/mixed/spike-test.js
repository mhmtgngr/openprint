/**
 * Spike Test
 *
 * Tests system behavior under sudden traffic spikes
 * Helps identify breaking points and recovery characteristics
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Gauge } from 'k6/metrics';
import { BASE_URL, REGISTRY_URL, JOB_URL } from '../../lib/config.js';

// Custom metrics
export const spikeSuccessRate = new Rate('spike_success_rate');
export const spikeResponseTime = new Trend('spike_response_time');
export const spikeSystemLoad = new Gauge('spike_system_load');
export const spikeRecoveryTime = new Trend('spike_recovery_time');

// Test configuration - dramatic spike pattern
export const options = {
  stages: [
    { duration: '1m', target: 10 },    // Baseline
    { duration: '30s', target: 1000 },  // SPIKE: Rapid ramp to 1000
    { duration: '2m', target: 1000 },   // Sustained spike
    { duration: '1m', target: 10 },     // Recovery
    { duration: '1m', target: 10 },     // Stabilized baseline
  ],
  thresholds: {
    'spike_response_time': ['p(95)<2000', 'p(99)<5000'], // More lenient during spike
    'spike_success_rate': ['rate>0.85'], // Allow some failures during spike
    'http_req_failed': ['rate<0.2'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-spike-test/1.0',
};

// Track when spike starts
let spikeStart = 0;

/**
 * Perform a simple auth request (login)
 */
export function authRequest() {
  const payload = JSON.stringify({
    email: `spike-test-${__VU}-${__ITER}@example.com`,
    password: 'TestPassword123!',
  });

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'SpikeAuth' },
  });
  const duration = Date.now() - startTime;

  const success = response.status === 200 || response.status === 201;

  spikeSuccessRate.add(success);
  spikeResponseTime.add(duration);

  return { success, duration };
}

/**
 * Perform a registry request (heartbeat)
 */
export function registryRequest() {
  const agentId = `spike-agent-${__VU}`;
  const payload = JSON.stringify({
    agent_id: agentId,
    status: 'online',
    printer_count: 1,
  });

  const startTime = Date.now();
  const response = http.post(`${REGISTRY_URL}/agents/${agentId}/heartbeat`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'SpikeRegistry' },
  });
  const duration = Date.now() - startTime;

  const success = response.status === 200;

  spikeSuccessRate.add(success);
  spikeResponseTime.add(duration);

  return { success, duration };
}

/**
 * Perform a job request (status query)
 */
export function jobRequest() {
  const jobId = `00000000-0000-4000-8000-00000000${(__ITER % 100).toString().padStart(3, '0')}`;

  const startTime = Date.now();
  const response = http.get(`${JOB_URL}/jobs/${jobId}`, {
    headers: BASE_HEADERS,
    tags: { name: 'SpikeJob' },
  });
  const duration = Date.now() - startTime;

  const success = response.status === 200 || response.status === 404;

  spikeSuccessRate.add(success);
  spikeResponseTime.add(duration);

  return { success, duration };
}

/**
 * Main spike test scenario
 */
export default function () {
  // Track spike start
  if (__VU === 500 && __ITER === 0 && spikeStart === 0) {
    spikeStart = Date.now();
  }

  group('Spike Test Operations', () => {
    // Distribute load across services
    const op = __ITER % 10;

    let result;

    if (op < 4) {
      // 40% - Auth requests (most expensive)
      result = authRequest();
    } else if (op < 7) {
      // 30% - Registry requests (high frequency)
      result = registryRequest();
    } else {
      // 30% - Job requests
      result = jobRequest();
    }

    // Track system load
    if (__ITER % 100 === 0) {
      spikeSystemLoad.add(__VU);
    }

    // Minimal sleep during spike to maximize load
    sleep(0.1);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Spike Test');
  console.log('========================================');
  console.log('Pattern: Baseline -> SPIKE -> Sustain -> Recovery -> Baseline');
  console.log('Peak load: 1000 concurrent VUs');
  console.log('Duration: ~6 minutes');
  console.log('========================================');

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('========================================');
  console.log('Spike Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Success rate: ${(spikeSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P50 response time: ${spikeResponseTime.p('50').toFixed(0)}ms`);
  console.log(`P95 response time: ${spikeResponseTime.p('95').toFixed(0)}ms`);
  console.log(`P99 response time: ${spikeResponseTime.p('99').toFixed(0)}ms`);
  console.log(`Max response time: ${spikeResponseTime.max.toFixed(0)}ms`);
  console.log('========================================');
}
