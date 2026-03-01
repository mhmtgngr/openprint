/**
 * Policies Page Object
 * Handles print policy engine management
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse } from '../helpers';

export interface PrintPolicy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  priority: number;
  conditions: PolicyCondition[];
  actions: PolicyAction[];
  scope: PolicyScope;
  createdAt: string;
  updatedAt: string;
}

export interface PolicyCondition {
  type: 'user' | 'group' | 'printer' | 'document' | 'time' | 'cost';
  operator: 'equals' | 'contains' | 'matches' | 'in' | 'greater_than' | 'less_than';
  value: string | number | string[];
}

export interface PolicyAction {
  type: 'allow' | 'deny' | 'redirect' | 'modify' | 'require_approval' | 'log';
  parameter?: Record<string, unknown>;
}

export interface PolicyScope {
  users?: string[];
  groups?: string[];
  printers?: string[];
  departments?: string[];
}

export class PoliciesPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly policiesList: Locator;
  readonly policyItems: Locator;
  readonly createPolicyButton: Locator;
  readonly searchInput: Locator;
  readonly statusFilters: Locator;
  readonly emptyState: Locator;
  readonly policyCount: Locator;

  // Policy form
  readonly policyForm: Locator;
  readonly policyNameInput: Locator;
  readonly policyDescriptionInput: Locator;
  readonly policyPriorityInput: Locator;
  readonly policyEnabledToggle: Locator;
  readonly addConditionButton: Locator;
  readonly addActionButton: Locator;
  readonly savePolicyButton: Locator;
  readonly cancelButton: Locator;

  // Conditions builder
  readonly conditionTypeSelect: Locator;
  readonly conditionOperatorSelect: Locator;
  readonly conditionValueInput: Locator;
  readonly removeConditionButton: Locator;

  // Actions builder
  readonly actionTypeSelect: Locator;
  readonly actionParameterInput: Locator;
  readonly removeActionButton: Locator;

  // Scope selector
  readonly scopeUsersSelect: Locator;
  readonly scopeGroupsSelect: Locator;
  readonly scopePrintersSelect: Locator;

  // Policy detail
  readonly policyDetailPage: Locator;
  readonly policyDetailName: Locator;
  readonly policyDetailConditions: Locator;
  readonly policyDetailActions: Locator;
  readonly policyDetailScope: Locator;
  readonly editPolicyButton: Locator;
  readonly deletePolicyButton: Locator;
  readonly duplicatePolicyButton: Locator;
  readonly testPolicyButton: Locator;

  // Policy testing
  readonly testPolicyModal: Locator;
  readonly testUserSelect: Locator;
  readonly testDocumentInput: Locator;
  readonly testPrinterSelect: Locator;
  readonly runTestButton: Locator;
  readonly testResult: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="policies-heading"]');
    this.policiesList = page.locator('[data-testid="policies-list"], .policies-list');
    this.policyItems = page.locator('[data-testid="policy-item"], .policy-item');
    this.createPolicyButton = page.locator('button:has-text("Create Policy"), [data-testid="create-policy"]');
    this.searchInput = page.locator('input[type="search"], [data-testid="search-input"]');
    this.statusFilters = page.locator('[data-testid="status-filters"], .status-filters button');
    this.emptyState = page.locator('[data-testid="empty-state"], .empty-state');
    this.policyCount = page.locator('[data-testid="policy-count"], .policy-count');

    // Policy form
    this.policyForm = page.locator('[data-testid="policy-form"], form[data-type="policy"]');
    this.policyNameInput = page.locator('input[name="name"], [data-testid="policy-name"]');
    this.policyDescriptionInput = page.locator('textarea[name="description"], [data-testid="policy-description"]');
    this.policyPriorityInput = page.locator('input[name="priority"], [data-testid="policy-priority"]');
    this.policyEnabledToggle = page.locator('input[name="enabled"], [data-testid="policy-enabled"]');
    this.addConditionButton = page.locator('button:has-text("Add Condition"), [data-testid="add-condition"]');
    this.addActionButton = page.locator('button:has-text("Add Action"), [data-testid="add-action"]');
    this.savePolicyButton = page.locator('button[type="submit"]:has-text("Save"), [data-testid="save-policy"]');
    this.cancelButton = page.locator('button:has-text("Cancel")');

    // Conditions builder
    this.conditionTypeSelect = page.locator('select[name="conditionType"], [data-testid="condition-type"]');
    this.conditionOperatorSelect = page.locator('select[name="conditionOperator"], [data-testid="condition-operator"]');
    this.conditionValueInput = page.locator('input[name="conditionValue"], [data-testid="condition-value"]');
    this.removeConditionButton = page.locator('button:has-text("Remove"), [data-testid="remove-condition"]');

    // Actions builder
    this.actionTypeSelect = page.locator('select[name="actionType"], [data-testid="action-type"]');
    this.actionParameterInput = page.locator('input[name="actionParameter"], [data-testid="action-parameter"]');
    this.removeActionButton = page.locator('button:has-text("Remove"), [data-testid="remove-action"]');

    // Scope selector
    this.scopeUsersSelect = page.locator('select[name="scopeUsers"], [data-testid="scope-users"]');
    this.scopeGroupsSelect = page.locator('select[name="scopeGroups"], [data-testid="scope-groups"]');
    this.scopePrintersSelect = page.locator('select[name="scopePrinters"], [data-testid="scope-printers"]');

    // Policy detail
    this.policyDetailPage = page.locator('[data-testid="policy-detail"]');
    this.policyDetailName = page.locator('[data-testid="policy-name"], .policy-name');
    this.policyDetailConditions = page.locator('[data-testid="policy-conditions"], .policy-conditions');
    this.policyDetailActions = page.locator('[data-testid="policy-actions"], .policy-actions');
    this.policyDetailScope = page.locator('[data-testid="policy-scope"], .policy-scope');
    this.editPolicyButton = page.locator('button:has-text("Edit"), [data-testid="edit-policy"]');
    this.deletePolicyButton = page.locator('button:has-text("Delete"), [data-testid="delete-policy"]');
    this.duplicatePolicyButton = page.locator('button:has-text("Duplicate"), [data-testid="duplicate-policy"]');
    this.testPolicyButton = page.locator('button:has-text("Test"), [data-testid="test-policy"]');

    // Policy testing
    this.testPolicyModal = page.locator('[data-testid="test-policy-modal"], .test-policy-modal');
    this.testUserSelect = page.locator('select[name="testUser"], [data-testid="test-user"]');
    this.testDocumentInput = page.locator('input[name="testDocument"], [data-testid="test-document"]');
    this.testPrinterSelect = page.locator('select[name="testPrinter"], [data-testid="test-printer"]');
    this.runTestButton = page.locator('button:has-text("Run Test"), [data-testid="run-test"]');
    this.testResult = page.locator('[data-testid="test-result"], .test-result');
  }

  /**
   * Navigate to policies page
   */
  async goto() {
    await this.goto('/policies');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for policies page
   */
  async setupMocks() {
    const mockPolicies: PrintPolicy[] = [
      {
        id: 'policy-1',
        name: 'Limit Color Printing',
        description: 'Restrict color printing to management group only',
        enabled: true,
        priority: 10,
        conditions: [
          { type: 'printer', operator: 'equals', value: 'color-printer-1' },
        ],
        actions: [
          { type: 'deny', parameter: { reason: 'Color printing restricted' } },
        ],
        scope: {
          groups: ['management'],
        },
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-02-01T00:00:00Z',
      },
      {
        id: 'policy-2',
        name: 'Require Approval for Large Jobs',
        description: 'Jobs over 50 pages require manager approval',
        enabled: true,
        priority: 5,
        conditions: [
          { type: 'document', operator: 'greater_than', value: '50' },
        ],
        actions: [
          { type: 'require_approval', parameter: { approvers: ['manager-1'] } },
        ],
        scope: {
          departments: ['sales', 'marketing'],
        },
        createdAt: '2024-01-15T00:00:00Z',
        updatedAt: '2024-02-15T00:00:00Z',
      },
    ];

    // Mock policies list
    await this.page.route('**/api/v1/policies*', async (route) => {
      await mockApiResponse(route, {
        data: mockPolicies,
        total: mockPolicies.length,
      });
    });

    // Mock policy detail
    await this.page.route('**/api/v1/policies/*', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, mockPolicies[0]);
      } else if (route.request().method() === 'DELETE') {
        await mockApiResponse(route, { message: 'Policy deleted' });
      } else if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
        await mockApiResponse(route, { message: 'Policy updated' });
      }
    });

    // Mock policy creation
    await this.page.route('**/api/v1/policies', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: 'new-policy-id',
          message: 'Policy created',
        });
      }
    });

    // Mock policy test
    await this.page.route('**/api/v1/policies/test', async (route) => {
      await mockApiResponse(route, {
        matched: true,
        action: 'deny',
        reason: 'Policy matched',
      });
    });

    // Mock users/groups/printers for dropdowns
    await this.page.route('**/api/v1/users*', async (route) => {
      await mockApiResponse(route, {
        data: [
          { id: 'user-1', name: 'John Doe', email: 'john@example.com' },
          { id: 'user-2', name: 'Jane Smith', email: 'jane@example.com' },
        ],
      });
    });

    await this.page.route('**/api/v1/groups*', async (route) => {
      await mockApiResponse(route, {
        data: [
          { id: 'group-1', name: 'Management' },
          { id: 'group-2', name: 'Sales' },
        ],
      });
    });
  }

  /**
   * Verify policies page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Get policy count
   */
  async getPolicyCount(): Promise<number> {
    const countText = await this.policyCount.textContent();
    return countText ? parseInt(countText) : await this.policyItems.count();
  }

  /**
   * Filter policies by status
   */
  async filterByStatus(status: 'enabled' | 'disabled') {
    const filter = this.statusFilters.filter({ hasText: new RegExp(status, 'i') });
    await filter.click();
  }

  /**
   * Search policies
   */
  async searchPolicies(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Open create policy form
   */
  async openCreatePolicyForm() {
    await this.createPolicyButton.click();
    await expect(this.policyForm).toBeVisible();
  }

  /**
   * Fill policy form
   */
  async fillPolicyForm(data: {
    name: string;
    description?: string;
    priority?: number;
    enabled?: boolean;
  }) {
    await this.policyNameInput.fill(data.name);

    if (data.description) {
      await this.policyDescriptionInput.fill(data.description);
    }

    if (data.priority !== undefined) {
      await this.policyPriorityInput.fill(String(data.priority));
    }

    if (data.enabled !== undefined) {
      if (data.enabled) {
        await this.policyEnabledToggle.check();
      } else {
        await this.policyEnabledToggle.uncheck();
      }
    }
  }

  /**
   * Add policy condition
   */
  async addCondition(condition: Omit<PolicyCondition, 'type'> & { type: string }) {
    await this.addConditionButton.click();

    await this.conditionTypeSelect.selectOption(condition.type);
    await this.conditionOperatorSelect.selectOption(condition.operator);
    await this.conditionValueInput.fill(String(condition.value));
  }

  /**
   * Add policy action
   */
  async addAction(action: Omit<PolicyAction, 'type'> & { type: string }) {
    await this.addActionButton.click();

    await this.actionTypeSelect.selectOption(action.type);

    if (action.parameter) {
      await this.actionParameterInput.fill(JSON.stringify(action.parameter));
    }
  }

  /**
   * Set policy scope
   */
  async setScope(scope: {
    users?: string[];
    groups?: string[];
    printers?: string[];
  }) {
    if (scope.users) {
      for (const user of scope.users) {
        await this.scopeUsersSelect.selectOption(user);
      }
    }

    if (scope.groups) {
      for (const group of scope.groups) {
        await this.scopeGroupsSelect.selectOption(group);
      }
    }

    if (scope.printers) {
      for (const printer of scope.printers) {
        await this.scopePrintersSelect.selectOption(printer);
      }
    }
  }

  /**
   * Save policy
   */
  async savePolicy() {
    await this.savePolicyButton.click();
    await expect(this.policyForm).not.toBeVisible();
    await this.verifyToast('Policy saved', 'success');
  }

  /**
   * View policy details
   */
  async viewPolicyDetails(policyId: string) {
    const policyItem = this.policyItems.filter({ hasText: policyId });
    await policyItem.click();
    await expect(this.policyDetailPage).toBeVisible();
  }

  /**
   * Edit policy
   */
  async editPolicy(policyId: string) {
    await this.viewPolicyDetails(policyId);
    await this.editPolicyButton.click();
    await expect(this.policyForm).toBeVisible();
  }

  /**
   * Delete policy
   */
  async deletePolicy(policyId: string) {
    await this.viewPolicyDetails(policyId);
    await this.deletePolicyButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm"), button:has-text("Delete")');
    await confirmButton.click();

    await this.verifyToast('Policy deleted', 'success');
  }

  /**
   * Duplicate policy
   */
  async duplicatePolicy(policyId: string) {
    const policyItem = this.policyItems.filter({ hasText: policyId });
    await policyItem.locator('button:has-text("Duplicate"), [data-testid="duplicate-policy"]').click();

    await expect(this.policyForm).toBeVisible();
  }

  /**
   * Toggle policy enabled status
   */
  async togglePolicy(policyId: string) {
    const policyItem = this.policyItems.filter({ hasText: policyId });
    const toggleButton = policyItem.locator('button[role="switch"], .toggle-switch');
    await toggleButton.click();
  }

  /**
   * Open test policy modal
   */
  async openTestPolicyModal(policyId: string) {
    await this.viewPolicyDetails(policyId);
    await this.testPolicyButton.click();
    await expect(this.testPolicyModal).toBeVisible();
  }

  /**
   * Run policy test
   */
  async runPolicyTest(testData: {
    userId: string;
    document: string;
    printerId: string;
  }) {
    await this.testUserSelect.selectOption(testData.userId);
    await this.testDocumentInput.fill(testData.document);
    await this.testPrinterSelect.selectOption(testData.printerId);

    await this.runTestButton.click();

    // Wait for result
    await expect(this.testResult).toBeVisible();
  }

  /**
   * Get test result
   */
  async getTestResult(): Promise<{ matched: boolean; action?: string; reason?: string }> {
    const resultText = await this.testResult.textContent();
    // Parse result text or check data attributes
    const matched = await this.testResult.getAttribute('data-matched') === 'true';
    const action = await this.testResult.getAttribute('data-action') || undefined;
    const reason = await this.testResult.getAttribute('data-reason') || undefined;

    return { matched, action, reason };
  }

  /**
   * Reorder policies by priority
   */
  async reorderPolicies(sourceId: string, targetId: string) {
    const sourceItem = this.policyItems.filter({ hasText: sourceId });
    const targetItem = this.policyItems.filter({ hasText: targetId });

    await sourceItem.dragTo(targetItem);
  }

  /**
   * Verify policy conditions display
   */
  async verifyConditionsDisplayed(conditions: PolicyCondition[]) {
    for (const condition of conditions) {
      await expect(this.policyDetailConditions).toContainText(condition.type);
      await expect(this.policyDetailConditions).toContainText(condition.operator);
    }
  }

  /**
   * Verify policy actions display
   */
  async verifyActionsDisplayed(actions: PolicyAction[]) {
    for (const action of actions) {
      await expect(this.policyDetailActions).toContainText(action.type);
    }
  }

  /**
   * Verify policy scope display
   */
  async verifyScopeDisplayed(scope: PolicyScope) {
    const scopeElement = this.policyDetailScope;

    if (scope.users) {
      for (const user of scope.users) {
        await expect(scopeElement).toContainText(user);
      }
    }

    if (scope.groups) {
      for (const group of scope.groups) {
        await expect(scopeElement).toContainText(group);
      }
    }
  }

  /**
   * Remove condition from policy form
   */
  async removeCondition(index: number) {
    const conditions = this.policyForm.locator('[data-testid="condition-item"], .condition-item');
    await conditions.nth(index).locator(this.removeConditionButton).click();
  }

  /**
   * Remove action from policy form
   */
  async removeAction(index: number) {
    const actions = this.policyForm.locator('[data-testid="action-item"], .action-item');
    await actions.nth(index).locator(this.removeActionButton).click();
  }

  /**
   * Get policy priority
   */
  async getPolicyPriority(policyId: string): Promise<number> {
    const policyItem = this.policyItems.filter({ hasText: policyId });
    const priorityElement = policyItem.locator('[data-testid="policy-priority"], .policy-priority');
    const priorityText = await priorityElement.textContent();
    return priorityText ? parseInt(priorityText) : 0;
  }

  /**
   * Verify policy is enabled/disabled
   */
  async verifyPolicyStatus(policyId: string, enabled: boolean) {
    const policyItem = this.policyItems.filter({ hasText: policyId });
    const statusBadge = policyItem.locator('[data-testid="policy-status"], .status-badge');

    if (enabled) {
      await expect(statusBadge).toContainText('enabled');
    } else {
      await expect(statusBadge).toContainText('disabled');
    }
  }

  /**
   * Export policies
   */
  async exportPolicies(format: 'json' | 'yaml' = 'json') {
    const exportButton = this.page.locator('button:has-text("Export"), [data-testid="export-button"]');
    await exportButton.click();

    const formatOption = this.page.locator(`button:has-text("${format.toUpperCase()}")`);
    if (await formatOption.isVisible()) {
      await formatOption.click();
    }

    const downloadPromise = this.page.waitForEvent('download');
    await this.page.locator('button:has-text("Download")').click();
    await downloadPromise;
  }

  /**
   * Import policies
   */
  async importPolicies(filePath: string) {
    const importButton = this.page.locator('button:has-text("Import"), [data-testid="import-button"]');
    await importButton.click();

    const fileInput = this.page.locator('input[type="file"]');
    await fileInput.setInputFiles(filePath);

    await this.page.locator('button:has-text("Import")').click();
    await this.verifyToast('Policies imported', 'success');
  }
}
