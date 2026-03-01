/**
 * Auth Service Performance Baseline
 *
 * FR-007: Establishes performance baseline for authentication service
 * Covers login, logout, token refresh, and profile operations
 *
 * Results are saved as JSON artifacts in CI for regression detection (FR-008)
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { BASE_URL, TEST_CREDENTIALS } from '../../lib/config.js';
import { isSuccess, getJson } from '../../lib/helpers.js';

// Custom metrics for baseline
export const baselineMetrics = {
  login: new Trend('baseline_auth_login'),
  logout: new Trend('baseline_auth_logout'),
  refresh: new Trend('baseline_auth_refresh'),
  profile: new Trend('baseline_auth_profile'),
  all: new Trend('baseline_auth_all'),
};

export const operationRates = {
  login: new Rate('baseline_auth_login_rate'),
  logout: new Rate('baseline_auth_logout_rate'),
  refresh: new Rate('baseline_auth_refresh_rate'),
  profile: new Rate('baseline_auth_profile_rate'),
};

// Test configuration - moderate, steady load
export const options = {
  scenarios: {
    auth_operations: {
      executor: 'constant-arrival-rate',
      rate: 50, // 50 operations per second
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 100,
    },
  },
  thresholds: {
    'baseline_auth_login': ['p(95)<300', 'p(99)<500'],
    'baseline_auth_logout': ['p(95)<200', 'p(99)<400'],
    'baseline_auth_refresh': ['p(95)<200', 'p(99)<400'],
    'baseline_auth_profile': ['p(95)<150', 'p(99)<300'],
    'baseline_auth_all': ['p(95)<250', 'p(99)<500'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-baseline-test/1.0',
};

// Store session data per VU
const sessions = new Map();

/**
 * Perform login operation
 */
function performLogin() {
  const payload = JSON.stringify({
    email: TEST_CREDENTIALS.email,
    password: TEST_CREDENTIALS.password,
  });

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'Login' },
  });
  const duration = Date.now() - startTime;

  const success = isSuccess(response, 'Login');
  baselineMetrics.login.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.login.add(success);

  if (success) {
    const body = getJson(response);
    if (body) {
      sessions.set(__VU, {
        accessToken: body.access_token,
        refreshToken: body.refresh_token,
        userId: body.user_id,
      });
    }
  }

  return { success, response, duration };
}

/**
 * Perform logout operation
 */
function performLogout(refreshToken) {
  if (!refreshToken) {
    operationRates.logout.add(false);
    return { success: false, duration: 0 };
  }

  const payload = JSON.stringify({
    refresh_token: refreshToken,
  });

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/auth/logout`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'Logout' },
  });
  const duration = Date.now() - startTime;

  const success = isSuccess(response, 'Logout');
  baselineMetrics.logout.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.logout.add(success);

  return { success, response, duration };
}

/**
 * Perform token refresh operation
 */
function performRefresh(refreshToken) {
  if (!refreshToken) {
    operationRates.refresh.add(false);
    return { success: false, duration: 0 };
  }

  const payload = JSON.stringify({
    refresh_token: refreshToken,
  });

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/auth/refresh`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'Refresh' },
  });
  const duration = Date.now() - startTime;

  const success = isSuccess(response, 'Refresh');
  baselineMetrics.refresh.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.refresh.add(success);

  if (success) {
    const body = getJson(response);
    if (body && body.access_token) {
      const session = sessions.get(__VU) || {};
      session.accessToken = body.access_token;
      if (body.refresh_token) {
        session.refreshToken = body.refresh_token;
      }
      sessions.set(__VU, session);
    }
  }

  return { success, response, duration };
}

/**
 * Perform profile fetch operation
 */
function performProfile(accessToken) {
  if (!accessToken) {
    operationRates.profile.add(false);
    return { success: false, duration: 0 };
  }

  const headers = {
    ...BASE_HEADERS,
    'Authorization': `Bearer ${accessToken}`,
  };

  const startTime = Date.now();
  const response = http.get(`${BASE_URL}/auth/profile`, {
    headers,
    tags: { name: 'Profile' },
  });
  const duration = Date.now() - startTime;

  const success = isSuccess(response, 'Profile');
  baselineMetrics.profile.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.profile.add(success);

  return { success, response, duration };
}

/**
 * Main test scenario - cycles through all auth operations
 */
export default function () {
  const iteration = __ITER;
  const session = sessions.get(__VU);

  group('Auth Baseline', () => {
    // Weighted operation distribution
    // 40% login, 20% logout, 25% refresh, 15% profile
    const op = iteration % 20;

    if (op < 8) {
      // Login operations
      performLogin();
    } else if (op < 12) {
      // Logout operations
      performLogout(session?.refreshToken);
      if (op === 11) {
        sessions.delete(__VU);
      }
    } else if (op < 17) {
      // Refresh operations
      performRefresh(session?.refreshToken);
    } else {
      // Profile operations
      performProfile(session?.accessToken);
    }
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Auth Service Baseline Test');
  console.log('========================================');
  console.log(`Target: ${BASE_URL}`);
  console.log(`Test User: ${TEST_CREDENTIALS.email}`);
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
  const endTime = Date.now();
  const duration = (endTime - data.startTime) / 1000;

  console.log('========================================');
  console.log('Auth Service Baseline Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);

  const results = {
    timestamp: new Date().toISOString(),
    duration,
    operations: {
      login: {
        p50: baselineMetrics.login.p('50'),
        p95: baselineMetrics.login.p('95'),
        p99: baselineMetrics.login.p('99'),
        avg: baselineMetrics.login.avg,
        min: baselineMetrics.login.min,
        max: baselineMetrics.login.max,
        success_rate: operationRates.login.rate,
      },
      logout: {
        p50: baselineMetrics.logout.p('50'),
        p95: baselineMetrics.logout.p('95'),
        p99: baselineMetrics.logout.p('99'),
        avg: baselineMetrics.logout.avg,
        min: baselineMetrics.logout.min,
        max: baselineMetrics.logout.max,
        success_rate: operationRates.logout.rate,
      },
      refresh: {
        p50: baselineMetrics.refresh.p('50'),
        p95: baselineMetrics.refresh.p('95'),
        p99: baselineMetrics.refresh.p('99'),
        avg: baselineMetrics.refresh.avg,
        min: baselineMetrics.refresh.min,
        max: baselineMetrics.refresh.max,
        success_rate: operationRates.refresh.rate,
      },
      profile: {
        p50: baselineMetrics.profile.p('50'),
        p95: baselineMetrics.profile.p('95'),
        p99: baselineMetrics.profile.p('99'),
        avg: baselineMetrics.profile.avg,
        min: baselineMetrics.profile.min,
        max: baselineMetrics.profile.max,
        success_rate: operationRates.profile.rate,
      },
      all: {
        p95: baselineMetrics.all.p('95'),
        p99: baselineMetrics.all.p('99'),
        avg: baselineMetrics.all.avg,
        success_rate: (
          operationRates.login.rate +
          operationRates.logout.rate +
          operationRates.refresh.rate +
          operationRates.profile.rate
        ) / 4,
      },
    },
  };

  console.log(JSON.stringify(results, null, 2));
  console.log('========================================');

  // Export to environment variable for CI/CD
  // In CI, this would be saved as an artifact
  if (__ENV.BASELINE_OUTPUT) {
    console.log(`Baseline results would be saved to: ${__ENV.BASELINE_OUTPUT}`);
  }

  sessions.clear();
}
