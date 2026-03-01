/**
 * Agent Discovery Load Test
 *
 * Tests the agent discovery and listing endpoints
 * under concurrent load
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Gauge } from 'k6/metrics';
import { REGISTRY_URL } from '../../lib/config.js';

// Custom metrics
export const discoverySuccessRate = new Rate('discovery_success_rate');
export const discoveryDuration = new Trend('discovery_duration');
export const agentsListed = new Gauge('agents_listed');
export const printersDiscovered = new Gauge('printers_discovered');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 100 },
    { duration: '1m', target: 100 },
    { duration: '30s', target: 20 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'discovery_duration': ['p(95)<500', 'p(99)<1000'],
    'discovery_success_rate': ['rate>0.95'],
    'http_req_duration': ['p(95)<800'],
    'http_req_failed': ['rate<0.05'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
};

/**
 * List all agents
 */
export function listAgents(limit = 50, offset = 0) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'ListAgents' },
  };

  const url = `${REGISTRY_URL}/agents?limit=${limit}&offset=${offset}`;
  return http.get(url, params);
}

/**
 * Get specific agent details
 */
export function getAgent(agentId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'GetAgent' },
  };

  return http.get(`${REGISTRY_URL}/agents/${agentId}`, params);
}

/**
 * List discovered printers for an agent
 */
export function listDiscoveredPrinters(agentId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'ListDiscoveredPrinters' },
  };

  return http.get(`${REGISTRY_URL}/agents/${agentId}/discovered-printers`, params);
}

/**
 * List all printers
 */
export function listPrinters(limit = 50, offset = 0) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'ListPrinters' },
  };

  const url = `${REGISTRY_URL}/printers?limit=${limit}&offset=${offset}`;
  return http.get(url, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Agent Discovery', () => {
    // Mix of discovery operations
    const op = __ITER % 10;

    let response;
    let operationName;

    if (op < 4) {
      // 40% - List agents
      response = listAgents(50, (__ITER * 50) % 1000);
      operationName = 'ListAgents';

      check(response, {
        'list agents status is 200': (r) => r.status === 200,
        'has agents array': (r) => {
          try {
            const body = r.json();
            return Array.isArray(body.agents) || Array.isArray(body.data);
          } catch (e) {
            return false;
          }
        },
      });

      if (response.status === 200) {
        try {
          const body = response.json();
          const count = body.agents?.length || body.data?.length || 0;
          agentsListed.set(count);
        } catch (e) {}
      }

    } else if (op < 7) {
      // 30% - Get agent details
      const agentId = `loadtest-agent-${(__ITER % 100)}`;
      response = getAgent(agentId);
      operationName = 'GetAgent';

      check(response, {
        'get agent status is 200 or 404': (r) => r.status === 200 || r.status === 404,
        'has agent data when found': (r) => {
          if (r.status !== 200) return true;
          try {
            const body = r.json();
            return body.agent_id !== undefined || body.id !== undefined;
          } catch (e) {
            return false;
          }
        },
      });

    } else if (op < 9) {
      // 20% - List discovered printers
      const agentId = `loadtest-agent-${(__ITER % 100)}`;
      response = listDiscoveredPrinters(agentId);
      operationName = 'ListDiscoveredPrinters';

      check(response, {
        'list printers status is 200 or 404': (r) => r.status === 200 || r.status === 404,
        'has printers array when found': (r) => {
          if (r.status !== 200) return true;
          try {
            const body = r.json();
            return Array.isArray(body.printers) || Array.isArray(body.data);
          } catch (e) {
            return false;
          }
        },
      });

      if (response.status === 200) {
        try {
          const body = response.json();
          const count = body.printers?.length || body.data?.length || 0;
          printersDiscovered.set(count);
        } catch (e) {}
      }

    } else {
      // 10% - List all printers
      response = listPrinters(50, (__ITER * 50) % 500);
      operationName = 'ListAllPrinters';

      check(response, {
        'list all printers status is 200': (r) => r.status === 200,
        'has data array': (r) => {
          try {
            const body = r.json();
            return Array.isArray(body.printers) || Array.isArray(body.data);
          } catch (e) {
            return false;
          }
        },
      });
    }

    const success = response.status >= 200 && response.status < 300;

    discoverySuccessRate.add(success);
    discoveryDuration.add(response.timings.duration);

    if (!success && response.status !== 404) {
      console.error(`${operationName} failed: ${response.status}`);
    }

    // Simulate user think time
    sleep(Math.random() * 1 + 0.5);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting agent discovery load test');
  console.log(`Target endpoint: ${REGISTRY_URL}`);
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  console.log('Agent discovery test completed');
  console.log(`Success rate: ${(discoverySuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 duration: ${discoveryDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 duration: ${discoveryDuration.p('99').toFixed(0)}ms`);
}
