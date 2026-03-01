/**
 * HTTP client wrapper with auth handling
 *
 * This module provides a wrapper around HTTP operations with automatic
 * authentication, retry logic, and response validation.
 */

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, API_PATHS, TEST_CREDENTIALS } from './config.js';
import { isSuccess, getJson, retry, TokenManager } from './helpers.js';

/**
 * HTTP client with authentication support
 */
export class ApiClient {
  constructor(options = {}) {
    this.baseURL = options.baseURL || BASE_URL;
    this.timeout = options.timeout || 30000; // 30 seconds default
    this.tokenManager = null;
    this.defaultHeaders = {
      'Content-Type': 'application/json',
      'User-Agent': 'k6-load-test/1.0',
    };
  }

  /**
   * Set authentication token manager
   */
  setTokenManager(manager) {
    this.tokenManager = manager;
  }

  /**
   * Get request headers including auth if available
   */
  getHeaders(additional = {}) {
    const headers = { ...this.defaultHeaders, ...additional };

    if (this.tokenManager) {
      const authHeaders = this.tokenManager.getAuthHeaders();
      Object.assign(headers, authHeaders);
    }

    return headers;
  }

  /**
   * Build full URL from path
   */
  buildURL(path) {
    if (path.startsWith('http')) {
      return path;
    }
    return `${this.baseURL}${path}`;
  }

  /**
   * Perform GET request
   */
  get(path, params = {}) {
    const url = this.buildURL(path);
    const headers = this.getHeaders();

    const options = {
      headers,
      timeout: this.timeout,
      tags: { name: path },
    };

    if (Object.keys(params).length > 0) {
      const queryString = new URLSearchParams(params).toString();
      url.search = queryString;
    }

    return http.get(url, options);
  }

  /**
   * Perform POST request
   */
  post(path, data = {}) {
    const url = this.buildURL(path);
    const headers = this.getHeaders();

    const options = {
      headers,
      timeout: this.timeout,
      tags: { name: path },
    };

    return http.post(url, JSON.stringify(data), options);
  }

  /**
   * Perform PUT request
   */
  put(path, data = {}) {
    const url = this.buildURL(path);
    const headers = this.getHeaders();

    const options = {
      headers,
      timeout: this.timeout,
      tags: { name: path },
    };

    return http.put(url, JSON.stringify(data), options);
  }

  /**
   * Perform PATCH request
   */
  patch(path, data = {}) {
    const url = this.buildURL(path);
    const headers = this.getHeaders();

    const options = {
      headers,
      timeout: this.timeout,
      tags: { name: path },
    };

    const http.patch(url, JSON.stringify(data), options);
  }

  /**
   * Perform DELETE request
   */
  delete(path) {
    const url = this.buildURL(path);
    const headers = this.getHeaders();

    const options = {
      headers,
      timeout: this.timeout,
      tags: { name: path },
    };

    return http.request('DELETE', url, null, options);
  }

  /**
   * Upload file with multipart/form-data
   */
  upload(path, fileData, fileName, mimeType, additionalFields = {}) {
    const url = this.buildURL(path);
    const headers = this.getHeaders({ 'Content-Type': 'multipart/form-data' });

    // Build multipart body
    const boundary = `----WebKitFormBoundary${Math.random().toString(16).substring(2)}`;
    let body = '';

    // Add file
    body += `--${boundary}\r\n`;
    body += `Content-Disposition: form-data; name="file"; filename="${fileName}"\r\n`;
    body += `Content-Type: ${mimeType}\r\n\r\n`;
    body += fileData;
    body += '\r\n';

    // Add additional fields
    for (const [key, value] of Object.entries(additionalFields)) {
      body += `--${boundary}\r\n`;
      body += `Content-Disposition: form-data; name="${key}"\r\n\r\n`;
      body += value;
      body += '\r\n';
    }

    body += `--${boundary}--\r\n`;

    headers['Content-Type'] = `multipart/form-data; boundary=${boundary}`;

    const options = {
      headers,
      timeout: this.timeout * 3, // Longer timeout for uploads
      tags: { name: path },
    };

    return http.post(url, body, options);
  }
}

/**
 * Auth service client
 */
export class AuthClient extends ApiClient {
  constructor(options = {}) {
    super({ ...options, baseURL: options.baseURL || BASE_URL });
  }

  /**
   * User login
   */
  login(email, password) {
    return this.post(API_PATHS.AUTH_LOGIN, {
      email: email || TEST_CREDENTIALS.email,
      password: password || TEST_CREDENTIALS.password,
    });
  }

  /**
   * User registration
   */
  register(email, password, firstName, lastName) {
    return this.post(API_PATHS.AUTH_REGISTER, {
      email,
      password,
      first_name: firstName,
      last_name: lastName,
    });
  }

  /**
   * Token refresh
   */
  refreshToken(refreshToken) {
    return this.post(API_PATHS.AUTH_REFRESH, {
      refresh_token: refreshToken,
    });
  }

  /**
   * Logout
   */
  logout(refreshToken) {
    return this.post(API_PATHS.AUTH_LOGOUT, {
      refresh_token: refreshToken,
    });
  }

  /**
   * Get user profile
   */
  getProfile() {
    return this.get(API_PATHS.AUTH_PROFILE);
  }
}

/**
 * Registry service client
 */
export class RegistryClient extends ApiClient {
  constructor(options = {}) {
    super({ ...options, baseURL: options.baseURL || __ENV.REGISTRY_URL || 'http://localhost:8002' });
  }

  /**
   * List agents
   */
  listAgents(limit = 50, offset = 0) {
    return this.get(API_PATHS.AGENTS, { limit, offset });
  }

  /**
   * Get agent details
   */
  getAgent(agentId) {
    return this.get(`${API_PATHS.AGENTS}/${agentId}`);
  }

  /**
   * Send agent heartbeat
   */
  sendHeartbeat(agentId, data) {
    const path = API_PATHS.AGENT_HEARTBEAT(agentId);
    return this.post(path, data);
  }

  /**
   * Batch heartbeat for multiple agents
   */
  batchHeartbeat(heartbeats) {
    return this.post('/agents/heartbeat/batch', { heartbeats });
  }

  /**
   * Register agent
   */
  registerAgent(name, agentType = 'desktop') {
    return this.post(API_PATHS.AGENT_REGISTER, {
      name,
      type: agentType,
      version: '2.0.0',
    });
  }

  /**
   * List discovered printers
   */
  listDiscoveredPrinters(agentId) {
    return this.get(`/agents/${agentId}/discovered-printers`);
  }

  /**
   * List printers
   */
  listPrinters(limit = 50, offset = 0) {
    return this.get(API_PATHS.PRINTERS, { limit, offset });
  }

  /**
   * Get printer details
   */
  getPrinter(printerId) {
    return this.get(`${API_PATHS.PRINTERS}/${printerId}`);
  }
}

/**
 * Job service client
 */
export class JobClient extends ApiClient {
  constructor(options = {}) {
    super({ ...options, baseURL: options.baseURL || __ENV.JOB_URL || 'http://localhost:8003' });
  }

  /**
   * Submit print job
   */
  submitJob(jobData) {
    return this.post(API_PATHS.JOBS, jobData);
  }

  /**
   * Get job status
   */
  getJob(jobId) {
    return this.get(`${API_PATHS.JOBS}/${jobId}`);
  }

  /**
   * List jobs
   */
  listJobs(limit = 50, offset = 0, status = null, printerId = null) {
    const params = { limit, offset };
    if (status) params.status = status;
    if (printerId) params.printer_id = printerId;
    return this.get(API_PATHS.JOBS, params);
  }

  /**
   * Cancel job
   */
  cancelJob(jobId) {
    return this.delete(`${API_PATHS.JOBS}/${jobId}`);
  }

  /**
   * Retry failed job
   */
  retryJob(jobId) {
    return this.post(`${API_PATHS.JOBS}/${jobId}/retry`, {});
  }

  /**
   * Pause job
   */
  pauseJob(jobId) {
    return this.post(`${API_PATHS.JOBS}/${jobId}/pause`, {});
  }

  /**
   * Resume job
   */
  resumeJob(jobId) {
    return this.post(`${API_PATHS.JOBS}/${jobId}/resume`, {});
  }

  /**
   * Get job history
   */
  getJobHistory(jobId) {
    return this.get(API_PATHS.JOB_HISTORY, { job_id: jobId });
  }

  /**
   * Get queue statistics
   */
  getQueueStats() {
    return this.get(API_PATHS.JOB_QUEUE_STATS);
  }

  /**
   * Agent poll for jobs
   */
  agentPoll(agentId) {
    return this.get(`/agents/${agentId}/poll`);
  }

  /**
   * Update job status (agent endpoint)
   */
  updateJobStatus(jobId, status, message = '', pages = 0) {
    return this.put(`${API_PATHS.JOBS}/${jobId}/status`, {
      status,
      message,
      pages,
    });
  }
}

/**
 * Storage service client
 */
export class StorageClient extends ApiClient {
  constructor(options = {}) {
    super({ ...options, baseURL: options.baseURL || __ENV.STORAGE_URL || 'http://localhost:8004' });
  }

  /**
   * Upload document
   */
  uploadDocument(fileName, fileData, contentType, userEmail) {
    return this.upload(API_PATHS.DOCUMENTS, fileData, fileName, contentType, {
      user_email: userEmail,
    });
  }

  /**
   * Get document metadata
   */
  getDocumentMetadata(documentId) {
    return this.get(`${API_PATHS.DOCUMENT(documentId)}/metadata`);
  }

  /**
   * Download document
   */
  downloadDocument(documentId) {
    return this.get(API_PATHS.DOCUMENT(documentId));
  }

  /**
   * Delete document
   */
  deleteDocument(documentId) {
    return this.delete(API_PATHS.DOCUMENT(documentId));
  }

  /**
   * List documents
   */
  listDocuments(userEmail = null, limit = 50, offset = 0) {
    const params = { limit, offset };
    if (userEmail) params.user_email = userEmail;
    return this.get(API_PATHS.DOCUMENTS, params);
  }

  /**
   * Generic upload endpoint
   */
  upload(fileName, fileData, contentType) {
    return this.upload(API_PATHS.UPLOAD, fileData, fileName, contentType, {});
  }

  /**
   * Generic download endpoint
   */
  download(path) {
    return this.get(API_PATHS.DOWNLOAD(path));
  }
}

/**
 * Notification service client
 */
export class NotificationClient extends ApiClient {
  constructor(options = {}) {
    super({ ...options, baseURL: options.baseURL || __ENV.NOTIFICATION_URL || 'http://localhost:8005' });
  }

  /**
   * Get WebSocket connection URL
   */
  getWebSocketURL(userId, orgId = null) {
    const url = new URL(this.buildURL(API_PATHS.WS_CONNECT));
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    url.searchParams.set('user_id', userId);
    if (orgId) {
      url.searchParams.set('org_id', orgId);
    }
    return url.toString();
  }

  /**
   * Broadcast message
   */
  broadcast(type, data, userId = null, orgId = null) {
    const payload = { type, data };
    if (userId) payload.user_id = userId;
    if (orgId) payload.org_id = orgId;

    return this.post(API_PATHS.BROADCAST, payload);
  }

  /**
   * Get connection statistics
   */
  getConnectionStats(userId = null, orgId = null) {
    const params = {};
    if (userId) params.user_id = userId;
    if (orgId) params.org_id = orgId;

    return this.get(API_PATHS.CONNECTIONS, params);
  }
}

/**
 * Create authenticated client instances
 */
export function createAuthenticatedClients() {
  const authClient = new AuthClient();

  // Create token manager
  const tokenManager = new TokenManager(
    () => authClient.login(),
    (token) => authClient.refreshToken(token)
  );

  // Create clients with auth
  const registryClient = new RegistryClient();
  const jobClient = new JobClient();
  const storageClient = new StorageClient();

  registryClient.setTokenManager(tokenManager);
  jobClient.setTokenManager(tokenManager);
  storageClient.setTokenManager(tokenManager);

  return {
    auth: authClient,
    registry: registryClient,
    job: jobClient,
    storage: storageClient,
    tokenManager,
  };
}
