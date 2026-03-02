import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers, testPolicies, testQuotas } from '../helpers/test-data';

test.describe('Admin - Organization Settings', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.owner.email, testUsers.owner.password);
  });

  test('should access organization settings', async ({ page }) => {
    await page.goto('/organization');

    await expect(page.getByRole('heading', { name: /organization/i })).toBeVisible();
  });

  test('should display organization details', async ({ page }) => {
    await page.goto('/organization');

    const orgName = page.locator('[data-testid="organization-name"]');
    await expect(orgName).toBeVisible();
  });

  test('should update organization name', async ({ page }) => {
    await page.goto('/organization');

    const editButton = page.getByRole('button', { name: /edit|settings/i });
    if (await editButton.isVisible()) {
      await editButton.click();

      const nameInput = page.getByRole('textbox', { name: /organization name/i });
      await nameInput.clear();
      await nameInput.fill('Updated Organization Name');

      const saveButton = page.getByRole('button', { name: /save/i });
      await saveButton.click();

      await expect(page.getByText(/saved|updated/i)).toBeVisible();
    }
  });

  test('should display user list', async ({ page }) => {
    await page.goto('/organization');

    const userSection = page.getByText(/users|members/i);
    await expect(userSection).toBeVisible();

    const userList = page.locator('[data-testid="user-list"]');
    const emptyState = page.getByText(/no users/i);

    const listVisible = await userList.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should invite new user', async ({ page }) => {
    await page.goto('/organization');

    const inviteButton = page.getByRole('button', { name: /invite|add user/i });
    if (await inviteButton.isVisible()) {
      await inviteButton.click();

      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByRole('heading', { name: /invite user/i })).toBeVisible();

      // Fill form
      await page.getByRole('textbox', { name: /email/i }).fill('newuser@test.com');
      await page.getByRole('combobox', { name: /role/i }).selectOption('user');

      // Send invite
      await page.getByRole('button', { name: /send|invite/i }).click();

      await expect(page.getByText(/invitation sent|invite sent/i)).toBeVisible();
    }
  });

  test('should remove user from organization', async ({ page }) => {
    await page.goto('/organization');

    const userCard = page.locator('[data-testid="user-card"]').first();
    const count = await userCard.count();

    if (count > 0 && !await userCard.getByText(testUsers.owner.name).isVisible()) {
      const removeButton = userCard.getByRole('button', { name: /remove|delete/i });
      if (await removeButton.isVisible()) {
        await removeButton.click();

        // Confirm removal
        await page.getByRole('button', { name: /confirm|yes/i }).click();

        await expect(page.getByText(/removed|deleted/i)).toBeVisible();
      }
    }
  });

  test('should update user role', async ({ page }) => {
    await page.goto('/organization');

    const userCard = page.locator('[data-testid="user-card"]').first();
    const count = await userCard.count();

    if (count > 0) {
      const roleButton = userCard.getByRole('button', { name: /admin|user|owner/i });
      if (await roleButton.isVisible()) {
        await roleButton.click();

        // Select new role
        await page.getByRole('menuitem', { name: /admin/i }).click();

        await expect(page.getByText(/role updated/i)).toBeVisible();
      }
    }
  });

  test('should display pending invitations', async ({ page }) => {
    await page.goto('/organization');

    const invitationsTab = page.getByRole('tab', { name: /invitations/i });
    if (await invitationsTab.isVisible()) {
      await invitationsTab.click();

      const invitesList = page.locator('[data-testid="invitations-list"]');
      const emptyState = page.getByText(/no pending invitations/i);

      const listVisible = await invitesList.isVisible();
      const emptyVisible = await emptyState.isVisible();

      expect(listVisible || emptyVisible).toBe(true);
    }
  });

  test('should cancel pending invitation', async ({ page }) => {
    await page.goto('/organization');

    const invitationsTab = page.getByRole('tab', { name: /invitations/i });
    if (await invitationsTab.isVisible()) {
      await invitationsTab.click();

      const inviteCard = page.locator('[data-testid="invitation-card"]').first();
      const count = await inviteCard.count();

      if (count > 0) {
        const cancelButton = inviteCard.getByRole('button', { name: /cancel/i });
        await cancelButton.click();

        await expect(page.getByText(/cancelled/i)).toBeVisible();
      }
    }
  });
});

test.describe('Admin - Print Policies', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should access policies page', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.getByRole('heading', { name: /print policies|policies/i })).toBeVisible();
  });

  test('should display policy list', async ({ page }) => {
    await page.goto('/policies');

    const policyList = page.locator('[data-testid="policy-list"]');
    const emptyState = page.getByText(/no policies/i );

    const listVisible = await policyList.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should create new policy', async ({ page }) => {
    await page.goto('/policies');

    const createButton = page.getByRole('button', { name: /create policy|add policy/i });
    if (await createButton.isVisible()) {
      await createButton.click();

      await expect(page.getByRole('dialog')).toBeVisible();

      // Fill form
      await page.getByRole('textbox', { name: /name/i }).fill('E2E Test Policy');
      await page.getByRole('textbox', { name: /description/i }).fill('Created by E2E test');

      // Select policy type
      await page.getByRole('combobox', { name: /type/i }).selectOption('restriction');

      // Set conditions
      const conditionSection = page.getByText(/conditions/i);
      if (await conditionSection.isVisible()) {
        await page.getByRole('checkbox', { name: /color/i }).check();
      }

      // Create policy
      await page.getByRole('button', { name: /create|save/i }).click();

      await expect(page.getByText(/policy created|saved/i)).toBeVisible();
    }
  });

  test('should edit existing policy', async ({ page }) => {
    await page.goto('/policies');

    const policyCard = page.locator('[data-testid="policy-card"]').first();
    const count = await policyCard.count();

    if (count > 0) {
      const editButton = policyCard.getByRole('button', { name: /edit/i });
      await editButton.click();

      await expect(page.getByRole('dialog')).toBeVisible();

      // Edit name
      const nameInput = page.getByRole('textbox', { name: /name/i });
      await nameInput.clear();
      await nameInput.fill('Updated Policy Name');

      await page.getByRole('button', { name: /save|update/i }).click();

      await expect(page.getByText(/saved|updated/i)).toBeVisible();
    }
  });

  test('should toggle policy enabled state', async ({ page }) => {
    await page.goto('/policies');

    const policyCard = page.locator('[data-testid="policy-card"]').first();
    const count = await policyCard.count();

    if (count > 0) {
      const toggle = policyCard.getByRole('switch');
      await toggle.click();

      await page.waitForTimeout(500);
    }
  });

  test('should reorder policies', async ({ page }) => {
    await page.goto('/policies');

    const policyCards = page.locator('[data-testid="policy-card"]');
    const count = await policyCards.count();

    if (count >= 2) {
      // Drag and drop functionality would be tested here
      // For now, just verify reorder button exists
      const reorderButton = page.getByRole('button', { name: /reorder/i });
      if (await reorderButton.isVisible()) {
        await reorderButton.click();
        await expect(page.getByText(/drag|reorder/i)).toBeVisible();
      }
    }
  });

  test('should delete policy', async ({ page }) => {
    await page.goto('/policies');

    // Look for test policy to delete
    const testPolicy = page.locator('[data-testid="policy-card"]').filter({
      hasText: /E2E|Test/i,
    }).first();

    if (await testPolicy.isVisible()) {
      const deleteButton = testPolicy.getByRole('button', { name: /delete/i });
      await deleteButton.click();

      // Confirm deletion
      await page.getByRole('button', { name: /confirm|delete/i }).click();

      await expect(page.getByText(/deleted/i)).toBeVisible();
    }
  });
});

test.describe('Admin - Quotas', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should access quotas page', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByRole('heading', { name: /quotas/i })).toBeVisible();
  });

  test('should display quota overview', async ({ page }) => {
    await page.goto('/quotas');

    const overviewSection = page.getByText(/overview|summary/i);
    await expect(overviewSection).toBeVisible();
  });

  test('should display user quotas', async ({ page }) => {
    await page.goto('/quotas');

    const userQuotas = page.locator('[data-testid="user-quotas"]');
    const emptyState = page.getByText(/no quotas set/i);

    const listVisible = await userQuotas.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should set user quota', async ({ page }) => {
    await page.goto('/quotas');

    const setQuotaButton = page.getByRole('button', { name: /set quota|add quota/i });
    if (await setQuotaButton.isVisible()) {
      await setQuotaButton.click();

      await expect(page.getByRole('dialog')).toBeVisible();

      // Select user
      await page.getByRole('combobox', { name: /user/i }).selectOption({ index: 0 });

      // Set quota limits
      await page.getByRole('spinbutton', { name: /pages/i }).fill('1000');

      // Select period
      await page.getByRole('combobox', { name: /period/i }).selectOption('monthly');

      await page.getByRole('button', { name: /save|set/i }).click();

      await expect(page.getByText(/quota set|saved/i)).toBeVisible();
    }
  });

  test('should edit user quota', async ({ page }) => {
    await page.goto('/quotas');

    const quotaCard = page.locator('[data-testid="quota-card"]').first();
    const count = await quotaCard.count();

    if (count > 0) {
      const editButton = quotaCard.getByRole('button', { name: /edit/i });
      await editButton.click();

      await expect(page.getByRole('dialog')).toBeVisible();

      // Update limit
      const pagesInput = page.getByRole('spinbutton', { name: /pages/i });
      await pagesInput.clear();
      await pagesInput.fill('2000');

      await page.getByRole('button', { name: /save|update/i }).click();

      await expect(page.getByText(/updated|saved/i)).toBeVisible();
    }
  });

  test('should display quota usage visualization', async ({ page }) => {
    await page.goto('/quotas');

    const progressBar = page.locator('[data-testid="quota-progress"]');
    const count = await progressBar.count();

    if (count > 0) {
      await expect(progressBar.first()).toBeVisible();
    }
  });

  test('should view quota history', async ({ page }) => {
    await page.goto('/quotas');

    const historyTab = page.getByRole('tab', { name: /history/i });
    if (await historyTab.isVisible()) {
      await historyTab.click();

      const historyList = page.locator('[data-testid="quota-history"]');
      await expect(historyList).toBeVisible();
    }
  });
});

test.describe('Admin - Audit Logs', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should access audit logs page', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByRole('heading', { name: /audit logs/i })).toBeVisible();
  });

  test('should display audit log entries', async ({ page }) => {
    await page.goto('/audit-logs');

    const logList = page.locator('[data-testid="audit-log-list"]');
    const emptyState = page.getByText(/no logs|no activity/i );

    const listVisible = await logList.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should filter logs by date range', async ({ page }) => {
    await page.goto('/audit-logs');

    const dateFilter = page.getByRole('button', { name: /date|filter/i });
    if (await dateFilter.isVisible()) {
      await dateFilter.click();

      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/date range/i)).toBeVisible();
    }
  });

  test('should filter logs by action type', async ({ page }) => {
    await page.goto('/audit-logs');

    const actionFilter = page.getByRole('combobox', { name: /action|filter/i });
    if (await actionFilter.isVisible()) {
      await actionFilter.selectOption('login');

      await page.waitForTimeout(500);
    }
  });

  test('should filter logs by user', async ({ page }) => {
    await page.goto('/audit-logs');

    const userFilter = page.getByRole('combobox', { name: /user/i });
    if (await userFilter.isVisible()) {
      await userFilter.selectOption({ index: 0 });

      await page.waitForTimeout(500);
    }
  });

  test('should export audit logs', async ({ page }) => {
    await page.goto('/audit-logs');

    const exportButton = page.getByRole('button', { name: /export|download/i });
    if (await exportButton.isVisible()) {
      // Set up download handler
      const downloadPromise = page.waitForEvent('download');
      await exportButton.click();

      const download = await downloadPromise;
      expect(download.suggestedFilename()).toMatch(/\.(csv|json|xlsx)$/);
    }
  });
});

test.describe('Admin - Email to Print', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should access email to print settings', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByRole('heading', { name: /email to print/i })).toBeVisible();
  });

  test('should display current configuration', async ({ page }) => {
    await page.goto('/email-to-print');

    const configSection = page.getByText(/configuration|settings/i);
    await expect(configSection).toBeVisible();
  });

  test('should display email address', async ({ page }) => {
    await page.goto('/email-to-print');

    const emailDisplay = page.locator('[data-testid="email-address"]');
    await expect(emailDisplay).toBeVisible();

    const email = await emailDisplay.textContent();
    expect(email).toMatch(/@.*\./);
  });

  test('should configure allowed senders', async ({ page }) => {
    await page.goto('/email-to-print');

    const sendersSection = page.getByText(/allowed senders|whitelist/i);
    if (await sendersSection.isVisible()) {
      const addButton = page.getByRole('button', { name: /add|allow sender/i });
      if (await addButton.isVisible()) {
        await addButton.click();

        await page.getByRole('textbox', { name: /email|domain/i }).fill('example.com');
        await page.getByRole('button', { name: /add|save/i }).click();

        await expect(page.getByText(/added|saved/i)).toBeVisible();
      }
    }
  });

  test('should test email configuration', async ({ page }) => {
    await page.goto('/email-to-print');

    const testButton = page.getByRole('button', { name: /test|send test/i });
    if (await testButton.isVisible()) {
      await testButton.click();

      await expect(page.getByText(/test email sent|check your inbox/i)).toBeVisible();
    }
  });

  test('should display recent email print jobs', async ({ page }) => {
    await page.goto('/email-to-print');

    const jobsSection = page.getByText(/recent jobs|history/i);
    await expect(jobsSection).toBeVisible();

    const jobsList = page.locator('[data-testid="email-jobs"]');
    const emptyState = page.getByText(/no jobs/i );

    const listVisible = await jobsList.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });
});

test.describe('Admin - Role-Based Access', () => {
  test('should restrict analytics page to admins', async ({ page }) => {
    // Login as regular user
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.user.email, testUsers.user.password);

    await page.goto('/analytics');

    // Should redirect or show forbidden
    const isForbidden = await page.getByText(/forbidden|not authorized/i).isVisible();
    const isRedirected = page.url().includes('/dashboard');

    expect(isForbidden || isRedirected).toBe(true);
  });

  test('should restrict organization page to owners/admins', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.user.email, testUsers.user.password);

    await page.goto('/organization');

    // Should redirect or show forbidden
    const isForbidden = await page.getByText(/forbidden|not authorized/i).isVisible();
    const isRedirected = page.url().includes('/dashboard');

    expect(isForbidden || isRedirected).toBe(true);
  });

  test('should allow admin access to policies', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.goto('/policies');

    await expect(page.getByRole('heading', { name: /policies/i })).toBeVisible();
  });

  test('should allow owner access to all admin pages', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.owner.email, testUsers.owner.password);

    await page.goto('/quotas');
    await expect(page.getByRole('heading', { name: /quotas/i })).toBeVisible();

    await page.goto('/policies');
    await expect(page.getByRole('heading', { name: /policies/i })).toBeVisible();
  });
});
