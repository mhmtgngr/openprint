# OpenPrint Dashboard E2E Tests

This directory contains end-to-end tests for the OpenPrint Dashboard application using Playwright.

## Test Structure

```
e2e/
├── login.spec.ts         # Authentication and registration tests
├── dashboard.spec.ts     # Main dashboard tests
├── printers.spec.ts      # Printer management tests
├── jobs.spec.ts          # Print job management tests
├── settings.spec.ts      # User settings tests
├── organization.spec.ts  # Organization management (admin) tests
├── analytics.spec.ts     # Analytics and reporting (admin) tests
├── helpers.ts            # Test utilities and mock data
└── tsconfig.json         # TypeScript configuration for E2E tests
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
npx playwright test login.spec.ts
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

All tests use mock data defined in `helpers.ts`:
- Mock users (regular and admin)
- Mock printers
- Mock print jobs
- Mock organization
- Mock usage statistics
- Mock audit logs

## Writing New Tests

1. Create a new spec file in the `e2e/` directory
2. Import necessary utilities from `helpers.ts`
3. Use `test.describe()` to group related tests
4. Use `test.beforeEach()` for common setup
5. Mock API responses using `page.route()`
6. Use Playwright's locators and assertions

Example:
```typescript
import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers } from './helpers';

test.describe('New Feature', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });
    await login(page);
  });

  test('should do something', async ({ page }) => {
    await page.goto('/new-page');
    await expect(page.locator('h1')).toContainText('New Page');
  });
});
```

## CI/CD

Tests run in CI with:
- Retries enabled (2 attempts)
- Single worker for stability
- HTML reporter for results
- Screenshots and videos on failure
