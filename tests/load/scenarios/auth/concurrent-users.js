/**
 * Concurrent User Login Simulation
 *
 * FR-001: Simulates multiple users logging in concurrently
 * Tests session creation and token generation under concurrent load
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Gauge } from 'k6/metrics';
import { BASE_URL, TEST_CREDENTIALS, TestData, STAGES } from '../../lib/config.js';
import { randomSleep } from '../../lib/helpers.js';

// Custom metrics
export const concurrentLogins = new Gauge('concurrent_logins');
export const sessionCreationRate = new Rate('session_creation_rate');
export const tokenGenerationRate = new Rate('token_generation_rate');
export const activeSessions = new Gauge('active_sessions');

// Track sessions
const sessions = new Map();

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 100 },   // Ramp up to 100 users
    { duration: '2m', target: 500 },   // Peak: 500 concurrent users
    { duration: '1m', target: 500 },   // Sustain peak
    { duration: '1m', target: 100 },   // Ramp down
    { duration: '30s', target: 0 },    // Ramp down to 0
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'http_req_failed': ['rate<0.02'],
    'session_creation_rate': ['rate>0.95'],
    'token_generation_rate': ['rate>0.98'],
  },
};

const LOGIN_ENDPOINT = `${BASE_URL}/auth/login`;
const REFRESH_ENDPOINT = `${BASE_URL}/auth/refresh`;

/**
 * Perform login with unique user credentials
 */
export function userLogin(userId) {
  const email = `user_${userId}@loadtest.example.com`;
  const password = 'TestPassword123!';

  const payload = JSON.stringify({
    email,
    password,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'UserLogin', user_id: userId },
  };

  return http.post(LOGIN_ENDPOINT, payload, params);
}

/**
 * Refresh an existing session
 */
export function refreshSession(refreshToken) {
  const payload = JSON.stringify({
    refresh_token: refreshToken,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'RefreshToken' },
  };

  return http.post(REFRESH_ENDPOINT, payload, params);
}

/**
 * Main test scenario
 */
export default function (data) {
  const userId = __VU;
  const iter = __ITER;

  group('Concurrent Login Flow', () => {
    // First iteration: login
    if (iter === 0) {
      const response = userLogin(`${userId}_${iter}`);

      const sessionCreated = check(response, {
        'concurrent login status is 200': (r) => r.status === 200,
        'has access token': (r) => {
          try {
            const body = r.json();
            return body.access_token !== undefined;
          } catch (e) {
            return false;
          }
        },
        'has refresh token': (r) => {
          try {
            const body = r.json();
            return body.refresh_token !== undefined;
          } catch (e) {
            return false;
          }
        },
      });

      sessionCreationRate.add(sessionCreated);

      if (sessionCreated) {
        const body = response.json();
        sessions.set(userId, {
          accessToken: body.access_token,
          refreshToken: body.refresh_token,
          createdAt: Date.now(),
        });
        activeSessions.add(sessions.size);
        concurrentLogins.add(1);
      }
    } else {
      // Subsequent iterations: refresh token
      const session = sessions.get(userId);

      if (session) {
        const refreshAge = Date.now() - session.createdAt;

        // Refresh if token is getting old (>5 minutes)
        if (refreshAge > 5 * 60 * 1000) {
          const response = refreshSession(session.refreshToken);

          const refreshed = check(response, {
            'refresh status is 200': (r) => r.status === 200,
            'new access token': (r) => {
              try {
                const body = r.json();
                return body.access_token !== undefined;
              } catch (e) {
                return false;
              }
            },
          });

          tokenGenerationRate.add(refreshed);

          if (refreshed) {
            const body = response.json();
            sessions.set(userId, {
              ...session,
              accessToken: body.access_token,
              refreshToken: body.refresh_token || session.refreshToken,
              createdAt: Date.now(),
            });
          }
        }

        concurrentLogins.add(1);
      } else {
        // Session lost, re-login
        const response = userLogin(`${userId}_${iter}`);
        const sessionCreated = check(response, {
          're-login status is 200': (r) => r.status === 200,
        });

        sessionCreationRate.add(sessionCreated);

        if (sessionCreated) {
          const body = response.json();
          sessions.set(userId, {
            accessToken: body.access_token,
            refreshToken: body.refresh_token,
            createdAt: Date.now(),
          });
        }
      }
    }

    // Simulate realistic user activity pattern
    randomSleep(1, 5);
  });
}

/**
 * Setup - create test users if needed
 */
export function setup() {
  console.log('Setting up concurrent user login test');
  console.log(`Target endpoint: ${LOGIN_ENDPOINT}`);

  // In a real setup, you might create test users here
  // For this test, we assume users can be created on-the-fly
  // or use a pool of pre-created test users

  return {
    startTime: Date.now(),
  };
}

/**
 * Teardown - cleanup and report
 */
export function teardown(data) {
  console.log('Concurrent login test completed');
  console.log(`Total sessions created: ${sessions.size}`);
  console.log(`Session creation rate: ${(sessionCreationRate.rate * 100).toFixed(2)}%`);
  console.log(`Token generation rate: ${(tokenGenerationRate.rate * 100).toFixed(2)}%`);
  console.log(`Test duration: ${((Date.now() - data.startTime) / 1000).toFixed(0)}s`);

  sessions.clear();
}
