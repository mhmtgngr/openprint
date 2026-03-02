/**
 * Token Refresh Under Load
 *
 * Tests the token refresh endpoint's ability to handle
 * concurrent refresh requests without performance degradation
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { BASE_URL, TEST_CREDENTIALS, STAGES } from '../../lib/config.js';
import { parseJWT, isTokenExpired, randomSleep } from '../../lib/helpers.js';

// Custom metrics
export const refreshSuccessRate = new Rate('refresh_success_rate');
export const refreshDuration = new Trend('refresh_duration');
export const refreshReuseRate = new Rate('refresh_token_reuse_rate');
export const accessTokenAge = new Trend('access_token_age_at_refresh');

// Track tokens per VU
const tokenStore = new Map();

// Test configuration
export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '30s', target: 100 },
    { duration: '1m', target: 500 },
    { duration: '30s', target: 100 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'refresh_duration': ['p(95)<200', 'p(99)<400'],
    'refresh_success_rate': ['rate>0.98'],
    'http_req_failed': ['rate<0.02'],
  },
};

const LOGIN_ENDPOINT = `${BASE_URL}/auth/login`;
const REFRESH_ENDPOINT = `${BASE_URL}/auth/refresh`;

/**
 * Login and get initial tokens
 */
export function login() {
  const payload = JSON.stringify({
    email: TEST_CREDENTIALS.email,
    password: TEST_CREDENTIALS.password,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'InitialLogin' },
  };

  return http.post(LOGIN_ENDPOINT, payload, params);
}

/**
 * Refresh access token
 */
export function refreshToken(refreshToken) {
  const payload = JSON.stringify({
    refresh_token: refreshToken,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'RefreshToken' },
  };

  return http.post(REFRESH_ENDPOINT, payload, params);
}

/**
 * Main test scenario
 */
export default function () {
  const vuId = __VU;

  group('Token Refresh', () => {
    let tokens = tokenStore.get(vuId);

    // Initial login if no tokens
    if (!tokens) {
      const response = login();

      if (check(response, {
        'initial login successful': (r) => r.status === 200,
        'has tokens': (r) => {
          try {
            const body = r.json();
            return body.access_token && body.refresh_token;
          } catch (e) {
            return false;
          }
        },
      })) {
        const body = response.json();
        tokens = {
          accessToken: body.access_token,
          refreshToken: body.refresh_token,
          issuedAt: Date.now(),
        };
        tokenStore.set(vuId, tokens);
      } else {
        console.error('Initial login failed');
        sleep(1);
        return;
      }
    }

    // Calculate token age
    const tokenAge = Date.now() - tokens.issuedAt;

    // Check if access token is expired or will expire soon
    let needsRefresh = false;

    try {
      const payload = parseJWT(tokens.accessToken);
      const expiresAt = payload.exp * 1000;
      const timeUntilExpiry = expiresAt - Date.now();

      // Refresh if less than 2 minutes remaining
      needsRefresh = timeUntilExpiry < 2 * 60 * 1000;

      accessTokenAge.add(tokenAge);
    } catch (e) {
      // If we can't parse, consider it needs refresh
      needsRefresh = true;
    }

    // Perform refresh
    if (needsRefresh || __ITER % 10 === 0) {
      const response = refreshToken(tokens.refreshToken);

      const success = check(response, {
        'refresh status is 200': (r) => r.status === 200,
        'new access token': (r) => {
          try {
            const body = r.json();
            return body.access_token !== undefined && body.access_token !== tokens.accessToken;
          } catch (e) {
            return false;
          }
        },
        'response time < 300ms': (r) => r.timings.duration < 300,
      });

      refreshSuccessRate.add(success);
      refreshDuration.add(response.timings.duration);

      if (success) {
        const body = response.json();

        // Check if refresh token was reused
        const reusedRefresh = body.refresh_token === undefined || body.refresh_token === tokens.refreshToken;
        refreshReuseRate.add(reusedRefresh);

        tokens = {
          accessToken: body.access_token,
          refreshToken: body.refresh_token || tokens.refreshToken,
          issuedAt: Date.now(),
        };
        tokenStore.set(vuId, tokens);
      } else {
        console.error(`Refresh failed: ${response.status}`);
      }
    }

    // Simulate time between token checks
    randomSleep(0.5, 2);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting token refresh load test');
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  console.log('Token refresh test completed');
  console.log(`Refresh success rate: ${(refreshSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`Average refresh duration: ${refreshDuration.avg.toFixed(0)}ms`);
  console.log(`P95 refresh duration: ${refreshDuration.p('95').toFixed(0)}ms`);
  console.log(`Refresh token reuse rate: ${(refreshReuseRate.rate * 100).toFixed(2)}%`);

  tokenStore.clear();
}
