/**
 * Broadcast Message Test
 *
 * Tests the notification service's ability to broadcast
 * messages to many connected clients
 */

import { check, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import websocket from 'k6/ws';
import http from 'k6/http';
import { NOTIFICATION_URL } from '../../lib/config.js';

// Custom metrics
export const broadcastSuccessRate = new Rate('broadcast_success_rate');
export const broadcastDuration = new Trend('broadcast_duration');
export const messagesBroadcast = new Counter('messages_broadcast');
export const messagesReceived = new Counter('broadcast_messages_received');
export const deliveryLatency = new Trend('broadcast_delivery_latency');

// Track message delivery
const messageTimestamps = new Map();

// Test configuration
export const options = {
  scenarios: {
    broadcasters: {
      executor: 'constant-vus',
      vus: 10,
      duration: '3m',
    },
    receivers: {
      executor: 'constant-vus',
      vus: 100,
      startTime: '10s',
      duration: '2m50s',
      gracefulStop: '10s',
    },
  },
  thresholds: {
    'broadcast_duration': ['p(95)<200', 'p(99)<500'],
    'broadcast_success_rate': ['rate>0.95'],
    'http_req_duration': ['p(95)<300'],
  },
};

const BROADCAST_ENDPOINT = `${NOTIFICATION_URL}/broadcast`;

/**
 * Send broadcast message via HTTP
 */
export function sendBroadcast(type, data, userId = null, orgId = null) {
  const payload = JSON.stringify({
    type,
    data,
    user_id: userId,
    org_id: orgId,
  });

  const startTime = Date.now();
  const response = http.post(BROADCAST_ENDPOINT, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'SendBroadcast' },
  });
  const duration = Date.now() - startTime;

  return { response, duration };
}

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
export default function (data) {
  const isBroadcaster = __VU <= 10;

  if (isBroadcaster) {
    // Broadcaster scenario - send messages via HTTP
    group('Broadcast Sender', () => {
      const orgId = `test-org-${__ITER % 5}`;
      const messageData = {
        title: `Broadcast ${__ITER}`,
        body: `Test broadcast message from VU ${__VU}`,
        timestamp: Date.now(),
      };

      const { response, duration } = sendBroadcast('notification', messageData, null, orgId);

      const success = check(response, {
        'broadcast status is 200': (r) => r.status === 200,
      });

      broadcastSuccessRate.add(success);
      broadcastDuration.add(duration);

      if (success) {
        messagesBroadcast.add(1);
        messageTimestamps.set(__ITER, Date.now());
      }

      sleep(Math.random() * 2 + 1);
    });

  } else {
    // Receiver scenario - connect via WebSocket
    group('Broadcast Receiver', () => {
      const userId = `receiver-user-${__VU}`;
      const orgId = `test-org-${__VU % 5}`;
      const url = getWebSocketURL(userId);

      websocket.connect(url, {}, function (socket) {
        socket.on('open', () => {
          // Subscribe to organization
          socket.send(JSON.stringify({
            type: 'subscribe',
            data: { org_id: orgId },
          }));
        });

        socket.on('message', (message) => {
          messagesReceived.add(1);

          try {
            const data = JSON.parse(message);
            if (data.type === 'notification' || data.type === 'broadcast') {
              // Calculate delivery latency if timestamp is present
              if (data.data && data.data.timestamp) {
                const latency = Date.now() - data.data.timestamp;
                deliveryLatency.add(latency);
              }
            }
          } catch (e) {}
        });

        socket.on('error', (e) => {
          console.error(`WebSocket error for receiver VU ${__VU}:`, e.error());
        });

        socket.setTimeout(() => {
          // Keep connection open
        }, 120000); // 2 minutes
      });

      sleep(1);
    });
  }
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Broadcast Message Test');
  console.log('========================================');
  console.log(`Target: ${NOTIFICATION_URL}`);
  console.log(`Broadcasters: 10`);
  console.log(`Receivers: 100`);
  console.log(`Duration: 3 minutes`);
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
  console.log('Broadcast Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Messages broadcast: ${messagesBroadcast.count}`);
  console.log(`Messages received: ${messagesReceived.count}`);
  console.log(`Broadcast success rate: ${(broadcastSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 broadcast duration: ${broadcastDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 broadcast duration: ${broadcastDuration.p('99').toFixed(0)}ms`);
  if (deliveryLatency.count > 0) {
    console.log(`P95 delivery latency: ${deliveryLatency.p('95').toFixed(0)}ms`);
  }
  console.log('========================================');

  messageTimestamps.clear();
}
