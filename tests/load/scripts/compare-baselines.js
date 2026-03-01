#!/usr/bin/env node

/**
 * Compare current test results with stored baseline
 * FR-008: Regression Detection
 *
 * Usage: node scripts/compare-baselines.js --current <file> --baseline <file> [--service <name>] [--tolerance <0-1>]
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');

// Parse command line arguments
function parseArgs() {
  const args = process.argv.slice(2);
  const result = {
    current: null,
    baseline: null,
    service: null,
    tolerance: 0.1, // 10% tolerance
    failOnRegression: true,
  };

  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case '--current':
        result.current = args[++i];
        break;
      case '--baseline':
        result.baseline = args[++i];
        break;
      case '--service':
        result.service = args[++i];
        break;
      case '--tolerance':
        result.tolerance = parseFloat(args[++i]);
        break;
      case '--no-fail':
        result.failOnRegression = false;
        break;
    }
  }

  return result;
}

/**
 * Load and parse k6 JSON output
 */
function loadK6Results(filePath) {
  if (!fs.existsSync(filePath)) {
    console.error(`File not found: ${filePath}`);
    process.exit(1);
  }

  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n').filter(line => line.trim());

  const results = {
    metrics: {},
    thresholds: {},
  };

  for (const line of lines) {
    try {
      const data = JSON.parse(line);
      if (data.type === 'Point') {
        const metricName = data.metric;
        if (!results.metrics[metricName]) {
          results.metrics[metricName] = {
            values: [],
            data_type: data.data_type,
          };
        }
        results.metrics[metricName].values.push(data.data_value);
      } else if (data.type === 'Metric') {
        results.metrics[data.name] = {
          ...data,
          values: [],
        };
      }
    } catch (e) {
      // Skip invalid lines
    }
  }

  // Calculate statistics
  for (const [name, metric] of Object.entries(results.metrics)) {
    if (metric.values && metric.values.length > 0) {
      metric.min = Math.min(...metric.values);
      metric.max = Math.max(...metric.values);
      metric.avg = metric.values.reduce((a, b) => a + b, 0) / metric.values.length;
      metric.values.sort((a, b) => a - b);
      metric.p50 = metric.values[Math.floor(metric.values.length * 0.5)];
      metric.p90 = metric.values[Math.floor(metric.values.length * 0.9)];
      metric.p95 = metric.values[Math.floor(metric.values.length * 0.95)];
      metric.p99 = metric.values[Math.floor(metric.values.length * 0.99)];
    }
  }

  return results;
}

/**
 * Extract relevant metrics from k6 results
 */
function extractMetrics(results, service) {
  const prefixMap = {
    auth: 'http_req',
    registry: 'registry_heartbeat',
    job: 'http_req',
    storage: 'http_req',
    notification: 'http_req',
  };

  const prefix = prefixMap[service] || 'http_req';

  return {
    p95: results.metrics[`${prefix}_duration`]?.p95 || 0,
    p99: results.metrics[`${prefix}_duration`]?.p99 || 0,
    avg: results.metrics[`${prefix}_duration`]?.avg || 0,
    error_rate: results.metrics[`${prefix}_failed`]?.avg || 0,
  };
}

/**
 * Compare current metrics with baseline
 */
function compareMetrics(current, baseline, tolerance) {
  const regressions = [];
  const improvements = [];

  for (const [key, currentValue] of Object.entries(current)) {
    const baselineValue = baseline[key];
    if (baselineValue === undefined) continue;

    const change = ((currentValue - baselineValue) / baselineValue);
    const threshold = tolerance;

    if (key === 'error_rate') {
      // For error rate, increase is bad
      if (change > threshold) {
        regressions.push({ metric: key, current: currentValue, baseline: baselineValue, change });
      } else if (change < -threshold) {
        improvements.push({ metric: key, current: currentValue, baseline: baselineValue, change });
      }
    } else {
      // For latency/metrics, increase is bad
      if (change > threshold) {
        regressions.push({ metric: key, current: currentValue, baseline: baselineValue, change });
      } else if (change < -threshold) {
        improvements.push({ metric: key, current: currentValue, baseline: baselineValue, change });
      }
    }
  }

  return { regressions, improvements };
}

/**
 * Format change as percentage
 */
function formatChange(change) {
  const sign = change >= 0 ? '+' : '';
  return `${sign}${(change * 100).toFixed(2)}%`;
}

/**
 * Main execution
 */
function main() {
  const args = parseArgs();

  if (!args.current) {
    console.error('Error: --current argument is required');
    process.exit(1);
  }

  const currentPath = path.resolve(ROOT, args.current);
  const baselinePath = args.baseline
    ? path.resolve(ROOT, args.baseline)
    : path.resolve(ROOT, '.baseline/metrics.json');

  console.log('========================================');
  console.log('Performance Baseline Comparison');
  console.log('========================================');
  console.log(`Current: ${args.current}`);
  console.log(`Baseline: ${baselinePath}`);
  console.log(`Tolerance: ±${(args.tolerance * 100).toFixed(1)}%`);
  console.log('========================================');

  const currentResults = loadK6Results(currentPath);

  let baselineResults;
  if (fs.existsSync(baselinePath)) {
    baselineResults = JSON.parse(fs.readFileSync(baselinePath, 'utf8'));
  } else {
    console.warn('Baseline file not found, creating new baseline');
    baselineResults = {};
  }

  const currentMetrics = extractMetrics(currentResults, args.service);
  const baselineMetrics = baselineResults[args.service] || currentMetrics;

  const { regressions, improvements } = compareMetrics(
    currentMetrics,
    baselineMetrics,
    args.tolerance
  );

  console.log('\n--- Comparison Results ---\n');
  console.log(`Service: ${args.service || 'all'}`);
  console.log('');

  console.log('Metrics:');
  console.log('  | Metric    | Current  | Baseline | Change   |');
  console.log('  |-----------|----------|----------|----------|');
  for (const [key, value] of Object.entries(currentMetrics)) {
    const baseline = baselineMetrics[key] || value;
    const change = baseline > 0 ? ((value - baseline) / baseline) : 0;
    const status = change > args.tolerance ? '⚠️' : change < -args.tolerance ? '✅' : '➡️';
    console.log(`  | ${key.padEnd(9)} | ${value.toFixed(0).padStart(7)} | ${baseline.toFixed(0).padStart(8)} | ${formatChange(change).padStart(8)} | ${status}`);
  }

  if (regressions.length > 0) {
    console.log('\n⚠️  REGRESSIONS DETECTED:');
    for (const r of regressions) {
      console.log(`  - ${r.metric}: ${formatChange(r.change)} (current: ${r.current.toFixed(0)}, baseline: ${r.baseline.toFixed(0)})`);
    }
  }

  if (improvements.length > 0) {
    console.log('\n✅ IMPROVEMENTS:');
    for (const i of improvements) {
      console.log(`  - ${i.metric}: ${formatChange(i.change)} (current: ${i.current.toFixed(0)}, baseline: ${i.baseline.toFixed(0)})`);
    }
  }

  // Exit with error if regressions detected
  if (regressions.length > 0 && args.failOnRegression) {
    console.log('\n❌ Performance regression detected!');
    process.exit(1);
  }

  console.log('\n✅ No significant regressions detected');
}

main();
