# OpenPrint Dashboard E2E Tests

This directory contains comprehensive end-to-end tests for the OpenPrint Dashboard application using Playwright with the Page Object Model pattern.

## Test Structure

```
e2e/
├── pages/                    # Page Object Model classes
│   ├── BasePage.ts          # Base class with common navigation/auth methods
│   ├── DashboardPage.ts     # Dashboard page object
│   ├── AuthPage.ts          # Authentication page object
│   ├── JobsPage.ts          # Jobs page object
│   ├── PrintersPage.ts      # Printers page object
│   ├── AnalyticsPage.ts     # Analytics page object
│   ├── SettingsPage.ts      # Settings page object
│   ├── SecurePrintPage.ts   # Secure print release page object
│   ├── QuotasPage.ts        # Cost tracking and quotas page object
│   ├── PoliciesPage.ts      # Print policy engine page object
│   ├── Microsoft365Page.ts  # Microsoft 365 integration page object
│   ├── CompliancePage.ts    # Compliance reports page object
│   └── index.ts             # Central export point
├── factories/                # Test data generation
│   └── TestDataFactory.ts   # Factory for generating realistic test data
├── fixtures/                 # Test fixtures and utilities
│   ├── visual-regression.spec.ts  # Visual regression tests
│   └── websocket-mock.ts           # WebSocket mock utilities
├── tests/                    # E2E test suites
│   ├── auth.spec.ts          # Authentication tests
│   ├── dashboard.spec.ts     # Dashboard tests
│   ├── jobs.spec.ts          # Job management tests
│   ├── printers.spec.ts      # Printer management tests
│   ├── settings.spec.ts      # Settings tests
│   ├── analytics.spec.ts     # Analytics tests
│   ├── organization.spec.ts  # Organization tests
│   ├── agents.spec.ts        # Agent management tests
│   ├── policies.spec.ts      # Policy engine tests
│   ├── quotas.spec.ts        # Quotas tests
│   ├── email-to-print.spec.ts # Email to print tests
│   ├── print-release.spec.ts # Secure print tests
│   ├── audit-logs.spec.ts    # Advanced audit logs tests
│   ├── microsoft-365.spec.ts # Microsoft 365 integration tests
│   ├── compliance.spec.ts    # FedRAMP/HIPAA compliance tests
│   ├── realtime-updates.spec.ts # WebSocket real-time tests
│   ├── code-splitting.spec.ts # Code splitting tests
│   └── lazy-loading.spec.ts  # Lazy loading tests
├── helpers.ts                # Test utilities and mock data
├── tsconfig.json             # TypeScript configuration for E2E tests
└── README.md                 # This file
```

## Running Tests

### Run all E2E tests
```bash
npm run test:e2e
```

### Run E2E tests with UI
```bash
npm run test:e2e:ui
```

### Run specific test file
```bash
npx playwright test tests/auth.spec.ts

# Run specific test suites
npx playwright test tests/microsoft-365.spec.ts
npx playwright test tests/compliance.spec.ts
npx playwright test tests/realtime-updates.spec.ts

# Run visual regression tests
npx playwright test fixtures/visual-regression.spec.ts

# Run tests matching a pattern
npx playwright test --grep "Microsoft"
```

### Run tests in specific browser
```bash
npx playwright test --project=chromium
npx playwright test --project=firefox
npx playwright test --project=webkit
```

### Debug tests
```bash
npx playwright test --debug
```

### Run tests in headed mode
```bash
npx playwright test --headed
```

## Test Scenarios

### Login Page (`login.spec.ts`)
- Form display and toggle between login/register
- Form validation
- Authentication flow
- Error handling
- Token storage

### Dashboard (`dashboard.spec.ts`)
- Stats display
- Recent jobs and printers
- Environmental impact report
- Empty states
- Navigation

### Printers (`printers.spec.ts`)
- Printer list display
- Search and filtering
- Printer status toggling
- Agent installation notice
- Printer details

### Jobs (`jobs.spec.ts`)
- Job list display
- Status filtering
- Job actions (cancel, retry)
- Search functionality
- Job details

### Settings (`settings.spec.ts`)
- Profile management
- Password change
- Active sessions
- Preferences (theme, notifications, language, timezone)

### Organization (`organization.spec.ts`) - Admin only
- Organization info and plan details
- User management
- Role changes
- User invitations
- Printer permissions

### Analytics (`analytics.spec.ts`) - Admin only
- Usage metrics
- Environmental impact
- Charts and graphs
- Audit logs
- Period selection

## Mock Data

Tests use mock data from two sources:

### helpers.ts
Contains static mock data for basic tests:
- Mock users (regular and admin)
- Mock printers
- Mock print jobs
- Mock organization
- Mock usage statistics
- Mock audit logs

### factories/TestDataFactory.ts
Provides dynamic data generation:
- `UserFactory` - Generate realistic user data
- `PrinterFactory` - Generate printer configurations
- `JobFactory` - Generate print job data
- `AgentFactory` - Generate agent data
- `OrganizationFactory` - Generate organization data
- `PolicyFactory` - Generate policy configurations
- `AuditLogFactory` - Generate audit log entries
- `EnvironmentReportFactory` - Generate environmental reports

```typescript
import { TestDataFactory } from './factories/TestDataFactory';

// Generate a single user
const user = TestDataFactory.User.create({ name: 'Test User' });

// Generate multiple jobs
const jobs = TestDataFactory.Job.createMany(10, { status: 'completed' });
```

## Page Object Model

The E2E tests use the Page Object Model pattern for maintainability:

```typescript
import { test } from '@playwright/test';
import { DashboardPage } from './pages';

test.describe('Dashboard', () => {
  test('should display stats', async ({ page }) => {
    const dashboard = new DashboardPage(page);
    await dashboard.setupMocks();
    await dashboard.goto();

    await dashboard.verifyStatsDisplayed();
    await dashboard.verifyEnvironmentalImpactDisplayed();
  });
});
```

## Visual Regression Tests

Visual regression tests ensure UI consistency:

```bash
# Run visual regression tests
npx playwright test visual-regression

# Update screenshots
npx playwright test --update-snapshots
```

## Real-time Updates Tests

Tests for WebSocket functionality use mock utilities:

```typescript
import { setupWebSocketMock } from './fixtures/websocket-mock';

test('should receive job updates', async ({ page }) => {
  const wsMock = await setupWebSocketMock(page);
  await setupAuthAndNavigate(page, '/jobs');

  wsMock.sendJobUpdate({
    jobId: 'job-1',
    status: 'processing',
    progress: 50,
  });

  // Verify UI updated
  // ...
});
```

## Writing New Tests

### Using Page Objects (Recommended)

```typescript
import { test, expect } from '@playwright/test';
import { JobsPage } from './pages';
import { setupAuthAndNavigate } from './helpers';

test.describe('Job Management', () => {
  test('should create new job', async ({ page }) => {
    const jobsPage = new JobsPage(page);
    await jobsPage.setupMocks();
    await setupAuthAndNavigate(page, '/jobs');

    await jobsPage.openCreateJobModal();
    await jobsPage.fillJobForm({
      printer: 'printer-1',
      copies: 2,
      color: true,
    });
    await jobsPage.submitJob();

    await expect(page.locator('.toast')).toContainText('Job created');
  });
});
```

### Using Test Data Factory

```typescript
import { TestDataFactory } from './factories/TestDataFactory';

test('should handle multiple jobs', async ({ page }) => {
  const jobs = TestDataFactory.Job.createMany(50, {
    status: 'completed',
  });

  // Mock API with generated data
  await page.route('**/api/v1/jobs*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: jobs, total: jobs.length }),
    });
  });
});
```

### Testing Real-time Updates

```typescript
import { setupWebSocketMock, simulateRealtimeScenario } from './fixtures/websocket-mock';

test('should show real-time job progress', async ({ page }) => {
  const wsMock = await setupWebSocketMock(page);
  await setupAuthAndNavigate(page, '/jobs');

  // Simulate job progress over 3 seconds
  await wsMock.simulateJobProgress('job-1', 3000);

  // Verify progress indicators
  // ...
});
```

## CI/CD

Tests run in CI with:
- Retries enabled (2 attempts)
- Single worker for stability
- HTML reporter for results
- Screenshots and videos on failure
