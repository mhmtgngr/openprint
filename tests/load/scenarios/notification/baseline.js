/**
 * Notification Service Performance Baseline
 *
 * Establishes performance baseline for notification service
 * Covers WebSocket connections, message delivery, and broadcasting
 */

import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import websocket from 'k6/ws';
import http from 'k6/http';
import { NOTIFICATION_URL } from '../../lib/config.js';

// Custom metrics for baseline
export const baselineMetrics = {
  connect: new Trend('baseline_notification_connect'),
  message: new Trend('baseline_notification_message'),
  broadcast: new Trend('baseline_notification_broadcast'),
  all: new Trend('baseline_notification_all'),
};

export const operationRates = {
  connect: new Rate('baseline_notification_connect_rate'),
  message: new Rate('baseline_notification_message_rate'),
  broadcast: new Rate('baseline_notification_broadcast_rate'),
};

// Test configuration
export const options = {
  scenarios: {
    notification_operations: {
      executor: 'constant-arrival-rate',
      rate: 50,
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 100,
    },
  },
  thresholds: {
    'baseline_notification_connect': ['p(95)<500', 'p(99)<1000'],
    'baseline_notification_message': ['p(95)<100', 'p(99)<200'],
    'baseline_notification_broadcast': ['p(95)<300', 'p(99)<600'],
    'baseline_notification_all': ['p(95)<300', 'p(99)<600'],
    'http_req_failed': ['rate<0.02'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-baseline-test/1.0',
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
 * Perform WebSocket connection test
 */
function performConnect() {
  const userId = `baseline-user-${__VU}-${__ITER}`;
  const url = getWebSocketURL(userId);

  const startTime = Date.now();

  const response = websocket.connect(url, {}, function (socket) {
    socket.on('open', () => {
      const duration = Date.now() - startTime;
      baselineMetrics.connect.add(duration);
      baselineMetrics.all.add(duration);
      operationRates.connect.add(true);
    });

    socket.on('error', () => {
      baselineMetrics.connect.add(Date.now() - startTime);
      baselineMetrics.all.add(Date.now() - startTime);
      operationRates.connect.add(false);
    });

    socket.setTimeout(() => {
      socket.close();
    }, 5000); // Close after 5 seconds
  });

  return response;
}

/**
 * Perform message send test
 */
function performMessage() {
  const payload = JSON.stringify({
    type: 'test_message',
    data: {
      message: `Baseline test message ${__ITER}`,
      timestamp: Date.now(),
    },
  });

  const startTime = Date.now();
  const response = http.post(`${NOTIFICATION_URL}/broadcast`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'SendMessage' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'message status is 200': (r) => r.status === 200,
  });

  baselineMetrics.message.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.message.add(success);

  return { success, duration };
}

/**
 * Perform broadcast test
 */
function performBroadcast() {
  const payload = JSON.stringify({
    type: 'notification',
    data: {
      title: `Baseline Broadcast ${__ITER}`,
      body: 'Test notification',
      org_id: `baseline-org-${__ITER % 10}`,
    },
  });

  const startTime = Date.now();
  const response = http.post(`${NOTIFICATION_URL}/broadcast`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'Broadcast' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'broadcast status is 200': (r) => r.status === 200,
  });

  baselineMetrics.broadcast.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.broadcast.add(success);

  return { success, duration };
}

/**
 * Perform connection stats query
 */
function performStats() {
  const startTime = Date.now();
  const response = http.get(`${NOTIFICATION_URL}/connections`, {
    headers: BASE_HEADERS,
    tags: { name: 'ConnectionStats' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'stats status is 200': (r) => r.status === 200,
  });

  baselineMetrics.message.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.message.add(success);

  return { success, duration };
}

/**
 * Main test scenario
 */
export default function () {
  group('Notification Baseline', () => {
    // Weighted operation distribution
    // 30% connect, 30% message, 30% broadcast, 10% stats
    const op = __ITER % 20;

    if (op < 6) {
      performConnect();
    } else if (op < 12) {
      performMessage();
    } else if (op < 18) {
      performBroadcast();
    } else {
      performStats();
    }
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Notification Service Baseline Test');
  console.log('========================================');
  console.log(`Target: ${NOTIFICATION_URL}`);
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
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('========================================');
  console.log('Notification Service Baseline Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);

  const results = {
    timestamp: new Date().toISOString(),
    duration,
    operations: {
      connect: {
        p50: baselineMetrics.connect.p('50'),
        p95: baselineMetrics.connect.p('95'),
        p99: baselineMetrics.connect.p('99'),
        avg: baselineMetrics.connect.avg,
        min: baselineMetrics.connect.min,
        max: baselineMetrics.connect.max,
        success_rate: operationRates.connect.rate,
      },
      message: {
        p50: baselineMetrics.message.p('50'),
        p95: baselineMetrics.message.p('95'),
        p99: baselineMetrics.message.p('99'),
        avg: baselineMetrics.message.avg,
        min: baselineMetrics.message.min,
        max: baselineMetrics.message.max,
        success_rate: operationRates.message.rate,
      },
      broadcast: {
        p50: baselineMetrics.broadcast.p('50'),
        p95: baselineMetrics.broadcast.p('95'),
        p99: baselineMetrics.broadcast.p('99'),
        avg: baselineMetrics.broadcast.avg,
        min: baselineMetrics.broadcast.min,
        max: baselineMetrics.broadcast.max,
        success_rate: operationRates.broadcast.rate,
      },
      all: {
        p95: baselineMetrics.all.p('95'),
        p99: baselineMetrics.all.p('99'),
        avg: baselineMetrics.all.avg,
        success_rate: (
          operationRates.connect.rate +
          operationRates.message.rate +
          operationRates.broadcast.rate
        ) / 3,
      },
    },
  };

  console.log(JSON.stringify(results, null, 2));
  console.log('========================================');
}
