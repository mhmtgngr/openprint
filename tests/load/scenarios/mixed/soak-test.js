/**
 * Soak Test (Endurance Test)
 *
 * Tests system stability over extended periods
 * Identifies memory leaks, connection pool exhaustion, and other resource issues
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { BASE_URL, REGISTRY_URL, JOB_URL, STORAGE_URL, TestData } from '../../lib/config.js';
import { createPrintJobData, generateFileData } from '../../lib/helpers.js';

// Custom metrics
export const soakSuccessRate = new Rate('soak_success_rate');
export const soakResponseTime = new Trend('soak_response_time');
export const soakErrors = new Counter('soak_errors');
export const soakIterations = new Counter('soak_iterations');
export const memoryUsage = new Gauge('memory_usage_estimate');
export const activeConnections = new Gauge('active_connections');

// Track long-running state
const vuState = new Map();

// Test configuration - long duration
export const options = {
  scenarios: {
    soak: {
      executor: 'constant-vus',
      vus: 100,
      duration: '30m', // 30 minute soak test
      gracefulStop: '1m',
    },
  },
  thresholds: {
    'soak_response_time': ['p(95)<1000'], // Should maintain performance
    'soak_success_rate': ['rate>0.95'],
    'http_req_failed': ['rate<0.05'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-soak-test/1.0',
};

/**
 * Get or initialize VU state
 */
export function getVuState() {
  if (!vuState.has(__VU)) {
    vuState.set(__VU, {
      lastLogin: 0,
      accessToken: null,
      refreshToken: null,
      documentsUploaded: 0,
      jobsSubmitted: 0,
      errors: 0,
      startTime: Date.now(),
    });
  }
  return vuState.get(__VU);
}

/**
 * Login and refresh token management
 */
export function ensureAuthenticated(state) {
  const now = Date.now();
  const tokenAge = now - state.lastLogin;

  // Refresh if token is older than 10 minutes
  if (!state.accessToken || tokenAge > 10 * 60 * 1000) {
    const payload = JSON.stringify({
      email: `soak-user-${__VU}@example.com`,
      password: 'TestPassword123!',
    });

    const response = http.post(`${BASE_URL}/auth/login`, payload, {
      headers: BASE_HEADERS,
      tags: { name: 'SoakLogin' },
    });

    if (response.status === 200) {
      try {
        const body = response.json();
        state.accessToken = body.access_token;
        state.refreshToken = body.refresh_token;
        state.lastLogin = now;
      } catch (e) {
        state.errors++;
        soakErrors.add(1);
      }
    }
  }

  return state.accessToken;
}

/**
 * Perform document upload
 */
export function uploadDocument(accessToken) {
  const fileName = `soak-doc-${__VU}-${Date.now()}.pdf`;
  const fileData = generateFileData(50); // 50KB files

  const boundary = `----Boundary${Math.random().toString(16).substring(2)}`;
  let body = '';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`;
  body += `Content-Type: application/pdf\r\n\r\n`;
  body += fileData;
  body += '\r\n';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="user_email"\r\n\r\n`;
  body += `soak-user-${__VU}@example.com`;
  body += '\r\n';
  body += `--${boundary}--\r\n`;

  const headers = {
    ...BASE_HEADERS,
    'Content-Type': `multipart/form-data; boundary=${boundary}`,
    'Authorization': `Bearer ${accessToken}`,
  };

  const startTime = Date.now();
  const response = http.post(`${STORAGE_URL}/documents`, body, {
    headers,
    tags: { name: 'SoakUpload' },
    timeout: '60s',
  });
  const duration = Date.now() - startTime;

  return { response, duration, success: response.status === 201 || response.status === 200 };
}

/**
 * Submit print job
 */
export function submitJob(accessToken) {
  const jobData = createPrintJobData();

  const headers = {
    ...BASE_HEADERS,
    'Authorization': `Bearer ${accessToken}`,
  };

  const startTime = Date.now();
  const response = http.post(`${JOB_URL}/jobs`, JSON.stringify(jobData), {
    headers,
    tags: { name: 'SoakSubmitJob' },
  });
  const duration = Date.now() - startTime;

  return { response, duration, success: response.status === 201 || response.status === 200 };
}

/**
 * Send heartbeat
 */
export function sendHeartbeat() {
  const agentId = `soak-agent-${__VU}`;
  const payload = JSON.stringify({
    agent_id: agentId,
    status: 'online',
    printer_count: 1,
  });

  const startTime = Date.now();
  const response = http.post(`${REGISTRY_URL}/agents/${agentId}/heartbeat`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'SoakHeartbeat' },
  });
  const duration = Date.now() - startTime;

  return { response, duration, success: response.status === 200 };
}

/**
 * Main soak test scenario
 */
export default function () {
  const state = getVuState();

  group('Soak Test Operations', () => {
    soakIterations.add(1);

    // Vary operations over time to prevent hotspots
    const minutesElapsed = Math.floor((Date.now() - state.startTime) / 60000);
    const op = (__ITER + minutesElapsed) % 20;

    let result;

    if (op < 5) {
      // 25% - Heartbeat (most frequent, low cost)
      result = sendHeartbeat();

    } else if (op < 10) {
      // 25% - Job submission
      const accessToken = ensureAuthenticated(state);
      if (accessToken) {
        result = submitJob(accessToken);
        if (result.success) {
          state.jobsSubmitted++;
        }
      } else {
        result = { success: false, duration: 0 };
      }

    } else if (op < 15) {
      // 25% - Document upload
      const accessToken = ensureAuthenticated(state);
      if (accessToken) {
        result = uploadDocument(accessToken);
        if (result.success) {
          state.documentsUploaded++;
        }
      } else {
        result = { success: false, duration: 0 };
      }

    } else {
      // 25% - Job status query
      const jobId = `00000000-0000-4000-8000-00000000${(__ITER % 100).toString().padStart(3, '0')}`;
      const startTime = Date.now();
      const response = http.get(`${JOB_URL}/jobs/${jobId}`, {
        headers: BASE_HEADERS,
        tags: { name: 'SoakJobStatus' },
      });
      const duration = Date.now() - startTime;
      result = { response, duration, success: response.status === 200 || response.status === 404 };
    }

    soakSuccessRate.add(result.success);
    soakResponseTime.add(result.duration);

    if (!result.success) {
      state.errors++;
      soakErrors.add(1);
    }

    // Periodic status logging
    if (__ITER % 1000 === 0 && __VU === 0) {
      const elapsed = (Date.now() - state.startTime) / 60000;
      console.log(`Soak test progress: ${elapsed.toFixed(1)} minutes elapsed`);
      console.log(`Total iterations: ${soakIterations.count}`);
      console.log(`Total errors: ${soakErrors.count}`);
      console.log(`Current success rate: ${(soakSuccessRate.rate * 100).toFixed(2)}%`);
      console.log(`P95 response time: ${soakResponseTime.p('95').toFixed(0)}ms`);
    }

    // Estimate memory usage based on VU state size
    if (__VU === 0 && __ITER % 100 === 0) {
      activeConnections.set(vuState.size);
    }

    // Short sleep between operations
    sleep(Math.random() * 0.5 + 0.5);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Soak Test (Endurance Test)');
  console.log('========================================');
  console.log(`Duration: 30 minutes`);
  console.log(`Concurrent VUs: 100`);
  console.log('Purpose: Detect memory leaks, resource exhaustion');
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
  console.log('Soak Test Results');
  console.log('========================================');
  console.log(`Duration: ${(duration / 60).toFixed(1)} minutes`);
  console.log(`Total iterations: ${soakIterations.count}`);
  console.log(`Total errors: ${soakErrors.count}`);
  console.log(`Error rate: ${((soakErrors.count / soakIterations.count) * 100).toFixed(2)}%`);
  console.log(`Success rate: ${(soakSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P50 response time: ${soakResponseTime.p('50').toFixed(0)}ms`);
  console.log(`P95 response time: ${soakResponseTime.p('95').toFixed(0)}ms`);
  console.log(`P99 response time: ${soakResponseTime.p('99').toFixed(0)}ms`);
  console.log('\nPerformance stability check:');
  console.log(`Min response time: ${soakResponseTime.min.toFixed(0)}ms`);
  console.log(`Max response time: ${soakResponseTime.max.toFixed(0)}ms`);
  console.log(`Avg response time: ${soakResponseTime.avg.toFixed(0)}ms`);
  console.log('========================================');

  vuState.clear();
}
