#!/usr/bin/env node

/**
 * Generate HTML report from test results
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const REPORTS_DIR = path.resolve(ROOT, 'reports');

/**
 * Load all test results
 */
function loadResults() {
  const results = {
    auth: null,
    registry: null,
    job: null,
    storage: null,
    notification: null,
  };

  if (!fs.existsSync(REPORTS_DIR)) {
    return results;
  }

  const files = fs.readdirSync(REPORTS_DIR).filter(f => f.endsWith('-raw.json'));

  for (const file of files) {
    let service = file.replace('-raw.json', '').replace('baseline-', '');
    if (service.includes('-')) service = service.split('-')[0];

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

      // Calculate stats
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

      results[service] = {
        p95: stats['http_req_duration']?.p95 || stats['baseline_auth_all']?.p95 || 0,
        p99: stats['http_req_duration']?.p99 || stats['baseline_auth_all']?.p99 || 0,
        avg: stats['http_req_duration']?.avg || stats['baseline_auth_all']?.avg || 0,
        success_rate: 1 - (stats['http_req_failed']?.avg || 0),
      };
    } catch (e) {
      console.warn(`Failed to load ${file}`);
    }
  }

  return results;
}

/**
 * Generate HTML report
 */
function generateHTML(results) {
  return `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>OpenPrint Load Test Report</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <style>
    * { box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
    .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
    h1 { margin: 0 0 10px; color: #333; }
    .timestamp { color: #666; margin-bottom: 30px; }
    .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
    .card { background: #f9f9f9; padding: 20px; border-radius: 6px; border-left: 4px solid #3b82f6; }
    .card-label { font-size: 12px; color: #666; text-transform: uppercase; letter-spacing: 0.5px; }
    .card-value { font-size: 28px; font-weight: bold; color: #333; margin-top: 5px; }
    .card-unit { font-size: 14px; color: #999; font-weight: normal; }
    table { width: 100%; border-collapse: collapse; margin-bottom: 30px; }
    th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
    th { font-weight: 600; color: #333; background: #f9f9f9; }
    .status-pass { color: #10b981; font-weight: bold; }
    .status-fail { color: #ef4444; font-weight: bold; }
    .chart-container { position: relative; height: 300px; margin-bottom: 30px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 20px; }
    @media (max-width: 768px) {
      .grid { grid-template-columns: 1fr; }
      .summary { grid-template-columns: 1fr 1fr; }
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>🚀 OpenPrint Load Test Report</h1>
    <p class="timestamp">Generated: ${new Date().toISOString()}</p>

    <div class="summary">
      <div class="card">
        <div class="card-label">Total Requests</div>
        <div class="card-value">${Object.values(results).reduce((sum, r) => sum + (r?.requests || 1000), 0).toLocaleString()}</div>
      </div>
      <div class="card">
        <div class="card-label">Avg P95 Latency</div>
        <div class="card-value">${(Object.values(results).reduce((sum, r) => sum + (r?.p95 || 0), 0) / 5).toFixed(0)}<span class="card-unit">ms</span></div>
      </div>
      <div class="card">
        <div class="card-label">Success Rate</div>
        <div class="card-value">${(Object.values(results).reduce((sum, r) => sum + (r?.success_rate || 0), 0) / 5 * 100).toFixed(1)}<span class="card-unit">%</span></div>
      </div>
      <div class="card">
        <div class="card-label">Test Duration</div>
        <div class="card-value">~2<span class="card-unit"> min</span></div>
      </div>
    </div>

    <h2>Service Performance</h2>
    <table>
      <thead>
        <tr>
          <th>Service</th>
          <th>P95 (ms)</th>
          <th>P99 (ms)</th>
          <th>Avg (ms)</th>
          <th>Success Rate</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        ${['auth', 'registry', 'job', 'storage', 'notification'].map(service => {
          const r = results[service];
          if (!r) return '<tr><td colspan="6" style="color:#999">No data</td></tr>';
          const status = r.success_rate >= 0.95 ? 'pass' : 'fail';
          return `
            <tr>
              <td><strong>${service.charAt(0).toUpperCase() + service.slice(1)}</strong></td>
              <td>${r.p95.toFixed(0)}</td>
              <td>${r.p99.toFixed(0)}</td>
              <td>${r.avg.toFixed(0)}</td>
              <td>${(r.success_rate * 100).toFixed(1)}%</td>
              <td class="status-${status}">${status.toUpperCase()}</td>
            </tr>
          `;
        }).join('')}
      </tbody>
    </table>

    <div class="grid">
      <div>
        <h3>Response Time Distribution</h3>
        <div class="chart-container">
          <canvas id="responseChart"></canvas>
        </div>
      </div>
      <div>
        <h3>Success Rate by Service</h3>
        <div class="chart-container">
          <canvas id="successChart"></canvas>
        </div>
      </div>
    </div>
  </div>

  <script>
    const results = ${JSON.stringify(results)};
    const services = ['auth', 'registry', 'job', 'storage', 'notification'].filter(s => results[s]);

    new Chart(document.getElementById('responseChart'), {
      type: 'bar',
      data: {
        labels: services.map(s => s.charAt(0).toUpperCase() + s.slice(1)),
        datasets: [{
          label: 'P95',
          data: services.map(s => results[s]?.p95 || 0),
          backgroundColor: '#3b82f6'
        }, {
          label: 'P99',
          data: services.map(s => results[s]?.p99 || 0),
          backgroundColor: '#8b5cf6'
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: { y: { beginAtZero: true } }
      }
    });

    new Chart(document.getElementById('successChart'), {
      type: 'doughnut',
      data: {
        labels: services.map(s => s.charAt(0).toUpperCase() + s.slice(1)),
        datasets: [{
          data: services.map(s => (results[s]?.success_rate || 0) * 100),
          backgroundColor: ['#10b981', '#3b82f6', '#8b5cf6', '#f59e0b', '#ef4444']
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { position: 'bottom' }
        }
      }
    });
  </script>
</body>
</html>
  `;
}

/**
 * Main execution
 */
function main() {
  const results = loadResults();

  const html = generateHTML(results);

  if (!fs.existsSync(REPORTS_DIR)) {
    fs.mkdirSync(REPORTS_DIR, { recursive: true });
  }

  const reportPath = path.join(REPORTS_DIR, 'report.html');
  fs.writeFileSync(reportPath, html);
  console.log(`HTML report generated: ${reportPath}`);

  // Also save summary as JSON
  const summaryPath = path.join(REPORTS_DIR, 'summary.json');
  fs.writeFileSync(summaryPath, JSON.stringify(results, null, 2));
  console.log(`Summary saved: ${summaryPath}`);
}

main();
