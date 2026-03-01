/**
 * Concurrent Download Test
 *
 * Tests concurrent document downloads
 * Simulates users retrieving stored documents
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { STORAGE_URL } from '../../lib/config.js';

// Custom metrics
export const downloadSuccessRate = new Rate('download_success_rate');
export const downloadDuration = new Trend('download_duration');
export const downloadThroughput = new Trend('download_throughput_mbps');
export const bytesDownloaded = new Counter('bytes_downloaded');
export const activeDownloads = new Gauge('active_downloads');

// Predefined document IDs for testing
const testDocumentIds = [];
for (let i = 0; i < 1000; i++) {
  testDocumentIds.push(`00000000-0000-0000-0000-00000000${i.toString().padStart(4, '0')}`);
}

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m', target: 200 },
    { duration: '1m', target: 200 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'download_duration': ['p(95)<1000', 'p(99)<2000'],
    'download_success_rate': ['rate>0.95'],
    'http_req_duration': ['p(95)<2000'],
    'http_req_failed': ['rate<0.05'],
  },
};

const BASE_HEADERS = {
  'User-Agent': 'k6-load-test/1.0',
};

/**
 * Download document
 */
export function downloadDocument(documentId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'DownloadDocument' },
    timeout: '60s',
  };

  return http.get(`${STORAGE_URL}/documents/${documentId}`, params);
}

/**
 * Get document metadata
 */
export function getDocumentMetadata(documentId) {
  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'GetDocumentMetadata' },
  };

  return http.get(`${STORAGE_URL}/documents/${documentId}/metadata`, params);
}

/**
 * List documents
 */
export function listDocuments(userEmail = null, limit = 50) {
  let url = `${STORAGE_URL}/documents?limit=${limit}`;
  if (userEmail) {
    url += `&user_email=${userEmail}`;
  }

  const params = {
    headers: BASE_HEADERS,
    tags: { name: 'ListDocuments' },
  };

  return http.get(url, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Document Download', () => {
    // Mix of download operations
    const op = __ITER % 10;

    let response;
    let success;
    const startTime = Date.now();

    if (op < 7) {
      // 70% - Download document
      const docId = testDocumentIds[__ITER % testDocumentIds.length];
      response = downloadDocument(docId);

      success = check(response, {
        'download status is 200 or 404': (r) => r.status === 200 || r.status === 404,
        'has content when found': (r) => {
          if (r.status !== 200) return true;
          return r.body !== undefined && r.body.length > 0;
        },
      });

      if (response.status === 200) {
        const bytes = response.body.length;
        bytesDownloaded.add(bytes);
        const duration = Date.now() - startTime;
        const mbps = (bytes * 8 / 1000000) / (duration / 1000);
        downloadThroughput.add(mbps);
      }

      downloadDuration.add(response.timings.duration);

    } else if (op < 9) {
      // 20% - Get document metadata
      const docId = testDocumentIds[__ITER % testDocumentIds.length];
      response = getDocumentMetadata(docId);

      success = check(response, {
        'metadata status is 200 or 404': (r) => r.status === 200 || r.status === 404,
        'has metadata when found': (r) => {
          if (r.status !== 200) return true;
          try {
            const body = r.json();
            return body.document_id !== undefined || body.id !== undefined;
          } catch (e) {
            return false;
          }
        },
      });

      downloadDuration.add(response.timings.duration);

    } else {
      // 10% - List documents
      const userEmail = `loadtest-${__VU % 10}@example.com`;
      response = listDocuments(userEmail, 50);

      success = check(response, {
        'list status is 200': (r) => r.status === 200,
        'has documents array': (r) => {
          try {
            const body = r.json();
            return Array.isArray(body.documents) || Array.isArray(body.data);
          } catch (e) {
            return false;
          }
        },
      });

      downloadDuration.add(response.timings.duration);
    }

    downloadSuccessRate.add(success);
    activeDownloads.add(__VU);

    // Simulate user think time
    sleep(Math.random() * 1 + 0.5);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting concurrent download test');
  console.log(`Target endpoint: ${STORAGE_URL}`);
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('Document download test completed');
  console.log(`Bytes downloaded: ${bytesDownloaded.count}`);
  console.log(`Total MB downloaded: ${(bytesDownloaded.count / 1024 / 1024).toFixed(2)}`);
  console.log(`Download success rate: ${(downloadSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 download duration: ${downloadDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 download duration: ${downloadDuration.p('99').toFixed(0)}ms`);
  console.log(`Average throughput: ${downloadThroughput.avg.toFixed(2)} Mbps`);
}
