/**
 * Reconnection Behavior Test
 *
 * Tests WebSocket reconnection behavior under unstable
 * network conditions
 */

import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import websocket from 'k6/ws';
import { NOTIFICATION_URL } from '../../lib/config.js';

// Custom metrics
export const reconnectSuccessRate = new Rate('reconnect_success_rate');
export const reconnectDuration = new Trend('reconnect_duration');
export const reconnectAttempts = new Counter('reconnect_attempts');
export const reconnectSuccesses = new Counter('reconnect_successes');
export const messagesDuringReconnect = new Counter('messages_during_reconnect');
export const stableConnections = new Gauge('stable_connections');

// Test configuration
export const options = {
  scenarios: {
    reconnection: {
      executor: 'constant-vus',
      vus: 100,
      duration: '5m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'reconnect_duration': ['p(95)<2000', 'p(99)<5000'],
    'reconnect_success_rate': ['rate>0.90'],
  },
};

/**
 * Build WebSocket URL
 */
export function getWebSocketURL(userId) {
  const protocol = NOTIFICATION_URL.startsWith('https') ? 'wss:' : 'ws:';
  const url = new URL(NOTIFICATION_URL);
  url.protocol = protocol;
  url.pathname = '/ws';
  url.searchParams.set('user_id', userId);
  return url.toString();
}

/**
 * Main test scenario
 */
export default function () {
  const userId = `reconnect-test-${__VU}`;
  const url = getWebSocketURL(userId);
  let reconnectCount = 0;
  const maxReconnects = 5;

  group('Reconnection Test', () => {
    const response = websocket.connect(url, {}, function (socket) {
      socket.on('open', () => {
        if (reconnectCount > 0) {
          reconnectSuccesses.add(1);
          reconnectSuccessRate.add(true);
        }
      });

      socket.on('message', (message) => {
        if (reconnectCount > 0) {
          messagesDuringReconnect.add(1);
        }
      });

      socket.on('error', (e) => {
        if (reconnectCount < maxReconnects) {
          reconnectAttempts.add(1);
          reconnectCount++;
        }
      });

      socket.setInterval(() => {
        // Simulate network instability by closing connection periodically
        if (Math.random() > 0.95 && reconnectCount < maxReconnects) {
          const reconnectStart = Date.now();
          socket.close();

          // Attempt reconnect
          setTimeout(() => {
            const reconnectDuration = Date.now() - reconnectStart;
            reconnectDuration.add(reconnectDuration);

            if (socket.isConnected()) {
              reconnectSuccesses.add(1);
            }
          }, Math.random() * 1000);
        }
      }, 30000); // Check every 30 seconds

      socket.setTimeout(() => {
        // Calculate stability metric
        if (reconnectCount === 0) {
          stableConnections.add(1);
        }
      }, 240000); // 4 minutes
    });

    if (response.error) {
      reconnectAttempts.add(1);
      reconnectSuccessRate.add(false);
    }
  });

  sleep(1);
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Reconnection Behavior Test');
  console.log('========================================');
  console.log(`Target: ${NOTIFICATION_URL}`);
  console.log(`Consecutive connections: 100`);
  console.log(`Max reconnects per client: 5`);
  console.log(`Duration: 5 minutes`);
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
  console.log('Reconnection Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Reconnect attempts: ${reconnectAttempts.count}`);
  console.log(`Reconnect successes: ${reconnectSuccesses.count}`);
  console.log(`Reconnect success rate: ${(reconnectSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`Stable connections: ${stableConnections.count}`);
  console.log(`P95 reconnect duration: ${reconnectDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 reconnect duration: ${reconnectDuration.p('99').toFixed(0)}ms`);
  console.log('========================================');
}
