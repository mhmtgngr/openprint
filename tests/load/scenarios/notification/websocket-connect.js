/**
 * WebSocket Connection Scaling Test
 *
 * FR-005: Tests WebSocket connection handling under
 * concurrent connection load
 */

import { check, group } from 'k6';
import { Rate, Trend, Gauge, Counter } from 'k6/metrics';
import websocket from 'k6/ws';
import { NOTIFICATION_URL } from '../../lib/config.js';

// Custom metrics
export const wsConnectionSuccessRate = new Rate('ws_connection_success_rate');
export const wsConnectionDuration = new Trend('ws_connection_duration');
export const wsActiveConnections = new Gauge('ws_active_connections');
export const wsMessagesReceived = new Counter('ws_messages_received');
export const wsMessagesSent = new Counter('ws_messages_sent');
export const wsConnectionErrors = new Counter('ws_connection_errors');
export const wsReconnections = new Counter('ws_reconnections');

// Test configuration
export const options = {
  scenarios: {
    websocket_connections: {
      executor: 'constant-vus',
      vus: 500,
      duration: '5m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'ws_connection_duration': ['p(95)<500', 'p(99)<1000'],
    'ws_connection_success_rate': ['rate>0.95'],
    'ws_connection_errors': ['count<50'],
  },
};

/**
 * Build WebSocket URL
 */
export function getWebSocketURL(userId, orgId = null) {
  const protocol = NOTIFICATION_URL.startsWith('https') ? 'wss:' : 'ws:';
  const url = new URL(NOTIFICATION_URL);
  url.protocol = protocol;
  url.pathname = '/ws';
  url.searchParams.set('user_id', userId);
  if (orgId) {
    url.searchParams.set('org_id', orgId);
  }
  return url.toString();
}

/**
 * Main test scenario
 */
export default function () {
  const userId = `loadtest-user-${__VU}`;
  const orgId = `loadtest-org-${__VU % 10}`;

  group('WebSocket Connection', () => {
    const url = getWebSocketURL(userId, orgId);
    const startTime = Date.now();

    // Create WebSocket connection
    const response = websocket.connect(url, {}, function (socket) {
      const connectionDuration = Date.now() - startTime;

      socket.on('open', () => {
        wsConnectionSuccessRate.add(true);
        wsConnectionDuration.add(connectionDuration);
        wsActiveConnections.add(__VU);

        // Send authentication/ping message
        socket.send(JSON.stringify({
          type: 'ping',
          data: {},
        }));
        wsMessagesSent.add(1);
      });

      socket.on('message', (message) => {
        wsMessagesReceived.add(1);

        try {
          const data = JSON.parse(message);
          if (data.type === 'pong') {
            // Successfully received pong
          }
        } catch (e) {
          // Non-JSON message
        }
      });

      socket.on('error', (e) => {
        wsConnectionErrors.add(1);
        wsConnectionSuccessRate.add(false);
        console.error(`WebSocket error for VU ${__VU}:`, e.error());
      });

      socket.setTimeout(() => {
        // Send periodic ping to keep connection alive
        const pingInterval = setInterval(() => {
          if (socket.isConnected()) {
            try {
              socket.send(JSON.stringify({ type: 'ping' }));
              wsMessagesSent.add(1);
            } catch (e) {
              clearInterval(pingInterval);
            }
          } else {
            clearInterval(pingInterval);
          }
        }, 30000); // Ping every 30 seconds

        // Keep connection open for test duration
        socket.setTimeout(() => {
          clearInterval(pingInterval);
        }, 240000); // 4 minutes (within 5m test)
      }, 10000); // Initial check after 10 seconds
    });

    if (response.error) {
      wsConnectionErrors.add(1);
      wsConnectionSuccessRate.add(false);
      console.error(`Failed to connect WebSocket for VU ${__VU}:`, response.error);
    }
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('WebSocket Connection Scaling Test');
  console.log('========================================');
  console.log(`Target: ${NOTIFICATION_URL}`);
  console.log(`Concurrent connections: 500`);
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
  console.log('WebSocket Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Connection success rate: ${(wsConnectionSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`Messages received: ${wsMessagesReceived.count}`);
  console.log(`Messages sent: ${wsMessagesSent.count}`);
  console.log(`Connection errors: ${wsConnectionErrors.count}`);
  console.log(`P95 connection duration: ${wsConnectionDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 connection duration: ${wsConnectionDuration.p('99').toFixed(0)}ms`);
  console.log('========================================');
}
