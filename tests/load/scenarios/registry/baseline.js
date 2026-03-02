/**
 * Registry Service Performance Baseline
 *
 * Establishes performance baseline for registry service
 * Covers agent registration, heartbeat, discovery, and printer operations
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { REGISTRY_URL, TestData } from '../../lib/config.js';

// Custom metrics for baseline
export const baselineMetrics = {
  heartbeat: new Trend('baseline_registry_heartbeat'),
  discovery: new Trend('baseline_registry_discovery'),
  registration: new Trend('baseline_registry_registration'),
  printerList: new Trend('baseline_registry_printer_list'),
  all: new Trend('baseline_registry_all'),
};

export const operationRates = {
  heartbeat: new Rate('baseline_registry_heartbeat_rate'),
  discovery: new Rate('baseline_registry_discovery_rate'),
  registration: new Rate('baseline_registry_registration_rate'),
  printerList: new Rate('baseline_registry_printer_list_rate'),
};

// Test configuration
export const options = {
  scenarios: {
    registry_operations: {
      executor: 'constant-arrival-rate',
      rate: 100, // 100 operations per second
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    'baseline_registry_heartbeat': ['p(95)<100', 'p(99)<200'],
    'baseline_registry_discovery': ['p(95)<500', 'p(99)<1000'],
    'baseline_registry_registration': ['p(95)<500', 'p(99)<1000'],
    'baseline_registry_printer_list': ['p(95)<300', 'p(99)<600'],
    'baseline_registry_all': ['p(95)<300', 'p(99)<600'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-baseline-test/1.0',
};

// Track registered agents
const registeredAgents = [];

/**
 * Perform heartbeat operation
 */
function performHeartbeat() {
  const agentId = registeredAgents.length > 0
    ? registeredAgents[__ITER % registeredAgents.length]
    : `baseline-agent-${__VU}`;

  const payload = JSON.stringify({
    agent_id: agentId,
    status: 'online',
    printer_count: Math.floor(Math.random() * 3),
    completed_jobs: Math.floor(Math.random() * 50),
    version: '2.0.0',
  });

  const startTime = Date.now();
  const response = http.post(`${REGISTRY_URL}/agents/${agentId}/heartbeat`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'Heartbeat' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'heartbeat status is 200': (r) => r.status === 200,
  });

  baselineMetrics.heartbeat.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.heartbeat.add(success);

  return { success, duration };
}

/**
 * Perform discovery/list operation
 */
function performDiscovery() {
  const startTime = Date.now();
  const response = http.get(`${REGISTRY_URL}/agents?limit=50`, {
    headers: BASE_HEADERS,
    tags: { name: 'ListAgents' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'discovery status is 200': (r) => r.status === 200,
    'has data': (r) => {
      try {
        const body = r.json();
        return Array.isArray(body.agents) || Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    },
  });

  baselineMetrics.discovery.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.discovery.add(success);

  return { success, duration };
}

/**
 * Perform registration operation
 */
function performRegistration() {
  const agentId = `baseline-agent-${__VU}-${__ITER}`;

  const payload = JSON.stringify({
    name: agentId,
    type: Math.random() > 0.5 ? 'desktop' : 'mobile',
    version: '2.0.0',
  });

  const startTime = Date.now();
  const response = http.post(`${REGISTRY_URL}/agents/register`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'RegisterAgent' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'registration status is 201 or 200': (r) => r.status === 201 || r.status === 200,
  });

  baselineMetrics.registration.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.registration.add(success);

  if (success && registeredAgents.length < 100) {
    registeredAgents.push(agentId);
  }

  return { success, duration };
}

/**
 * Perform printer list operation
 */
function performPrinterList() {
  const startTime = Date.now();
  const response = http.get(`${REGISTRY_URL}/printers?limit=50`, {
    headers: BASE_HEADERS,
    tags: { name: 'ListPrinters' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'printer list status is 200': (r) => r.status === 200,
  });

  baselineMetrics.printerList.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.printerList.add(success);

  return { success, duration };
}

/**
 * Main test scenario
 */
export default function () {
  group('Registry Baseline', () => {
    // Weighted operation distribution
    // 50% heartbeat (most common), 20% discovery, 15% registration, 15% printer list
    const op = __ITER % 20;

    if (op < 10) {
      performHeartbeat();
    } else if (op < 14) {
      performDiscovery();
    } else if (op < 17) {
      performRegistration();
    } else {
      performPrinterList();
    }
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Registry Service Baseline Test');
  console.log('========================================');
  console.log(`Target: ${REGISTRY_URL}`);
  console.log('Duration: 2 minutes');
  console.log('Rate: 100 ops/sec');
  console.log('========================================');

  // Register some initial agents
  for (let i = 0; i < 50; i++) {
    registeredAgents.push(`baseline-agent-${i}`);
  }

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
  console.log('Registry Service Baseline Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);

  const results = {
    timestamp: new Date().toISOString(),
    duration,
    operations: {
      heartbeat: {
        p50: baselineMetrics.heartbeat.p('50'),
        p95: baselineMetrics.heartbeat.p('95'),
        p99: baselineMetrics.heartbeat.p('99'),
        avg: baselineMetrics.heartbeat.avg,
        min: baselineMetrics.heartbeat.min,
        max: baselineMetrics.heartbeat.max,
        success_rate: operationRates.heartbeat.rate,
      },
      discovery: {
        p50: baselineMetrics.discovery.p('50'),
        p95: baselineMetrics.discovery.p('95'),
        p99: baselineMetrics.discovery.p('99'),
        avg: baselineMetrics.discovery.avg,
        min: baselineMetrics.discovery.min,
        max: baselineMetrics.discovery.max,
        success_rate: operationRates.discovery.rate,
      },
      registration: {
        p50: baselineMetrics.registration.p('50'),
        p95: baselineMetrics.registration.p('95'),
        p99: baselineMetrics.registration.p('99'),
        avg: baselineMetrics.registration.avg,
        min: baselineMetrics.registration.min,
        max: baselineMetrics.registration.max,
        success_rate: operationRates.registration.rate,
      },
      printer_list: {
        p50: baselineMetrics.printerList.p('50'),
        p95: baselineMetrics.printerList.p('95'),
        p99: baselineMetrics.printerList.p('99'),
        avg: baselineMetrics.printerList.avg,
        min: baselineMetrics.printerList.min,
        max: baselineMetrics.printerList.max,
        success_rate: operationRates.printerList.rate,
      },
      all: {
        p95: baselineMetrics.all.p('95'),
        p99: baselineMetrics.all.p('99'),
        avg: baselineMetrics.all.avg,
        success_rate: (
          operationRates.heartbeat.rate +
          operationRates.discovery.rate +
          operationRates.registration.rate +
          operationRates.printerList.rate
        ) / 4,
      },
    },
  };

  console.log(JSON.stringify(results, null, 2));
  console.log('========================================');
}
