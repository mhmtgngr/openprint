/**
 * Reusable helper functions for all k6 tests
 *
 * This module provides common utility functions used across test scenarios.
 */

import { check, sleep } from 'k6';
import { TestData } from './config.js';

/**
 * Sleep for a random duration between min and max seconds
 * @param {number} min - Minimum sleep duration in seconds
 * @param {number} max - Maximum sleep duration in seconds
 */
export function randomSleep(min, max) {
  const duration = Math.random() * (max - min) + min;
  sleep(duration);
}

/**
 * Generate a random integer between min and max (inclusive)
 * @param {number} min - Minimum value
 * @param {number} max - Maximum value
 * @returns {number}
 */
export function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * Pick a random item from an array
 * @param {Array} array - Array to pick from
 * @returns {*} Random item from array
 */
export function randomChoice(array) {
  return array[Math.floor(Math.random() * array.length)];
}

/**
 * Check if response is successful
 * @param {object} response - k6 HTTP response object
 * @param {string} context - Context for error messages
 * @returns {boolean}
 */
export function isSuccess(response, context = 'Request') {
  const success = check(response, {
    [`${context} status is 2xx`]: (r) => r.status >= 200 && r.status < 300,
    [`${context} has body`]: (r) => r.body !== undefined && r.body.length > 0,
  });

  if (!success && response.status >= 400) {
    console.error(`${context} failed: ${response.status} ${response.status_text}`);
  }

  return success;
}

/**
 * Extract JSON from response with error handling
 * @param {object} response - k6 HTTP response object
 * @returns {object|null} Parsed JSON or null
 */
export function getJson(response) {
  try {
    return response.json();
  } catch (e) {
    console.error('Failed to parse JSON:', e);
    return null;
  }
}

/**
 * Create a basic auth header
 * @param {string} username
 * @param {string} password
 * @returns {string} Base64 encoded auth header
 */
export function basicAuth(username, password) {
  return `Basic ${btoa(`${username}:${password}`)}`;
}

/**
 * Parse JWT token (decode base64)
 * @param {string} token - JWT token
 * @returns {object} Decoded payload
 */
export function parseJWT(token) {
  const parts = token.split('.');
  if (parts.length !== 3) {
    throw new Error('Invalid JWT format');
  }

  const payload = parts[1];
  const decoded = atob(payload);
  return JSON.parse(decoded);
}

/**
 * Check if JWT token is expired
 * @param {string} token - JWT token
 * @returns {boolean} True if expired
 */
export function isTokenExpired(token) {
  try {
    const payload = parseJWT(token);
    const now = Math.floor(Date.now() / 1000);
    return payload.exp < now;
  } catch (e) {
    return true;
  }
}

/**
 * Get token expiration time
 * @param {string} token - JWT token
 * @returns {number} Expiration timestamp
 */
export function getTokenExpiration(token) {
  try {
    const payload = parseJWT(token);
    return payload.exp;
  } catch (e) {
    return 0;
  }
}

/**
 * Rate limiter - tracks request rate per virtual user
 */
export class RateLimiter {
  constructor(requestsPerSecond) {
    this.requestsPerSecond = requestsPerSecond;
    this.lastRequest = 0;
  }

  wait() {
    const now = Date.now();
    const elapsed = now - this.lastRequest;
    const interval = 1000 / this.requestsPerSecond;

    if (elapsed < interval) {
      sleep((interval - elapsed) / 1000);
    }

    this.lastRequest = Date.now();
  }
}

/**
 * Token manager - handles JWT refresh logic
 */
export class TokenManager {
  constructor(loginFn, refreshFn) {
    this.accessToken = null;
    this.refreshToken = null;
    this.loginFn = loginFn;
    this.refreshFn = refreshFn;
  }

  login() {
    const response = this.loginFn();
    if (isSuccess(response, 'Login')) {
      const data = getJson(response);
      if (data && data.access_token) {
        this.accessToken = data.access_token;
        this.refreshToken = data.refresh_token;
        return true;
      }
    }
    return false;
  }

  getAccessToken() {
    if (!this.accessToken || isTokenExpired(this.accessToken)) {
      this.refresh();
    }
    return this.accessToken;
  }

  refresh() {
    if (this.refreshToken && this.refreshFn) {
      const response = this.refreshFn(this.refreshToken);
      if (isSuccess(response, 'Token refresh')) {
        const data = getJson(response);
        if (data && data.access_token) {
          this.accessToken = data.access_token;
          if (data.refresh_token) {
            this.refreshToken = data.refresh_token;
          }
          return true;
        }
      }
    }
    // Fallback to login
    return this.login();
  }

  getAuthHeaders() {
    const token = this.getAccessToken();
    if (!token) {
      return {};
    }
    return {
      'Authorization': `Bearer ${token}`,
    };
  }
}

/**
 * Counter for tracking test iterations
 */
export class Counter {
  constructor(initial = 0) {
    this.count = initial;
  }

  increment() {
    return ++this.count;
  }

  get() {
    return this.count;
  }

  reset() {
    this.count = 0;
  }
}

/**
 * Weighted selector for choosing from options with probabilities
 */
export class WeightedSelector {
  constructor(options) {
    // options is an array of {value, weight}
    this.options = options;
    this.totalWeight = options.reduce((sum, opt) => sum + opt.weight, 0);
  }

  select() {
    let random = Math.random() * this.totalWeight;
    for (const option of this.options) {
      random -= option.weight;
      if (random <= 0) {
        return option.value;
      }
    }
    return this.options[0].value;
  }
}

/**
 * Create realistic print job data
 */
export function createPrintJobData() {
  return {
    document_id: TestData.printerId(),
    printer_id: TestData.printerId(),
    user_name: TestData.userName(),
    user_email: TestData.email(),
    title: TestData.documentTitle(),
    copies: randomInt(1, 5),
    color_mode: randomChoice(['color', 'monochrome']),
    duplex: Math.random() > 0.5,
    media_type: randomChoice(['a4', 'letter', 'legal', 'a3']),
    quality: randomChoice(['draft', 'normal', 'high']),
    pages: randomInt(1, 100),
  };
}

/**
 * Create heartbeat request data
 */
export function createHeartbeatData(agentId) {
  return {
    agent_id: agentId,
    status: randomChoice(['online', 'processing', 'idle']),
    printer_count: randomInt(0, 5),
    completed_jobs: randomInt(0, 100),
    failed_jobs: randomInt(0, 10),
    version: '2.0.0',
  };
}

/**
 * Create registration request data
 */
export function createRegistrationData() {
  return {
    email: TestData.email(),
    password: TestData.password(),
    first_name: `Test${randomInt(1, 1000)}`,
    last_name: `User${randomInt(1, 1000)}`,
  };
}

/**
 * Measure and log custom timing metric
 */
export function measureTime(name, fn) {
  const start = new Date();
  const result = fn();
  const end = new Date();
  const duration = end - start;

  // Create a custom metric entry if metrics module is available
  if (typeof metrics !== 'undefined' && metrics.customTimings) {
    metrics.customTimings.add(name, duration);
  }

  return result;
}

/**
 * Batch processor for bulk operations
 */
export class BatchProcessor {
  constructor(batchSize = 100) {
    this.batchSize = batchSize;
    this.currentBatch = [];
  }

  add(item) {
    this.currentBatch.push(item);
    if (this.currentBatch.length >= this.batchSize) {
      return this.flush();
    }
    return false;
  }

  flush() {
    if (this.currentBatch.length === 0) {
      return [];
    }
    const batch = [...this.currentBatch];
    this.currentBatch = [];
    return batch;
  }

  hasPending() {
    return this.currentBatch.length > 0;
  }
}

/**
 * Retry wrapper for operations
 */
export function retry(fn, maxAttempts = 3, delayMs = 1000) {
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const result = fn();
      if (result && result.status && result.status >= 200 && result.status < 300) {
        return result;
      }
      if (attempt < maxAttempts) {
        sleep(delayMs / 1000);
      }
    } catch (e) {
      console.error(`Attempt ${attempt} failed:`, e);
      if (attempt < maxAttempts) {
        sleep(delayMs / 1000);
      }
    }
  }
  return null;
}

/**
 * Progress logger for long-running tests
 */
export class ProgressLogger {
  constructor(interval = 10) {
    this.interval = interval;
    this.lastLog = 0;
    this.counters = {};
  }

  increment(key) {
    if (!this.counters[key]) {
      this.counters[key] = 0;
    }
    this.counters[key]++;
    this.maybeLog();
  }

  maybeLog() {
    const now = Math.floor(Date.now() / 1000);
    if (now - this.lastLog >= this.interval) {
      this.log();
      this.lastLog = now;
    }
  }

  log() {
    console.log(`Progress: ${JSON.stringify(this.counters)}`);
  }
}
