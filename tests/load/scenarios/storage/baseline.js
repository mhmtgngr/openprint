/**
 * Storage Service Performance Baseline
 *
 * Establishes performance baseline for storage service
 * Covers document upload, download, metadata queries, and deletion
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { STORAGE_URL, TestData } from '../../lib/config.js';

// Custom metrics for baseline
export const baselineMetrics = {
  upload: new Trend('baseline_storage_upload'),
  download: new Trend('baseline_storage_download'),
  metadata: new Trend('baseline_storage_metadata'),
  list: new Trend('baseline_storage_list'),
  all: new Trend('baseline_storage_all'),
};

export const operationRates = {
  upload: new Rate('baseline_storage_upload_rate'),
  download: new Rate('baseline_storage_download_rate'),
  metadata: new Rate('baseline_storage_metadata_rate'),
  list: new Rate('baseline_storage_list_rate'),
};

// Test configuration
export const options = {
  scenarios: {
    storage_operations: {
      executor: 'constant-arrival-rate',
      rate: 20, // 20 operations per second (lower for storage)
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 20,
      maxVUs: 50,
    },
  },
  thresholds: {
    'baseline_storage_upload': ['p(95)<2000', 'p(99)<5000'],
    'baseline_storage_download': ['p(95)<1000', 'p(99)<2000'],
    'baseline_storage_metadata': ['p(95)<200', 'p(99)<400'],
    'baseline_storage_list': ['p(95)<300', 'p(99)<600'],
    'baseline_storage_all': ['p(95)<1500', 'p(99)<3000'],
    'http_req_failed': ['rate<0.02'],
  },
};

const BASE_HEADERS = {
  'User-Agent': 'k6-baseline-test/1.0',
};

// Test document IDs
const testDocumentIds = [];
for (let i = 0; i < 100; i++) {
  testDocumentIds.push(`00000000-0000-0000-0000-00000000${i.toString().padStart(4, '0')}`);
}

/**
 * Perform document upload
 */
function performUpload() {
  const fileName = `baseline-test-${__VU}-${__ITER}.pdf`;
  const fileSize = 100 * 1024; // 100KB
  const fileData = 'X'.repeat(fileSize);

  const boundary = `----Boundary${Math.random().toString(16).substring(2)}`;
  let body = '';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`;
  body += `Content-Type: application/pdf\r\n\r\n`;
  body += fileData;
  body += '\r\n';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="user_email"\r\n\r\n`;
  body += `baseline-test@example.com`;
  body += '\r\n';
  body += `--${boundary}--\r\n`;

  const startTime = Date.now();
  const response = http.post(`${STORAGE_URL}/documents`, body, {
    headers: { 'Content-Type': `multipart/form-data; boundary=${boundary}` },
    tags: { name: 'UploadDocument' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'upload status is 201 or 200': (r) => r.status === 201 || r.status === 200,
  });

  baselineMetrics.upload.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.upload.add(success);

  return { success, duration };
}

/**
 * Perform document download
 */
function performDownload() {
  const docId = testDocumentIds[__ITER % testDocumentIds.length];

  const startTime = Date.now();
  const response = http.get(`${STORAGE_URL}/documents/${docId}`, {
    headers: BASE_HEADERS,
    tags: { name: 'DownloadDocument' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'download status is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  baselineMetrics.download.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.download.add(success);

  return { success, duration };
}

/**
 * Perform metadata query
 */
function performMetadata() {
  const docId = testDocumentIds[__ITER % testDocumentIds.length];

  const startTime = Date.now();
  const response = http.get(`${STORAGE_URL}/documents/${docId}/metadata`, {
    headers: BASE_HEADERS,
    tags: { name: 'GetMetadata' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'metadata status is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  baselineMetrics.metadata.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.metadata.add(success);

  return { success, duration };
}

/**
 * Perform list documents
 */
function performList() {
  const startTime = Date.now();
  const response = http.get(`${STORAGE_URL}/documents?limit=50`, {
    headers: BASE_HEADERS,
    tags: { name: 'ListDocuments' },
  });
  const duration = Date.now() - startTime;

  const success = check(response, {
    'list status is 200': (r) => r.status === 200,
  });

  baselineMetrics.list.add(duration);
  baselineMetrics.all.add(duration);
  operationRates.list.add(success);

  return { success, duration };
}

/**
 * Main test scenario
 */
export default function () {
  group('Storage Baseline', () => {
    // Weighted operation distribution
    // 30% upload, 30% download, 25% metadata, 15% list
    const op = __ITER % 20;

    if (op < 6) {
      performUpload();
    } else if (op < 12) {
      performDownload();
    } else if (op < 17) {
      performMetadata();
    } else {
      performList();
    }
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Storage Service Baseline Test');
  console.log('========================================');
  console.log(`Target: ${STORAGE_URL}`);
  console.log('Duration: 2 minutes');
  console.log('Rate: 20 ops/sec');
  console.log('File size: 100KB');
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
  console.log('Storage Service Baseline Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);

  const results = {
    timestamp: new Date().toISOString(),
    duration,
    operations: {
      upload: {
        p50: baselineMetrics.upload.p('50'),
        p95: baselineMetrics.upload.p('95'),
        p99: baselineMetrics.upload.p('99'),
        avg: baselineMetrics.upload.avg,
        min: baselineMetrics.upload.min,
        max: baselineMetrics.upload.max,
        success_rate: operationRates.upload.rate,
      },
      download: {
        p50: baselineMetrics.download.p('50'),
        p95: baselineMetrics.download.p('95'),
        p99: baselineMetrics.download.p('99'),
        avg: baselineMetrics.download.avg,
        min: baselineMetrics.download.min,
        max: baselineMetrics.download.max,
        success_rate: operationRates.download.rate,
      },
      metadata: {
        p50: baselineMetrics.metadata.p('50'),
        p95: baselineMetrics.metadata.p('95'),
        p99: baselineMetrics.metadata.p('99'),
        avg: baselineMetrics.metadata.avg,
        min: baselineMetrics.metadata.min,
        max: baselineMetrics.metadata.max,
        success_rate: operationRates.metadata.rate,
      },
      list: {
        p50: baselineMetrics.list.p('50'),
        p95: baselineMetrics.list.p('95'),
        p99: baselineMetrics.list.p('99'),
        avg: baselineMetrics.list.avg,
        min: baselineMetrics.list.min,
        max: baselineMetrics.list.max,
        success_rate: operationRates.list.rate,
      },
      all: {
        p95: baselineMetrics.all.p('95'),
        p99: baselineMetrics.all.p('99'),
        avg: baselineMetrics.all.avg,
        success_rate: (
          operationRates.upload.rate +
          operationRates.download.rate +
          operationRates.metadata.rate +
          operationRates.list.rate
        ) / 4,
      },
    },
  };

  console.log(JSON.stringify(results, null, 2));
  console.log('========================================');
}
