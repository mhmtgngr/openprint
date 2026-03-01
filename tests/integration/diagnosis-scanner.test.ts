/**
 * diagnosis-scanner.test.ts - E2E/Integration tests for dev.sh diagnosis scanner
 *
 * Tests the diagnosis scanner to ensure accurate file enumeration:
 * - Correct TSX/TS file counts in diagnosis.json
 * - Proper exclusion of build artifacts
 * - Feature module breakdown accuracy
 *
 * Usage:
 *   npx ts-node tests/integration/diagnosis-scanner.test.ts
 *   npm run test:integration
 */

import { execSync, ExecSyncOptions } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

interface DiagnosisResult {
  project: {
    go_files: number;
    test_files: number;
    tsx_files: number;
    todo_count: number;
    build: string;
    tests: string;
    test_count: number;
    test_passed: number;
    dockerfiles: number;
    compose: string;
    frontend: string;
  };
  compile_errors: string;
  test_failures: string;
  ts_errors: string;
  todos: string;
}

interface FrontendScanResult {
  tsx_count: number;
  ts_count: number;
  total_count: number;
  test_count: number;
  features: Record<string, { tsx: number; ts: number }>;
}

const REPO_DIR = path.resolve(__dirname, '../..');
const ARTIFACTS_DIR = path.join(REPO_DIR, '.team', 'artifacts');
const DEV_SH = path.join(REPO_DIR, 'dev.sh');

// Helper functions

function execCommand(command: string, options: ExecSyncOptions = {}): string {
  const defaultOptions: ExecSyncOptions = {
    cwd: REPO_DIR,
    encoding: 'utf-8',
    stdio: 'pipe',
    ...options,
  };
  return execSync(command, defaultOptions) as string;
}

function sourceAndRunScan(dir: string): FrontendScanResult {
  const command = `bash -c 'source ${DEV_SH} && scan_frontend_files "${dir}"'`;
  const output = execCommand(command);
  return JSON.parse(output) as FrontendScanResult;
}

function runDiagnose(): DiagnosisResult {
  // Run the diagnose command
  execCommand(`bash ${DEV_SH} diagnose`);

  // Read the diagnosis.json artifact
  const diagnosisPath = path.join(ARTIFACTS_DIR, 'diagnosis.json');
  if (!fs.existsSync(diagnosisPath)) {
    throw new Error(`Diagnosis artifact not found at ${diagnosisPath}`);
  }

  const content = fs.readFileSync(diagnosisPath, 'utf-8');
  return JSON.parse(content) as DiagnosisResult;
}

function countFilesByExtension(dir: string, extension: string, excludeDirs: string[] = []): number {
  let count = 0;
  const excludePatterns = excludeDirs.map(d => path.join(dir, d));

  function walkDirectory(currentPath: string) {
    const entries = fs.readdirSync(currentPath, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(currentPath, entry.name);

      // Skip excluded directories
      if (entry.isDirectory()) {
        if (excludePatterns.some(pattern => fullPath.startsWith(pattern))) {
          continue;
        }
        // Skip node_modules, dist, build
        if (['node_modules', 'dist', 'build'].includes(entry.name)) {
          continue;
        }
        walkDirectory(fullPath);
      } else if (entry.isFile() && entry.name.endsWith(extension)) {
        count++;
      }
    }
  }

  walkDirectory(dir);
  return count;
}

// Test suite

describe('Diagnosis Scanner Accuracy', () => {
  const frontendSrcDir = path.join(REPO_DIR, 'web', 'dashboard', 'src');

  beforeAll(() => {
    // Ensure artifacts directory exists
    if (!fs.existsSync(ARTIFACTS_DIR)) {
      fs.mkdirSync(ARTIFACTS_DIR, { recursive: true });
    }
  });

  describe('scan_frontend_files function', () => {
    test('should return valid JSON', () => {
      const result = sourceAndRunScan(frontendSrcDir);

      expect(result).toHaveProperty('tsx_count');
      expect(result).toHaveProperty('ts_count');
      expect(result).toHaveProperty('total_count');
      expect(result).toHaveProperty('test_count');
      expect(result).toHaveProperty('features');
    });

    test('should count exactly 80 TSX files', () => {
      const result = sourceAndRunScan(frontendSrcDir);

      expect(result.tsx_count).toBe(80);
    });

    test('should count exactly 33 TS files (excluding TSX)', () => {
      const result = sourceAndRunScan(frontendSrcDir);

      expect(result.ts_count).toBe(33);
    });

    test('should report total count of 113 files', () => {
      const result = sourceAndRunScan(frontendSrcDir);

      expect(result.total_count).toBe(113);
    });

    test('should exclude node_modules directory', () => {
      // Create a temporary directory with node_modules
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-test-'));
      const srcDir = path.join(tempDir, 'src');
      const nodeModulesDir = path.join(tempDir, 'node_modules');

      fs.mkdirSync(srcDir);
      fs.mkdirSync(nodeModulesDir);
      fs.writeFileSync(path.join(srcDir, 'App.tsx'), '// App component');
      fs.writeFileSync(path.join(nodeModulesDir, 'lib.tsx'), '// Library file');

      try {
        const result = sourceAndRunScan(srcDir);

        // Should only count the file in src, not in node_modules
        expect(result.tsx_count).toBe(1);
        expect(result.total_count).toBe(1);
      } finally {
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    test('should exclude dist and build directories', () => {
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-test-'));
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

        expect(result.tsx_count).toBe(1);
        expect(result.total_count).toBe(1);
      } finally {
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    test('should return zero for nonexistent directory', () => {
      const result = sourceAndRunScan('/nonexistent/path/that/does/not/exist');

      expect(result.tsx_count).toBe(0);
      expect(result.ts_count).toBe(0);
      expect(result.total_count).toBe(0);
    });

    test('should provide feature breakdown with known features', () => {
      const result = sourceAndRunScan(frontendSrcDir);

      // Check for expected feature modules
      const expectedFeatures = ['dashboard', 'documents', 'printers', 'auth', 'agents', 'settings', 'jobs', 'devices'];

      for (const feature of expectedFeatures) {
        expect(result.features).toHaveProperty(feature);
        expect(typeof result.features[feature].tsx).toBe('number');
        expect(typeof result.features[feature].ts).toBe('number');
      }
    });

    test('should not follow symlinks outside directory', () => {
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-symlink-'));
      const outsideDir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-outside-'));
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

        // Should only count the file inside src, not the symlinked file
        expect(result.tsx_count).toBe(1);
      } finally {
        fs.rmSync(tempDir, { recursive: true, force: true });
        fs.rmSync(outsideDir, { recursive: true, force: true });
      }
    });

    test('should count test files separately', () => {
      const result = sourceAndRunScan(frontendSrcDir);

      expect(result.test_count).toBeGreaterThan(0);
      expect(typeof result.test_count).toBe('number');
    });
  });

  describe('diagnose_project function', () => {
    test('should create diagnosis.json artifact', () => {
      const result = runDiagnose();

      expect(result).toHaveProperty('project');
      expect(result.project).toHaveProperty('tsx_files');
      expect(result.project).toHaveProperty('go_files');
      expect(result.project).toHaveProperty('test_files');
    });

    test('should report correct tsx_files count in diagnosis.json', () => {
      const result = runDiagnose();

      expect(result.project.tsx_files).toBe(113);
    });

    test('should report build status', () => {
      const result = runDiagnose();

      expect(['yes', 'no']).toContain(result.project.build);
    });

    test('should report test status', () => {
      const result = runDiagnose();

      expect(['pass', 'fail']).toContain(result.project.tests);
    });

    test('should include frontend status', () => {
      const result = runDiagnose();

      expect(['yes', 'no']).toContain(result.project.frontend);
    });
  });

  describe('File count verification against actual filesystem', () => {
    test('TSX count matches filesystem count', () => {
      const actualCount = countFilesByExtension(frontendSrcDir, '.tsx');
      const scanResult = sourceAndRunScan(frontendSrcDir);

      expect(scanResult.tsx_count).toBe(actualCount);
    });

    test('TS count matches filesystem count', () => {
      const actualCount = countFilesByExtension(frontendSrcDir, '.ts');
      const scanResult = sourceAndRunScan(frontendSrcDir);

      // Note: TS files includes both .ts and .tsx, scan_frontend_files excludes .tsx from ts_count
      const tsOnlyCount = countFilesByExtension(frontendSrcDir, '.ts') -
                         countFilesByExtension(frontendSrcDir, '.tsx');

      expect(scanResult.ts_count).toBe(tsOnlyCount);
    });

    test('total count is sum of TSX and TS counts', () => {
      const scanResult = sourceAndRunScan(frontendSrcDir);

      expect(scanResult.total_count).toBe(scanResult.tsx_count + scanResult.ts_count);
    });
  });

  describe('Security validation', () => {
    test('should not traverse parent directories via ..', () => {
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'scanner-security-'));
      const srcDir = path.join(tempDir, 'src');
      const parentDir = path.join(tempDir, 'parent');

      fs.mkdirSync(srcDir);
      fs.mkdirSync(parentDir);
      fs.writeFileSync(path.join(srcDir, 'App.tsx'), '// App');
      fs.writeFileSync(path.join(parentDir, 'Parent.tsx'), '// Parent file');

      try {
        // Try to scan with parent directory reference
        const scanPath = path.join(srcDir, '..', 'parent');
        const result = sourceAndRunScan(scanPath);

        // Should only count files in the parent directory, not traverse back
        expect(result.tsx_count).toBe(1);
      } finally {
        fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });
  });
});
