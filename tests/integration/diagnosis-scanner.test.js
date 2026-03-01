#!/usr/bin/env node
/**
 * diagnosis-scanner.test.js - E2E/Integration tests for dev.sh diagnosis scanner
 *
 * Tests the diagnosis scanner to ensure accurate file enumeration:
 * - Correct TSX/TS file counts in diagnosis.json
 * - Proper exclusion of build artifacts
 * - Feature module breakdown accuracy
 *
 * Usage:
 *   node tests/integration/diagnosis-scanner.test.js
 *   npm run test:diagnosis
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

const REPO_DIR = path.resolve(__dirname, '../..');
const ARTIFACTS_DIR = path.join(REPO_DIR, '.team', 'artifacts');
const DEV_SH = path.join(REPO_DIR, 'dev.sh');
const FRONTEND_SRC_DIR = path.join(REPO_DIR, 'web', 'dashboard', 'src');

// Test state
let testsRun = 0;
let testsPassed = 0;
let testsFailed = 0;
const failures = [];

// Colors for terminal output
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
};

function log(color, ...args) {
  console.log(color + args.join(' ') + colors.reset);
}

function info(...args) {
  log(colors.blue, '[INFO]', ...args);
}

function success(...args) {
  log(colors.green, '[PASS]', ...args);
}

function error(...args) {
  log(colors.red, '[FAIL]', ...args);
}

function warn(...args) {
  log(colors.yellow, '[WARN]', ...args);
}

// Helper functions

function execCommand(command, options = {}) {
  const defaultOptions = {
    cwd: REPO_DIR,
    encoding: 'utf-8',
    stdio: 'pipe',
    ...options,
  };
  try {
    return execSync(command, defaultOptions);
  } catch (err) {
    return null;
  }
}

function sourceAndRunScan(dir) {
  const command = `bash -c 'source ${DEV_SH} 2>/dev/null && scan_frontend_files "${dir}"'`;
  const output = execCommand(command);
  if (!output) {
    throw new Error(`Failed to run scan for ${dir}`);
  }
  // Extract JSON from output - look for lines starting with '{'
  const lines = output.split('\n');
  for (const line of lines) {
    if (line.trim().startsWith('{')) {
      return JSON.parse(line);
    }
  }
  throw new Error(`No JSON found in output for ${dir}`);
}

function runDiagnose() {
  // First check if diagnosis.json already exists from a recent run
  const diagnosisPath = path.join(ARTIFACTS_DIR, 'diagnosis.json');

  if (fs.existsSync(diagnosisPath)) {
    // Check if it's recent (less than 10 minutes old)
    const stats = fs.statSync(diagnosisPath);
    const age = Date.now() - stats.mtime.getTime();
    if (age < 10 * 60 * 1000) { // 10 minutes
      const content = fs.readFileSync(diagnosisPath, 'utf-8');
      return JSON.parse(content);
    }
  }

  // Run the diagnose command with a timeout
  try {
    // Run in background with timeout
    execCommand(`timeout 120 bash ${DEV_SH} diagnose 2>/dev/null || true`);
  } catch (e) {
    // If timeout, try to use existing artifact
  }

  // Read the diagnosis.json artifact
  if (!fs.existsSync(diagnosisPath)) {
    throw new Error(`Diagnosis artifact not found at ${diagnosisPath}`);
  }

  const content = fs.readFileSync(diagnosisPath, 'utf-8');
  return JSON.parse(content);
}

function countFilesByExtension(dir, extension, excludeDirs = []) {
  let count = 0;

  function walkDirectory(currentPath) {
    let entries;
    try {
      entries = fs.readdirSync(currentPath, { withFileTypes: true });
    } catch (e) {
      return;
    }

    for (const entry of entries) {
      const fullPath = path.join(currentPath, entry.name);

      if (entry.isDirectory()) {
        // Skip node_modules, dist, build
        if (['node_modules', 'dist', 'build'].includes(entry.name)) {
          continue;
        }
        walkDirectory(fullPath);
      } else if (entry.isFile()) {
        // Match exact extension to avoid .tsx matching .ts
        const isMatch = extension === '.tsx'
          ? entry.name.endsWith('.tsx')
          : entry.name.endsWith('.ts') && !entry.name.endsWith('.tsx');
        if (isMatch) {
          count++;
        }
      }
    }
  }

  walkDirectory(dir);
  return count;
}

function assertEqual(actual, expected, message) {
  testsRun++;
  if (actual === expected) {
    testsPassed++;
    success(message);
    return true;
  } else {
    testsFailed++;
    const msg = `${message}\n  Expected: ${expected}\n  Actual: ${actual}`;
    error(msg);
    failures.push(msg);
    return false;
  }
}

function assertTrue(value, message) {
  testsRun++;
  if (value) {
    testsPassed++;
    success(message);
    return true;
  } else {
    testsFailed++;
    error(message);
    failures.push(message);
    return false;
  }
}

function assertHasProperty(obj, prop, message) {
  testsRun++;
  if (obj && obj.hasOwnProperty(prop)) {
    testsPassed++;
    success(message);
    return true;
  } else {
    testsFailed++;
    error(message);
    failures.push(message);
    return false;
  }
}

// Test cleanup helpers
const tempDirs = [];

function createTempDir(prefix) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  tempDirs.push(dir);
  return dir;
}

function cleanup() {
  for (const dir of tempDirs) {
    try {
      fs.rmSync(dir, { recursive: true, force: true });
    } catch (e) {
      // Ignore cleanup errors
    }
  }
}

// Test suites

function testScanFrontendFilesReturnsValidJson() {
  info('Test: scan_frontend_files returns valid JSON');
  const result = sourceAndRunScan(FRONTEND_SRC_DIR);

  assertHasProperty(result, 'tsx_count', '  Has tsx_count property');
  assertHasProperty(result, 'ts_count', '  Has ts_count property');
  assertHasProperty(result, 'total_count', '  Has total_count property');
  assertHasProperty(result, 'test_count', '  Has test_count property');
  assertHasProperty(result, 'features', '  Has features property');
}

function testTsxCountReturns80() {
  info('Test: TSX count returns 80');
  const result = sourceAndRunScan(FRONTEND_SRC_DIR);
  assertEqual(result.tsx_count, 80, '  TSX file count should be 80');
}

function testTsCountReturns33() {
  info('Test: TS count returns 33');
  const result = sourceAndRunScan(FRONTEND_SRC_DIR);
  assertEqual(result.ts_count, 33, '  TS file count should be 33');
}

function testTotalCountReturns113() {
  info('Test: Total count returns 113');
  const result = sourceAndRunScan(FRONTEND_SRC_DIR);
  assertEqual(result.total_count, 113, '  Total TS/TSX file count should be 113');
}

function testExcludesNodeModules() {
  info('Test: Excludes node_modules directory');
  const tempDir = createTempDir('scanner-test-');
  const srcDir = path.join(tempDir, 'src');
  const nodeModulesDir = path.join(tempDir, 'node_modules');

  fs.mkdirSync(srcDir);
  fs.mkdirSync(nodeModulesDir);
  fs.writeFileSync(path.join(srcDir, 'App.tsx'), '// App component');
  fs.writeFileSync(path.join(nodeModulesDir, 'lib.tsx'), '// Library file');

  try {
    const result = sourceAndRunScan(srcDir);
    assertEqual(result.tsx_count, 1, '  Should only count files outside node_modules');
  } catch (e) {
    error(`  Error: ${e.message}`);
    testsFailed++;
  }
}

function testExcludesDistAndBuild() {
  info('Test: Excludes dist and build directories');
  const tempDir = createTempDir('scanner-test-');
  const srcDir = path.join(tempDir, 'src');
  const distDir = path.join(tempDir, 'dist');
  const buildDir = path.join(tempDir, 'build');

  fs.mkdirSync(srcDir);
  fs.mkdirSync(distDir);
  fs.mkdirSync(buildDir);
  fs.writeFileSync(path.join(srcDir, 'App.tsx'), '// App component');
  fs.writeFileSync(path.join(distDir, 'bundle.ts'), '// Built bundle');
  fs.writeFileSync(path.join(buildDir, 'output.ts'), '// Build output');

  try {
    const result = sourceAndRunScan(srcDir);
    assertEqual(result.total_count, 1, '  Should only count files in src, not in dist or build');
  } catch (e) {
    error(`  Error: ${e.message}`);
    testsFailed++;
  }
}

function testNonexistentDirectoryReturnsZeros() {
  info('Test: Nonexistent directory returns zeros');
  const result = sourceAndRunScan('/nonexistent/path/that/does/not/exist');

  assertEqual(result.tsx_count, 0, '  Nonexistent directory should return 0 TSX files');
  assertEqual(result.ts_count, 0, '  Nonexistent directory should return 0 TS files');
  assertEqual(result.total_count, 0, '  Nonexistent directory should return 0 total files');
}

function testFeatureBreakdownAccuracy() {
  info('Test: Feature breakdown includes known features');
  const result = sourceAndRunScan(FRONTEND_SRC_DIR);

  const expectedFeatures = ['dashboard', 'documents', 'printers', 'auth', 'agents', 'settings', 'jobs', 'devices'];

  for (const feature of expectedFeatures) {
    assertTrue(
      result.features && result.features.hasOwnProperty(feature),
      `  Feature '${feature}' should be in breakdown`
    );
  }
}

function testNotFollowSymlinks() {
  info('Test: Does not follow symlinks outside directory');
  const tempDir = createTempDir('scanner-symlink-');
  const outsideDir = createTempDir('scanner-outside-');
  const srcDir = path.join(tempDir, 'src');

  fs.mkdirSync(srcDir);

  // Create a symlink to outside directory
  const linkPath = path.join(srcDir, 'external-link');
  fs.symlinkSync(outsideDir, linkPath);

  // Create files in both locations
  fs.writeFileSync(path.join(srcDir, 'inside.tsx'), '// Inside file');
  fs.writeFileSync(path.join(outsideDir, 'outside.tsx'), '// Outside file');

  try {
    const result = sourceAndRunScan(srcDir);
    assertEqual(result.tsx_count, 1, '  Should only count files inside src, not symlinked files');
  } catch (e) {
    error(`  Error: ${e.message}`);
    testsFailed++;
  }
}

function testDiagnoseCreatesArtifact() {
  info('Test: diagnose_project creates diagnosis.json artifact');

  try {
    const result = runDiagnose();

    assertHasProperty(result, 'project', '  Has project property');
    assertHasProperty(result.project, 'tsx_files', '  Has tsx_files property');
    assertHasProperty(result.project, 'go_files', '  Has go_files property');
  } catch (e) {
    error(`  Error: ${e.message}`);
    testsFailed++;
  }
}

function testDiagnosisJsonHasCorrectCounts() {
  info('Test: diagnosis.json has correct tsx_files count');

  try {
    const result = runDiagnose();
    assertEqual(result.project.tsx_files, 113, '  tsx_files should be 113');
  } catch (e) {
    error(`  Error: ${e.message}`);
    testsFailed++;
  }
}

function testFilesystemCountMatchesScan() {
  info('Test: Scan count matches actual filesystem count');

  const actualTsxCount = countFilesByExtension(FRONTEND_SRC_DIR, '.tsx');
  const actualTsCount = countFilesByExtension(FRONTEND_SRC_DIR, '.ts');
  const actualTotalCount = actualTsxCount + actualTsCount;

  const scanResult = sourceAndRunScan(FRONTEND_SRC_DIR);

  assertEqual(scanResult.tsx_count, actualTsxCount, `  TSX count matches filesystem (${actualTsxCount})`);
  assertEqual(scanResult.ts_count, actualTsCount, `  TS count matches filesystem (${actualTsCount})`);
  assertEqual(scanResult.total_count, actualTotalCount, `  Total count matches filesystem (${actualTotalCount})`);
}

// Main test runner

function runAllTests() {
  console.log('\n==========================================');
  console.log('Diagnosis Scanner E2E Tests');
  console.log('==========================================');
  console.log(`Repository: ${REPO_DIR}`);
  console.log(`Frontend: ${FRONTEND_SRC_DIR}`);
  console.log('==========================================\n');

  const startTime = Date.now();

  try {
    // Run all tests
    testScanFrontendFilesReturnsValidJson();
    testTsxCountReturns80();
    testTsCountReturns33();
    testTotalCountReturns113();
    testExcludesNodeModules();
    testExcludesDistAndBuild();
    testNonexistentDirectoryReturnsZeros();
    testFeatureBreakdownAccuracy();
    testNotFollowSymlinks();
    testDiagnoseCreatesArtifact();
    testDiagnosisJsonHasCorrectCounts();
    testFilesystemCountMatchesScan();
  } finally {
    cleanup();
  }

  const duration = ((Date.now() - startTime) / 1000).toFixed(2);

  // Print summary
  console.log('\n==========================================');
  console.log('Test Summary');
  console.log('==========================================');
  console.log(`  Total:   ${testsRun}`);
  console.log(colors.green + `  Passed:  ${testsPassed}` + colors.reset);
  if (testsFailed > 0) {
    console.log(colors.red + `  Failed:  ${testsFailed}` + colors.reset);
  }
  console.log(`  Duration: ${duration}s`);
  console.log('==========================================');

  if (testsFailed > 0) {
    console.log('\nFailures:');
    failures.forEach(f => console.log(`  - ${f}`));
    console.log('');
  }

  return testsFailed === 0 ? 0 : 1;
}

// Run tests if executed directly
if (require.main === module) {
  const exitCode = runAllTests();
  process.exit(exitCode);
}

module.exports = { runAllTests };
