/**
 * Stress Test
 *
 * Tests system behavior beyond expected capacity
 * Identifies breaking points and failure modes
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { BASE_URL, REGISTRY_URL, JOB_URL } from '../../lib/config.js';

// Custom metrics
export const stressSuccessRate = new Rate('stress_success_rate');
export const stressResponseTime = new Trend('stress_response_time');
export const stressErrors = new Counter('stress_errors');
export const stressBreaks = new Counter('stress_breaks'); // System failures
export const stressCapacity = new Gauge('stress_capacity'); // Current load level

// Track breaking points
const breakingPoints = [];

// Test configuration - push to breaking point
export const options = {
  scenarios: {
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },   // Warm up
        { duration: '2m', target: 500 },   // Load
        { duration: '2m', target: 1000 },  // Heavy load
        { duration: '2m', target: 2000 },  // Stress
        { duration: '2m', target: 3000 },  // Breaking point
        { duration: '1m', target: 100 },   // Recovery
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    // Very lenient thresholds - we expect failures
    'stress_response_time': ['p(95)<5000'],
    'stress_success_rate': ['rate>0.5'], // Accept 50% failure rate at peak
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-stress-test/1.0',
};

/**
 * Perform lightweight request for max throughput
 */
export function lightRequest() {
  const agentId = `stress-agent-${__VU}-${__ITER}`;
  const payload = JSON.stringify({
    agent_id: agentId,
    status: 'online',
  });

  const startTime = Date.now();
  const response = http.post(`${REGISTRY_URL}/agents/${agentId}/heartbeat`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'StressHeartbeat' },
    timeout: '10s', // Shorter timeout to fail fast
  });
  const duration = Date.now() - startTime;

  const success = response.status === 200;
  const isBreak = response.status === 503 || response.status === 502 || response.status === 504;

  stressSuccessRate.add(success);
  stressResponseTime.add(duration);

  if (isBreak) {
    stressBreaks.add(1);
    breakingPoints.push({
      vus: __VU,
      time: Date.now(),
      status: response.status,
    });
  }

  if (!success && !isBreak) {
    stressErrors.add(1);
  }

  return { success, duration, isBreak };
}

/**
 * Perform heavier request
 */
export function heavyRequest() {
  const payload = JSON.stringify({
    email: `stress-user-${__VU}-${__ITER}@example.com`,
    password: 'TestPassword123!',
  });

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'StressLogin' },
    timeout: '15s',
  });
  const duration = Date.now() - startTime;

  const success = response.status === 200;
  const isBreak = response.status === 503 || response.status === 502 || response.status === 504;

  stressSuccessRate.add(success);
  stressResponseTime.add(duration);

  if (isBreak) {
    stressBreaks.add(1);
  }

  if (!success && !isBreak) {
    stressErrors.add(1);
  }

  return { success, duration, isBreak };
}

/**
 * Perform read-only request (job status)
 */
export function readRequest() {
  const jobId = `00000000-0000-4000-8000-00000000${(__ITER % 1000).toString().padStart(4, '0')}`;

  const startTime = Date.now();
  const response = http.get(`${JOB_URL}/jobs/${jobId}`, {
    headers: BASE_HEADERS,
    tags: { name: 'StressRead' },
    timeout: '10s',
  });
  const duration = Date.now() - startTime;

  const success = response.status === 200 || response.status === 404;
  const isBreak = response.status === 503 || response.status === 502 || response.status === 504;

  stressSuccessRate.add(success);
  stressResponseTime.add(duration);

  if (isBreak) {
    stressBreaks.add(1);
  }

  if (!success && !isBreak) {
    stressErrors.add(1);
  }

  return { success, duration, isBreak };
}

/**
 * Main stress test scenario
 */
export default function () {
  // Update capacity metric
  stressCapacity.add(__VU);

  group('Stress Test Operations', () => {
    // Mix of operations to stress different parts of system
    const op = __ITER % 10;

    let result;

    if (op < 5) {
      // 50% - Light requests (maximize throughput)
      result = lightRequest();
    } else if (op < 8) {
      // 30% - Read requests
      result = readRequest();
    } else {
      // 20% - Heavy requests
      result = heavyRequest();
    }

    // Log breaking points
    if (result.isBreak && breakingPoints.length < 10) {
      console.error(`Breaking point detected at ${__VU} VUs: ${result.duration}ms`);
    }

    // Minimal sleep - push hard
    sleep(0.05);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Stress Test');
  console.log('========================================');
  console.log('Purpose: Find system breaking point');
  console.log('Max VUs: 3000');
  console.log('Duration: ~11 minutes');
  console.log('Warning: This test will cause failures');
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
  console.log('Stress Test Results');
  console.log('========================================');
  console.log(`Duration: ${(duration / 60).toFixed(1)} minutes`);
  console.log(`Total errors: ${stressErrors.count}`);
  console.log(`System breaks: ${stressBreaks.count}`);
  console.log(`Break rate: ${((stressBreaks.count / __ITER) * 100).toFixed(2)}%`);
  console.log(`Success rate: ${(stressSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P50 response time: ${stressResponseTime.p('50').toFixed(0)}ms`);
  console.log(`P95 response time: ${stressResponseTime.p('95').toFixed(0)}ms`);
  console.log(`P99 response time: ${stressResponseTime.p('99').toFixed(0)}ms`);
  console.log(`Max response time: ${stressResponseTime.max.toFixed(0)}ms`);

  if (breakingPoints.length > 0) {
    console.log('\nBreaking points detected:');
    breakingPoints.slice(0, 5).forEach((bp, i) => {
      console.log(`  ${i + 1}. At ~${bp.vus} VUs: HTTP ${bp.status}`);
    });
  }

  // Capacity assessment
  const breakingPoint = stressBreaks.count > 100 ? 'DEGRADED' : 'STABLE';
  console.log(`\nSystem state at peak: ${breakingPoint}`);
  console.log('========================================');
}
