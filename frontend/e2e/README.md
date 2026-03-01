# OpenPlay E2E Tests

End-to-end tests for the OpenPrint Dashboard using Playwright.

## Setup

```bash
cd frontend/e2e
npm install
npx playwright install
```

## Running Tests

```bash
# Run all tests (with local dev server)
npm test

# Run tests in UI mode (interactive)
npm run test:ui

# Run tests in headed mode (watch browser)
npm run test:headed

# Debug tests
npm run test:debug

# Run specific browser
npm run test:chromium
```

## Running Tests Against Staging/Production

```bash
# Against staging
BASE_URL=https://staging.openprint.test npm test

# Against production
BASE_URL=https://openprint.example.com npm test

# Skip web server (when testing against running instance)
SKIP_WEBSERVER=1 npm test
```

## Test Structure

```
e2e/
├── helpers/
│   ├── page-objects.ts   # Page Object Model classes
│   ├── test-data.ts      # Test data fixtures
│   └── api-helpers.ts    # API helper functions
├── tests/
│   ├── auth.spec.ts      # Authentication tests
│   ├── dashboard.spec.ts # Dashboard tests
│   ├── printers.spec.ts  # Printers & devices tests
│   ├── jobs.spec.ts      # Print jobs tests
│   └── admin.spec.ts     # Admin features tests
├── playwright.config.ts  # Playwright configuration
└── tsconfig.json         # TypeScript configuration
```

## Test Users

The tests use these predefined users (created via API helpers):

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@openprint.test | TestAdmin123! |
| User | user@openprint.test | TestUser123! |
| Owner | owner@openprint.test | TestOwner123! |

## Writing New Tests

### Page Objects

Use Page Object Model for maintainable tests:

```typescript
import { Page } from '@playwright/test';
import { BasePage } from '../helpers/page-objects';

export class MyPage extends BasePage {
  constructor(page: Page) {
    super(page);
  }

  async navigate() {
    await this.goto('/my-page');
  }

  async clickButton() {
    await this.page.getByRole('button', { name: /click me/i }).click();
  }
}
```

### Test Example

```typescript
import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';

test('my test', async ({ page }) => {
  const loginPage = new LoginPage(page);
  await loginPage.goto();
  await loginPage.login('user@test.com', 'password123');

  await expect(page).toHaveURL('/dashboard');
});
```

## CI/CD Integration

```yaml
# .github/workflows/e2e.yml example
- name: Install dependencies
  run: cd frontend/e2e && npm install

- name: Install Playwright browsers
  run: npx playwright install --with-deps

- name: Run E2E tests
  run: cd frontend/e2e && npm test
  env:
    BASE_URL: http://localhost:3000
```

## Troubleshooting

### Tests fail to find elements
- Increase timeouts in `playwright.config.ts`
- Check if selectors are correct
- Use Playwright's codegen: `npx playwright codegen`

### Flaky tests
- Increase `retries` in config
- Use proper `waitFor` conditions
- Avoid hard-coded delays

### Browser installation issues
```bash
npx playwright install --force
npx playwright install-deps
```
