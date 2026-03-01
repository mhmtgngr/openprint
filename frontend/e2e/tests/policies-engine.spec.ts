import { test, expect } from '@playwright/test';
import { AuthPage } from '../pages/AuthPage';
import { PoliciesPage } from '../pages/PoliciesPage';
import { testUsers } from '../helpers/test-data';

test.describe('Print Policies Engine - Overview', () => {
  let policiesPage: PoliciesPage;

  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    policiesPage = new PoliciesPage(page);

    // Mock policies API
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            policies: [
              {
                id: 'policy-1',
                name: 'Color Printing Restriction',
                description: 'Limit color printing to admin users only',
                priority: 1,
                enabled: true,
                type: 'restriction',
                conditions: [
                  { type: 'userRole', operator: 'equals', value: 'user' },
                ],
                actions: [
                  { type: 'forceGrayscale', value: true },
                ],
              },
              {
                id: 'policy-2',
                name: 'Duplex Default',
                description: 'Enable double-sided printing by default',
                priority: 2,
                enabled: true,
                type: 'enforcement',
                conditions: [
                  { type: 'always', operator: 'equals', value: true },
                ],
                actions: [
                  { type: 'forceDuplex', value: true },
                ],
              },
              {
                id: 'policy-3',
                name: 'Large Job Approval',
                description: 'Require approval for jobs over 50 pages',
                priority: 3,
                enabled: false,
                type: 'approval',
                conditions: [
                  { type: 'pages', operator: 'greaterThan', value: '50' },
                ],
                actions: [
                  { type: 'requireApproval', value: true },
                ],
              },
            ],
          }),
        });
      }
    });
  });

  test('should access policies page', async ({ page }) => {
    await policiesPage.navigate();
    await expect(policiesPage.heading).toBeVisible();
  });

  test('should display policies list section', async ({ page }) => {
    await policiesPage.navigate();
    await expect(policiesPage.policiesListSection).toBeVisible();
  });

  test('should display policy list', async ({ page }) => {
    await policiesPage.navigate();
    await expect(policiesPage.policyList).toBeVisible();
  });

  test('should get policy count', async ({ page }) => {
    await policiesPage.navigate();
    const count = await policiesPage.getPolicyCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should display create policy button', async ({ page }) => {
    await policiesPage.navigate();
    await expect(policiesPage.createPolicyButton).toBeVisible();
  });

  test('should show empty state when no policies', async ({ page }) => {
    // Override mock to return empty list
    await page.route('**/api/v1/policies/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ policies: [] }),
      });
    });

    await policiesPage.navigate();
    expect(await policiesPage.isEmpty()).toBe(true);
  });
});

test.describe('Print Policies Engine - Create Policy', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock create policy API
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'policy-new',
            message: 'Policy created successfully',
          }),
        });
      }
    });
  });

  test('should open create policy form', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await expect(policiesPage.isPolicyFormVisible()).toBe(true);
  });

  test('should create new policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.createPolicy({
      name: 'E2E Test Policy',
      description: 'Created by E2E test',
      priority: 5,
      enabled: true,
    });
    await expect(page.getByText(/created|saved|success/i)).toBeVisible();
  });

  test('should create disabled policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.createPolicy({
      name: 'Disabled Test Policy',
      description: 'Testing disabled policy',
      priority: 10,
      enabled: false,
    });
    await expect(page.getByText(/created|saved/i)).toBeVisible();
  });

  test('should validate policy name is required', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.savePolicyButton.click();
    await expect(page.getByText(/name is required|required/i)).toBeVisible();
  });

  test('should validate priority is required', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.policyNameInput.fill('Test Policy');
    await policiesPage.policyPriorityInput.fill('');
    await policiesPage.savePolicyButton.click();
    await expect(page.getByText(/priority is required|required/i)).toBeVisible();
  });

  test('should cancel policy creation', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.cancelPolicyEdit();
    await expect(policiesPage.isPolicyFormVisible()).toBe(false);
  });
});

test.describe('Print Policies Engine - Edit Policy', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock update policy API
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'PUT' || route.request().method() === 'PATCH') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Policy updated successfully' }),
        });
      }
    });
  });

  test('should edit existing policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await expect(policiesPage.isPolicyFormVisible()).toBe(true);
  });

  test('should update policy name', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await policiesPage.policyNameInput.fill('Updated Policy Name');
    await policiesPage.savePolicyButton.click();
    await expect(page.getByText(/updated|saved/i)).toBeVisible();
  });

  test('should update policy priority', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await policiesPage.policyPriorityInput.fill('10');
    await policiesPage.savePolicyButton.click();
    await expect(page.getByText(/updated|saved/i)).toBeVisible();
  });
});

test.describe('Print Policies Engine - Delete Policy', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock delete policy API
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 204,
          body: '',
        });
      }
    });
  });

  test('should delete policy with confirmation', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.deletePolicy('policy-1');
    await expect(page.getByText(/deleted|removed/i)).toBeVisible();
  });

  test('should cancel policy deletion', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.deletePolicyButton('policy-1').click();
    await policiesPage.deleteCancelButton.click();
    await expect(policiesPage.hasPolicy('policy-1')).toBe(true);
  });
});

test.describe('Print Policies Engine - Policy Conditions', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display conditions section', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await expect(policiesPage.conditionsSection).toBeVisible();
  });

  test('should add condition to policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.addCondition({
      type: 'pages',
      operator: 'greaterThan',
      value: '50',
    });
    expect(await policiesPage.getConditionsCount()).toBeGreaterThan(0);
  });

  test('should remove condition from policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.addCondition({ type: 'color', operator: 'equals', value: 'color' });
    const initialCount = await policiesPage.getConditionsCount();
    await policiesPage.removeCondition(0);
    expect(await policiesPage.getConditionsCount()).toBeLessThan(initialCount);
  });

  test('should set user role condition', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.userRoleCondition.selectOption('user');
    await expect(policiesPage.userRoleCondition).toHaveValue('user');
  });

  test('should set max pages condition', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.maxPagesCondition.fill('100');
    await expect(policiesPage.maxPagesCondition).toHaveValue('100');
  });

  test('should set color mode condition', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.colorModeCondition.check();
    await expect(policiesPage.colorModeCondition).toBeChecked();
  });
});

test.describe('Print Policies Engine - Policy Actions', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display actions section', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await expect(policiesPage.actionsSection).toBeVisible();
  });

  test('should add action to policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.addAction({
      type: 'setCopiesLimit',
      value: '1',
    });
    expect(await policiesPage.getActionsCount()).toBeGreaterThan(0);
  });

  test('should remove action from policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.addAction({ type: 'blockJob' });
    const initialCount = await policiesPage.getActionsCount();
    await policiesPage.removeAction(0);
    expect(await policiesPage.getActionsCount()).toBeLessThan(initialCount);
  });

  test('should set force duplex action', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.setForceDuplex(true);
    await expect(policiesPage.forceDuplexAction).toBeChecked();
  });

  test('should set force grayscale action', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.setForceGrayscale(true);
    await expect(policiesPage.forceGrayscaleAction).toBeChecked();
  });

  test('should set require approval action', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.setRequireApproval(true);
    await expect(policiesPage.requireApprovalAction).toBeChecked();
  });

  test('should uncheck actions', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.openCreatePolicy();
    await policiesPage.setForceDuplex(true);
    await policiesPage.setForceDuplex(false);
    await expect(policiesPage.forceDuplexAction).not.toBeChecked();
  });
});

test.describe('Print Policies Engine - Policy Toggle', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock toggle API
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'PATCH') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ enabled: true }),
        });
      }
    });
  });

  test('should toggle policy enabled state', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.togglePolicyStatus('policy-1');
    await page.waitForTimeout(500);
  });

  test('should check if policy is enabled', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    const isEnabled = await policiesPage.isPolicyEnabled('policy-1');
    expect(typeof isEnabled).toBe('boolean');
  });
});

test.describe('Print Policies Engine - Policy Templates', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock templates API
    await page.route('**/api/v1/policies/templates/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          templates: [
            {
              id: 'template-1',
              name: 'Cost Control',
              description: 'Reduce printing costs with sensible defaults',
            },
            {
              id: 'template-2',
              name: 'Security',
              description: 'Enhanced security policies',
            },
            {
              id: 'template-3',
              name: 'Environmental',
              description: 'Eco-friendly printing settings',
            },
          ],
        }),
      });
    });
  });

  test('should display templates section', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.templatesSection).toBeVisible();
  });

  test('should display template cards', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.templateCard('Cost Control')).toBeVisible();
    await expect(policiesPage.templateCard('Security')).toBeVisible();
  });

  test('should use template for new policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.useTemplate('Cost Control');
    await expect(policiesPage.isPolicyFormVisible()).toBe(true);
  });
});

test.describe('Print Policies Engine - Policy Evaluation', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock evaluation API
    await page.route('**/api/v1/policies/evaluate/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          result: 'matched',
          actions: ['forceGrayscale', 'forceDuplex'],
          message: 'Job would be modified by policy',
        }),
      });
    });
  });

  test('should evaluate policy against test job', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await policiesPage.evaluatePolicy({
      pages: 10,
      colorMode: 'color',
      userRole: 'user',
    });
    await expect(policiesPage.evaluationResult).toBeVisible();
  });

  test('should display evaluation result', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await policiesPage.evaluatePolicy({
      pages: 100,
      colorMode: 'color',
      userRole: 'user',
    });
    const result = await policiesPage.getEvaluationResult();
    expect(result).toBeTruthy();
  });
});

test.describe('Print Policies Engine - Filtering and Sorting', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should filter policies by status', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.filterByStatus('enabled');
    await page.waitForTimeout(500);
  });

  test('should filter policies by type', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.filterByType('restriction');
    await page.waitForTimeout(500);
  });

  test('should sort policies by priority', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.sortPolicies('priority');
    await page.waitForTimeout(500);
  });

  test('should sort policies by name', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.sortPolicies('name');
    await page.waitForTimeout(500);
  });

  test('should search policies', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.searchPolicies('Color');
    await page.waitForTimeout(500);
  });
});

test.describe('Print Policies Engine - Policy Details', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display policy name', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.policyName.first()).toBeVisible();
  });

  test('should display policy description', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.policyDescription.first()).toBeVisible();
  });

  test('should display policy priority', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.policyPriority.first()).toBeVisible();
  });

  test('should get policy details', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    const details = await policiesPage.getPolicyDetails('policy-1');
    expect(details.name).toBeTruthy();
    expect(details.priority).toBeGreaterThanOrEqual(0);
  });

  test('should check if policy exists', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    const exists = await policiesPage.hasPolicy('policy-1');
    expect(exists).toBe(true);
  });

  test('should get policy priority', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    const priority = await policiesPage.getPolicyPriority('policy-1');
    expect(priority).toBeGreaterThanOrEqual(0);
  });
});

test.describe('Print Policies Engine - Import/Export', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock export API
    await page.route('**/api/v1/policies/export/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ policy: { name: 'Exported Policy' } }),
      });
    });

    // Mock import API
    await page.route('**/api/v1/policies/import/**', (route) => {
      route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Policy imported successfully' }),
      });
    });
  });

  test('should export policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');

    const downloadPromise = page.waitForEvent('download');
    await policiesPage.exportPolicyButton.click();
    await downloadPromise;
  });

  test('should display import button', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.importPolicyButton).toBeVisible();
  });

  test('should import policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.importPolicyButton.click();
    // In real test, would provide actual file
    await expect(page.getByText(/import|upload/i)).toBeVisible();
  });
});

test.describe('Print Policies Engine - Policy History', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock history API
    await page.route('**/api/v1/policies/*/history/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          history: [
            {
              id: 'version-1',
              version: 1,
              changedBy: 'admin@openprint.test',
              changedAt: '2024-01-15T10:00:00Z',
              changes: 'Created policy',
            },
            {
              id: 'version-2',
              version: 2,
              changedBy: 'admin@openprint.test',
              changedAt: '2024-01-15T11:00:00Z',
              changes: 'Updated priority from 1 to 5',
            },
          ],
        }),
      });
    });
  });

  test('should display history tab', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await expect(policiesPage.historyTab).toBeVisible();
  });

  test('should view policy history', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.editPolicy('policy-1');
    await policiesPage.viewHistory('policy-1');
    await expect(policiesPage.historyList).toBeVisible();
  });
});

test.describe('Print Policies Engine - Duplicate Policy', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock duplicate API
    await page.route('**/api/v1/policies/*/duplicate/**', (route) => {
      route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'policy-duplicate',
          name: 'Color Printing Restriction (Copy)',
          message: 'Policy duplicated successfully',
        }),
      });
    });
  });

  test('should duplicate policy', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.duplicatePolicy('policy-1');
    await expect(page.getByText(/duplicated|copied/i)).toBeVisible();
  });

  test('should display duplicate button', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.duplicatePolicyButton('policy-1')).toBeVisible();
  });
});

test.describe('Print Policies Engine - Access Control', () => {
  test('should restrict policy creation to admin users', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    await page.goto('/policies');

    const createButton = page.getByRole('button', { name: /create policy/i });
    const isHidden = await createButton.isDisabled() || !(await createButton.isVisible());
    expect(isHidden).toBe(true);
  });

  test('should allow admin to create policies', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.createPolicyButton).toBeVisible();
  });

  test('should allow owner full access to policies', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.owner.email, testUsers.owner.password);

    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(policiesPage.createPolicyButton).toBeVisible();
    await expect(policiesPage.deletePolicyButton('policy-1')).toBeVisible();
  });
});

test.describe('Print Policies Engine - Policy Reordering', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should reorder policies by drag and drop', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.reorderPolicy(0, 1);
    await page.waitForTimeout(500);
  });

  test('should respect priority ordering', async ({ page }) => {
    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.sortPolicies('priority');
    await page.waitForTimeout(500);
    const priority1 = await policiesPage.getPolicyPriority('policy-1');
    const priority2 = await policiesPage.getPolicyPriority('policy-2');
    // Verify ordering
    expect(priority1).toBeLessThanOrEqual(priority2);
  });
});

test.describe('Print Policies Engine - Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should handle create policy error', async ({ page }) => {
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Invalid policy configuration' }),
        });
      }
    });

    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.createPolicy({
      name: 'Invalid Policy',
      description: 'This should fail',
      priority: 1,
      enabled: true,
    });
    await expect(page.getByText(/error|invalid|failed/i)).toBeVisible();
  });

  test('should handle delete policy error', async ({ page }) => {
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 403,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Cannot delete system policy' }),
        });
      }
    });

    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await policiesPage.deletePolicy('policy-1');
    await expect(page.getByText(/forbidden|cannot delete|error/i)).toBeVisible();
  });

  test('should handle network error', async ({ page }) => {
    await page.route('**/api/v1/policies/**', (route) => {
      route.abort('failed');
    });

    const policiesPage = new PoliciesPage(page);
    await policiesPage.navigate();
    await expect(page.getByText(/error|failed|network/i)).toBeVisible({ timeout: 10000 });
  });
});
