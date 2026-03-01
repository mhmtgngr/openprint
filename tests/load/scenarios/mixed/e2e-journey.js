/**
 * End-to-End User Journey Test
 *
 * FR-006: Simulates complete user workflows across all services
 * Covers login -> document upload -> job submission -> status checking -> logout
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { BASE_URL, STORAGE_URL, JOB_URL, TestData } from '../../lib/config.js';
import { createPrintJobData, generateFileData } from '../../lib/helpers.js';

// Custom metrics
export const e2eSuccessRate = new Rate('e2e_success_rate');
export const e2eDuration = new Trend('e2e_duration');
export const e2eSteps = {
  login: new Trend('e2e_step_login'),
  upload: new Trend('e2e_step_upload'),
  submit: new Trend('e2e_step_submit'),
  status: new Trend('e2e_step_status'),
  logout: new Trend('e2e_step_logout'),
};
export const completedJourneys = new Counter('completed_journeys');
export const failedJourneys = new Counter('failed_journeys');

// Test configuration
export const options = {
  scenarios: {
    user_journeys: {
      executor: 'constant-vus',
      vus: 50,
      duration: '5m',
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'e2e_duration': ['p(95)<10000', 'p(99)<20000'], // Full journey in 10-20 seconds
    'e2e_success_rate': ['rate>0.90'],
    'http_req_failed': ['rate<0.1'],
  },
};

const BASE_HEADERS = {
  'Content-Type': 'application/json',
  'User-Agent': 'k6-e2e-test/1.0',
};

/**
 * Step 1: Login
 */
export function stepLogin() {
  const startTime = Date.now();

  const payload = JSON.stringify({
    email: `e2e-user-${__VU}@example.com`,
    password: 'TestPassword123!',
  });

  const response = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: BASE_HEADERS,
    tags: { name: 'E2E_Login' },
  });

  const duration = Date.now() - startTime;
  e2eSteps.login.add(duration);

  if (check(response, {
    'login status is 200': (r) => r.status === 200,
    'has access_token': (r) => {
      try {
        const body = r.json();
        return body.access_token !== undefined;
      } catch (e) {
        return false;
      }
    },
  })) {
    return response.json();
  }

  return null;
}

/**
 * Step 2: Upload document
 */
export function stepUploadDocument(accessToken) {
  const startTime = Date.now();

  const fileName = `e2e-document-${__VU}-${Date.now()}.pdf`;
  const fileData = generateFileData(100); // 100KB file
  const userEmail = `e2e-user-${__VU}@example.com`;

  const boundary = `----Boundary${Math.random().toString(16).substring(2)}`;
  let body = '';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`;
  body += `Content-Type: application/pdf\r\n\r\n`;
  body += fileData;
  body += '\r\n';
  body += `--${boundary}\r\n`;
  body += `Content-Disposition: form-data; name="user_email"\r\n\r\n`;
  body += userEmail;
  body += '\r\n';
  body += `--${boundary}--\r\n`;

  const headers = {
    ...BASE_HEADERS,
    'Content-Type': `multipart/form-data; boundary=${boundary}`,
    'Authorization': `Bearer ${accessToken}`,
  };

  const response = http.post(`${STORAGE_URL}/documents`, body, {
    headers,
    tags: { name: 'E2E_Upload' },
    timeout: '30s',
  });

  const duration = Date.now() - startTime;
  e2eSteps.upload.add(duration);

  if (check(response, {
    'upload status is 201 or 200': (r) => r.status === 201 || r.status === 200,
    'has document_id': (r) => {
      try {
        const body = r.json();
        return body.document_id !== undefined || body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
  })) {
    return response.json().document_id || response.json().id;
  }

  return null;
}

/**
 * Step 3: Submit print job
 */
export function stepSubmitJob(accessToken, documentId) {
  const startTime = Date.now();

  const jobData = createPrintJobData();
  jobData.document_id = documentId;
  jobData.user_email = `e2e-user-${__VU}@example.com`;

  const payload = JSON.stringify(jobData);

  const headers = {
    ...BASE_HEADERS,
    'Authorization': `Bearer ${accessToken}`,
  };

  const response = http.post(`${JOB_URL}/jobs`, payload, {
    headers,
    tags: { name: 'E2E_SubmitJob' },
  });

  const duration = Date.now() - startTime;
  e2eSteps.submit.add(duration);

  if (check(response, {
    'submit status is 201 or 200': (r) => r.status === 201 || r.status === 200,
    'has job_id': (r) => {
      try {
        const body = r.json();
        return body.job_id !== undefined || body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
  })) {
    return response.json().job_id || response.json().id;
  }

  return null;
}

/**
 * Step 4: Check job status
 */
export function stepCheckStatus(accessToken, jobId) {
  const startTime = Date.now();

  const headers = {
    ...BASE_HEADERS,
    'Authorization': `Bearer ${accessToken}`,
  };

  const response = http.get(`${JOB_URL}/jobs/${jobId}`, {
    headers,
    tags: { name: 'E2E_CheckStatus' },
  });

  const duration = Date.now() - startTime;
  e2eSteps.status.add(duration);

  if (check(response, {
    'status query is 200 or 404': (r) => r.status === 200 || r.status === 404,
  })) {
    return true;
  }

  return false;
}

/**
 * Step 5: Logout
 */
export function stepLogout(accessToken, refreshToken) {
  const startTime = Date.now();

  const payload = JSON.stringify({
    refresh_token: refreshToken,
  });

  const headers = {
    ...BASE_HEADERS,
    'Authorization': `Bearer ${accessToken}`,
  };

  const response = http.post(`${BASE_URL}/auth/logout`, payload, {
    headers,
    tags: { name: 'E2E_Logout' },
  });

  const duration = Date.now() - startTime;
  e2eSteps.logout.add(duration);

  if (check(response, {
    'logout status is 200': (r) => r.status === 200,
  })) {
    return true;
  }

  return false;
}

/**
 * Main E2E journey scenario
 */
export default function () {
  const journeyStart = Date.now();
  let journeyComplete = false;

  group('E2E User Journey', () => {
    // Step 1: Login
    const authData = stepLogin();
    if (!authData) {
      failedJourneys.add(1);
      return;
    }

    const { access_token, refresh_token } = authData;

    // Step 2: Upload document
    const documentId = stepUploadDocument(access_token);
    if (!documentId) {
      failedJourneys.add(1);
      return;
    }

    // Step 3: Submit print job
    const jobId = stepSubmitJob(access_token, documentId);
    if (!jobId) {
      failedJourneys.add(1);
      return;
    }

    // Step 4: Check job status
    const statusOk = stepCheckStatus(access_token, jobId);
    if (!statusOk) {
      failedJourneys.add(1);
      return;
    }

    // Step 5: Logout
    stepLogout(access_token, refresh_token);

    journeyComplete = true;
    completedJourneys.add(1);

    const journeyDuration = Date.now() - journeyStart;
    e2eDuration.add(journeyDuration);
    e2eSuccessRate.add(true);

    // Think time before next journey
    sleep(Math.random() * 3 + 2);
  });

  if (!journeyComplete) {
    e2eSuccessRate.add(false);
    e2eDuration.add(Date.now() - journeyStart);
  }
}

/**
 * Setup function
 */
export function setup() {
  console.log('========================================');
  console.log('E2E User Journey Test');
  console.log('========================================');
  console.log('Journey: Login -> Upload -> Submit Job -> Check Status -> Logout');
  console.log(`Concurrent users: 50`);
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
  console.log('E2E Journey Results');
  console.log('========================================');
  console.log(`Duration: ${duration.toFixed(0)}s`);
  console.log(`Completed journeys: ${completedJourneys.count}`);
  console.log(`Failed journeys: ${failedJourneys.count}`);
  console.log(`Success rate: ${(e2eSuccessRate.rate * 100).toFixed(2)}%`);
  console.log(`P95 journey duration: ${(e2eDuration.p('95') / 1000).toFixed(2)}s`);
  console.log(`P99 journey duration: ${(e2eDuration.p('99') / 1000).toFixed(2)}s`);
  console.log('\nStep timings (P95):');
  console.log(`  Login: ${e2eSteps.login.p('95').toFixed(0)}ms`);
  console.log(`  Upload: ${e2eSteps.upload.p('95').toFixed(0)}ms`);
  console.log(`  Submit: ${e2eSteps.submit.p('95').toFixed(0)}ms`);
  console.log(`  Status: ${e2eSteps.status.p('95').toFixed(0)}ms`);
  console.log(`  Logout: ${e2eSteps.logout.p('95').toFixed(0)}ms`);
  console.log('========================================');
}
