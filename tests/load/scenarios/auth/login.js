/**
 * User Login Load Test
 *
 * Tests the authentication service login endpoint under load.
 * FR-001: Concurrent User Login Simulation
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { BASE_URL, TEST_CREDENTIALS, STAGES } from '../../lib/config.js';
import { randomSleep } from '../../lib/helpers.js';

// Custom metrics
export const loginSuccessRate = new Rate('login_success_rate');
export const loginDuration = new Trend('login_duration');
export const loginErrors = new Rate('login_errors');

// Test configuration
export const options = {
  stages: STAGES.baseline,
  thresholds: {
    'login_duration': ['p(95)<300', 'p(99)<500'],
    'login_success_rate': ['rate>0.95'],
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'http_req_failed': ['rate<0.05'],
  },
};

const LOGIN_ENDPOINT = `${BASE_URL}/auth/login`;

/**
 * Perform user login
 */
export function login(email, password) {
  const payload = JSON.stringify({
    email: email || TEST_CREDENTIALS.email,
    password: password || TEST_CREDENTIALS.password,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'User-Agent': 'k6-load-test/1.0',
    },
    tags: { name: 'Login' },
  };

  return http.post(LOGIN_ENDPOINT, payload, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Authentication: Login', () => {
    const response = login();

    const success = check(response, {
      'login status is 200': (r) => r.status === 200,
      'login has access_token': (r) => {
        try {
          const body = r.json();
          return body.access_token !== undefined;
        } catch (e) {
          return false;
        }
      },
      'login has refresh_token': (r) => {
        try {
          const body = r.json();
          return body.refresh_token !== undefined;
        } catch (e) {
          return false;
        }
      },
      'login has user_id': (r) => {
        try {
          const body = r.json();
          return body.user_id !== undefined;
        } catch (e) {
          return false;
        }
      },
      'login response time < 500ms': (r) => r.timings.duration < 500,
    });

    loginSuccessRate.add(success);
    loginDuration.add(response.timings.duration);
    loginErrors.add(!success);

    if (!success) {
      console.error(`Login failed: ${response.status} - ${response.body}`);
    }

    // Small sleep between iterations to simulate realistic user behavior
    randomSleep(0.5, 2);
  });
}

/**
 * Setup function - runs once before test
 */
export function setup() {
  console.log(`Starting login load test against ${BASE_URL}`);
  console.log(`Using test user: ${TEST_CREDENTIALS.email}`);
}

/**
 * Teardown function - runs once after test
 */
export function teardown(data) {
  console.log('Login load test completed');
  console.log(`Success rate: ${loginSuccessRate.rate * 100}%`);
  console.log(`Average duration: ${loginDuration.avg}ms`);
  console.log(`P95 duration: ${loginDuration.p('95')}ms`);
}
