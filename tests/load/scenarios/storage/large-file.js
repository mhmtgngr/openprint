/**
 * Large File Handling Test
 *
 * Tests upload and download of large files (5MB - 50MB)
 * Validates performance with substantial file sizes
 */

import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { STORAGE_URL } from '../../lib/config.js';

// Custom metrics
export const largeFileSuccessRate = new Rate('large_file_success_rate');
export const largeFileUploadDuration = new Trend('large_file_upload_duration');
export const largeFileDownloadDuration = new Trend('large_file_download_duration');
export const largeFileThroughput = new Trend('large_file_throughput_mbps');
export const largeFileBytesTransferred = new Counter('large_file_bytes_transferred');

// File size configurations (in MB)
const FILE_SIZES = [5, 10, 25, 50];

// Test configuration
export const options = {
  scenarios: {
    large_files: {
      executor: 'constant-vus',
      vus: 20,
      duration: '5m',
      gracefulStop: '1m',
    },
  },
  thresholds: {
    'large_file_upload_duration': ['p(95)<30000', 'p(99)<60000'], // 30s/60s for large files
    'large_file_download_duration': ['p(95)<20000', 'p(99)<40000'],
    'large_file_success_rate': ['rate>0.90'],
    'http_req_failed': ['rate<0.1'],
  },
};

const UPLOAD_ENDPOINT = `${STORAGE_URL}/upload`;

/**
 * Generate large file data
 */
export function generateLargeFile(sizeMB) {
  const chunkSize = 1024 * 1024; // 1MB chunks
  const chunk = 'X'.repeat(chunkSize);
  const chunks = Math.ceil(sizeMB);

  let fileData = '';
  for (let i = 0; i < chunks; i++) {
    fileData += chunk;
    if (i % 10 === 0) {
      // Yield periodically to prevent overwhelming the JS heap
      sleep(0.001);
    }
  }

  // Trim to exact size
  return fileData.substring(0, sizeMB * 1024 * 1024);
}

/**
 * Upload large file
 */
export function uploadLargeFile(fileName, fileData, mimeType) {
  const boundary = `----Boundary${Math.random().toString(16).substring(2)}`;

  let body = '';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`;
  body += `Content-Type: ${mimeType}\r\n\r\n`;
  body += fileData;
  body += '\r\n';
  body += `--${boundary}--\r\n`;

  const params = {
    headers: {
      'Content-Type': `multipart/form-data; boundary=${boundary}`,
    },
    tags: { name: 'UploadLargeFile' },
    timeout: '300s',
  };

  return http.post(UPLOAD_ENDPOINT, body, params);
}

/**
 * Download large file
 */
export function downloadLargeFile(path) {
  const params = {
    headers: { 'User-Agent': 'k6-load-test/1.0' },
    tags: { name: 'DownloadLargeFile' },
    timeout: '300s',
  };

  return http.get(`${STORAGE_URL}/download/${path}`, params);
}

/**
 * Main test scenario
 */
export default function () {
  group('Large File Transfer', () => {
    // Cycle through different file sizes
    const sizeIndex = __ITER % FILE_SIZES.length;
    const sizeMB = FILE_SIZES[sizeIndex];

    const fileName = `large-file-${sizeMB}mb-${__VU}-${__ITER}.bin`;
    const mimeType = 'application/octet-stream';

    // Upload
    const uploadStart = Date.now();
    const fileData = generateLargeFile(sizeMB);
    const generatedDuration = Date.now() - uploadStart;

    const uploadResponse = uploadLargeFile(fileName, fileData, mimeType);
    const uploadDuration = Date.now() - uploadStart - generatedDuration;

    const uploadSuccess = check(uploadResponse, {
      'upload status is 201 or 200': (r) => r.status === 201 || r.status === 200,
      'upload completed': (r) => r.status < 500,
    });

    largeFileSuccessRate.add(uploadSuccess);
    largeFileUploadDuration.add(uploadDuration);

    if (uploadSuccess) {
      // Calculate upload throughput
      const mbps = (sizeMB * 8) / (uploadDuration / 1000);
      largeFileThroughput.add(mbps);
      largeFileBytesTransferred.add(sizeMB * 1024 * 1024);

      // Download the same file if we got a document ID
      try {
        const body = uploadResponse.json();
        const docId = body.document_id || body.id;
        if (docId) {
          // Wait a bit before downloading
          sleep(1);

          const downloadStart = Date.now();
          const downloadResponse = http.get(`${STORAGE_URL}/documents/${docId}`, {
            tags: { name: 'DownloadLargeFile' },
            timeout: '300s',
          });
          const downloadDuration = Date.now() - downloadStart;

          const downloadSuccess = check(downloadResponse, {
            'download status is 200': (r) => r.status === 200,
            'has content': (r) => r.body && r.body.length > 0,
          });

          largeFileSuccessRate.add(downloadSuccess);
          largeFileDownloadDuration.add(downloadDuration);

          if (downloadSuccess) {
            const downloadMbps = (sizeMB * 8) / (downloadDuration / 1000);
            largeFileThroughput.add(downloadMbps);
            largeFileBytesTransferred.add(sizeMB * 1024 * 1024);
          }
        }
      } catch (e) {
        // Response might not be JSON
      }
    }

    // Log progress periodically
    if (__ITER % 10 === 0) {
      console.log(`VU ${__VU}: Processed ${__ITER} files, current size: ${sizeMB}MB`);
    }

    // Sleep between large file operations
    sleep(Math.random() * 3 + 2);
  });
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('Large File Handling Test');
  console.log('========================================');
  console.log(`Target endpoint: ${STORAGE_URL}`);
  console.log(`File sizes: ${FILE_SIZES.join('MB, ')}MB`);
  console.log('Concurrent VUs: 20');
  console.log('Duration: 5 minutes');
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
  console.log('Large File Test Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Bytes transferred: ${(largeFileBytesTransferred.count / 1024 / 1024).toFixed(2)} MB`);
  console.log(`Success rate: ${(largeFileSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 upload duration: ${(largeFileUploadDuration.p('95') / 1000).toFixed(2)}s`);
  console.log(`P99 upload duration: ${(largeFileUploadDuration.p('99') / 1000).toFixed(2)}s`);
  console.log(`P95 download duration: ${(largeFileDownloadDuration.p('95') / 1000).toFixed(2)}s`);
  console.log(`P99 download duration: ${(largeFileDownloadDuration.p('99') / 1000).toFixed(2)}s`);
  console.log(`Average throughput: ${largeFileThroughput.avg.toFixed(2)} Mbps`);
  console.log('========================================');
}
