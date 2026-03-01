#!/usr/bin/env node

/**
 * Generate baseline metrics from test results
 * Creates .baseline/metrics.json for regression detection
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const REPORTS_DIR = path.resolve(ROOT, 'reports');
const BASELINE_DIR = path.resolve(ROOT, '.baseline');

/**
 * Load all JSON report files
 */
function loadReports() {
  const reports = {};

  if (!fs.existsSync(REPORTS_DIR)) {
    console.warn('Reports directory not found');
    return reports;
  }

  const files = fs.readdirSync(REPORTS_DIR).filter(f => f.endsWith('-raw.json'));

  for (const file of files) {
    const service = file.replace('-raw.json', '');
    const filePath = path.join(REPORTS_DIR, file);

    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const lines = content.split('\n').filter(line => line.trim());

      const metrics = {};
      for (const line of lines) {
        try {
          const data = JSON.parse(line);
          if (data.type === 'Point') {
            const name = data.metric;
            if (!metrics[name]) metrics[name] = [];
            metrics[name].push(data.data_value);
          }
        } catch (e) {}
      }

      // Calculate statistics
      const stats = {};
      for (const [name, values] of Object.entries(metrics)) {
        if (values.length > 0) {
          values.sort((a, b) => a - b);
          stats[name] = {
            min: Math.min(...values),
            max: Math.max(...values),
            avg: values.reduce((a, b) => a + b, 0) / values.length,
            p50: values[Math.floor(values.length * 0.5)],
            p95: values[Math.floor(values.length * 0.95)],
            p99: values[Math.floor(values.length * 0.99)],
          };
        }
      }

      // Extract key metrics
      reports[service] = {
        p95: stats['http_req_duration']?.p95 || 0,
        p99: stats['http_req_duration']?.p99 || 0,
        avg: stats['http_req_duration']?.avg || 0,
        error_rate: stats['http_req_failed']?.avg || 0,
      };
    } catch (e) {
      console.warn(`Failed to parse ${file}:`, e.message);
    }
  }

  return reports;
}

/**
 * Save baseline to file
 */
function saveBaseline(baseline) {
  if (!fs.existsSync(BASELINE_DIR)) {
    fs.mkdirSync(BASELINE_DIR, { recursive: true });
  }

  const baselinePath = path.join(BASELINE_DIR, 'metrics.json');
  fs.writeFileSync(baselinePath, JSON.stringify(baseline, null, 2));
  console.log(`Baseline saved to ${baselinePath}`);
}

/**
 * Main execution
 */
function main() {
  console.log('Generating performance baseline...');

  const reports = loadReports();

  // Organize by service
  const baseline = {
    timestamp: new Date().toISOString(),
    commit: process.env.GIT_SHA || 'unknown',
    branch: process.env.GIT_BRANCH || 'unknown',
    services: {},
  };

  // Map report keys to services
  const serviceMap = {
    'auth': 'auth',
    'auth-baseline': 'auth',
    'registry': 'registry',
    'registry-baseline': 'registry',
    'job': 'job',
    'job-baseline': 'job',
    'storage': 'storage',
    'storage-baseline': 'storage',
    'notification': 'notification',
    'notification-baseline': 'notification',
  };

  for (const [key, data] of Object.entries(reports)) {
    const service = serviceMap[key] || key;
    baseline.services[service] = data;
  }

  // Create baseline structure
  const outputBaseline = {
    auth: baseline.services.auth || { p95: 0, p99: 0, avg: 0, error_rate: 0 },
    registry: baseline.services.registry || { p95: 0, p99: 0, avg: 0, error_rate: 0 },
    job: baseline.services.job || { p95: 0, p99: 0, avg: 0, error_rate: 0 },
    storage: baseline.services.storage || { p95: 0, p99: 0, avg: 0, error_rate: 0 },
    notification: baseline.services.notification || { p95: 0, p99: 0, avg: 0, error_rate: 0 },
    metadata: {
      timestamp: baseline.timestamp,
      commit: baseline.commit,
      branch: baseline.branch,
    },
  };

  saveBaseline(outputBaseline);

  console.log('\nBaseline Summary:');
  console.log('  | Service     | P95 (ms) | P99 (ms) | Error Rate |');
  console.log('  |-------------|----------|----------|------------|');
  for (const [service, data] of Object.entries(outputBaseline)) {
    if (data.metadata) continue;
    console.log(`  | ${service.padEnd(11)} | ${data.p95.toFixed(0).padStart(8)} | ${data.p99.toFixed(0).padStart(8)} | ${(data.error_rate * 100).toFixed(2)}%`.padEnd(12) + ' |');
  }
}

main();
