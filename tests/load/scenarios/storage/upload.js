/**
 * Document Upload Burst Test
 *
 * FR-004: Tests the storage service's ability to handle
 * burst document uploads without performance degradation
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { STORAGE_URL, TestData } from '../../lib/config.js';

// Custom metrics
export const uploadSuccessRate = new Rate('upload_success_rate');
export const uploadDuration = new Trend('upload_duration');
export const uploadThroughput = new Trend('upload_throughput_mbps');
export const bytesUploaded = new Counter('bytes_uploaded');
export const documentsStored = new Counter('documents_stored');
export const activeUploads = new Gauge('active_uploads');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 100 },
    { duration: '2m', target: 500 },  // Burst: 500 concurrent uploads
    { duration: '1m', target: 500 },
    { duration: '30s', target: 10 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'upload_duration': ['p(95)<2000', 'p(99)<5000'],
    'upload_success_rate': ['rate>0.95'],
    'http_req_duration': ['p(95)<5000'],
    'http_req_failed': ['rate<0.05'],
  },
};

const UPLOAD_ENDPOINT = `${STORAGE_URL}/documents`;

/**
 * Generate test file data
 */
export function generateFileData(sizeKB = 100) {
  // Create a simple text file pattern
  const pattern = 'OpenPrint test document data. ';
  const repeats = Math.ceil((sizeKB * 1024) / pattern.length);
  return pattern.repeat(repeats).substring(0, sizeKB * 1024);
}

/**
 * Upload a document
 */
export function uploadDocument(fileName, fileData, mimeType, userEmail) {
  const boundary = `----WebKitFormBoundary${Math.random().toString(16).substring(2)}`;

  let body = '';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`;
  body += `Content-Type: ${mimeType}\r\n\r\n`;
  body += fileData;
  body += '\r\n';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="user_email"\r\n\r\n`;
  body += userEmail;
  body += '\r\n';
  body += `--${boundary}--\r\n`;

  const params = {
    headers: {
      'Content-Type': `multipart/form-data; boundary=${boundary}`,
    },
    tags: { name: 'UploadDocument' },
    timeout: '300s', // 5 minute timeout for large uploads
  };

  return http.post(UPLOAD_ENDPOINT, body, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Document Upload', () => {
    // Vary file sizes for more realistic testing
    const sizeKB = [50, 100, 500, 1024, 2048][__ITER % 5];
    const fileName = `test-document-${__VU}-${__ITER}-${Date.now()}.pdf`;
    const fileData = generateFileData(sizeKB);
    const mimeType = 'application/pdf';
    const userEmail = `loadtest-${__VU}@example.com`;

    activeUploads.add(__VU);

    const startTime = Date.now();
    const response = uploadDocument(fileName, fileData, mimeType, userEmail);
    const duration = Date.now() - startTime;

    const success = check(response, {
      'upload status is 201 or 200': (r) => r.status === 201 || r.status === 200,
      'has document_id': (r) => {
        try {
          const body = r.json();
          return body.document_id !== undefined || body.id !== undefined;
        } catch (e) {
          return false;
        }
      },
      'has size': (r) => {
        try {
          const body = r.json();
          return body.size !== undefined;
        } catch (e) {
          return false;
        }
      },
      'response time < 5s': (r) => r.timings.duration < 5000,
    });

    uploadSuccessRate.add(success);
    uploadDuration.add(duration);

    // Calculate throughput in Mbps
    const bytes = fileData.length;
    const mbps = (bytes * 8 / 1000000) / (duration / 1000);
    uploadThroughput.add(mbps);

    if (success) {
      bytesUploaded.add(bytes);
      documentsStored.add(1);
    }

    // Simulate user delay between uploads
    sleep(Math.random() * 2 + 1);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('Starting document upload burst test');
  console.log(`Target endpoint: ${UPLOAD_ENDPOINT}`);
  console.log('File sizes: 50KB, 100KB, 500KB, 1MB, 2MB');
  return { startTime: Date.now() };
}

/**
 * Teardown function
 */
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;

  console.log('Document upload test completed');
  console.log(`Documents stored: ${documentsStored.count}`);
  console.log(`Bytes uploaded: ${bytesUploaded.count}`);
  console.log(`Total MB uploaded: ${(bytesUploaded.count / 1024 / 1024).toFixed(2)}`);
  console.log(`Upload success rate: ${(uploadSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`Average throughput: ${uploadThroughput.avg.toFixed(2)} Mbps`);
  console.log(`P95 upload duration: ${uploadDuration.p('95').toFixed(0)}ms`);
  console.log(`P99 upload duration: ${uploadDuration.p('99').toFixed(0)}ms`);
  console.log(`Upload rate: ${(documentsStored.count / duration).toFixed(2)} docs/sec`);
}
