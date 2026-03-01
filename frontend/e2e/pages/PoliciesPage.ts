import { type Page, type Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Print Policies Page Object
 * Handles print policy engine management including rule evaluation and configuration
 */
export class PoliciesPage extends BasePage {
  // Page heading and sections
  readonly heading: Locator;
  readonly overviewSection: Locator;
  readonly policiesListSection: Locator;
  readonly policyEditorSection: Locator;

  // Policy list
  readonly policyList: Locator;
  readonly policyCard: (policyId: string) => Locator;
  readonly createPolicyButton: Locator;
  readonly emptyState: Locator;

  // Policy filtering and sorting
  readonly searchInput: Locator;
  readonly statusFilter: Locator;
  readonly typeFilter: Locator;
  readonly sortBySelect: Locator;

  // Policy card elements
  readonly policyName: Locator;
  readonly policyDescription: Locator;
  readonly policyPriority: Locator;
  readonly policyStatusToggle: Locator;
  readonly editPolicyButton: (policyId: string) => Locator;
  readonly deletePolicyButton: (policyId: string) => Locator;
  readonly duplicatePolicyButton: (policyId: string) => Locator;

  // Policy form/editor
  readonly policyNameInput: Locator;
  readonly policyDescriptionInput: Locator;
  readonly policyPriorityInput: Locator;
  readonly policyEnabledToggle: Locator;
  readonly savePolicyButton: Locator;
  readonly cancelPolicyButton: Locator;

  // Policy conditions section
  readonly conditionsSection: Locator;
  readonly addConditionButton: Locator;
  readonly conditionRow: Locator;
  readonly conditionTypeSelect: Locator;
  readonly conditionOperatorSelect: Locator;
  readonly conditionValueInput: Locator;
  readonly removeConditionButton: Locator;

  // Policy actions section
  readonly actionsSection: Locator;
  readonly addActionButton: Locator;
  readonly actionRow: Locator;
  readonly actionTypeSelect: Locator;
  readonly actionValueInput: Locator;
  readonly removeActionButton: Locator;

  // Predefined condition checkboxes
  readonly colorModeCondition: Locator;
  readonly maxPagesCondition: Locator;
  readonly minPagesCondition: Locator;
  readonly userRoleCondition: Locator;
  readonly fileSizeCondition: Locator;
  readonly printerTypeCondition: Locator;
  readonly timeRangeCondition: Locator;

  // Predefined action checkboxes
  readonly forceDuplexAction: Locator;
  readonly forceGrayscaleAction: Locator;
  readonly requireApprovalAction: Locator;
  readonly blockJobAction: Locator;
  readonly setCopiesLimitAction: Locator;
  readonly setPriorityAction: Locator;
  readonly routeToPrinterAction: Locator;

  // Policy evaluation
  readonly evaluatePolicyButton: Locator;
  readonly testJobButton: Locator;
  readonly evaluationResult: Locator;
  readonly evaluationPreview: Locator;

  // Policy templates
  readonly templatesSection: Locator;
  readonly templateCard: (templateName: string) => Locator;
  readonly useTemplateButton: Locator;

  // Policy history
  readonly historyTab: Locator;
  readonly historyList: Locator;
  readonly restorePolicyButton: Locator;

  // Policy import/export
  readonly exportPolicyButton: Locator;
  readonly importPolicyButton: Locator;
  readonly importFileInput: Locator;

  // Confirmation modals
  readonly deleteConfirmModal: Locator;
  readonly deleteConfirmButton: Locator;
  readonly deleteCancelButton: Locator;

  constructor(page: Page) {
    super(page);

    // Page heading and sections
    this.heading = page.getByRole('heading', { name: /print policies|policies/i });
    this.overviewSection = page.locator('[data-testid="policies-overview"], section:has-text("Overview")');
    this.policiesListSection = page.locator('[data-testid="policies-list"], section:has-text("Policies")');
    this.policyEditorSection = page.locator('[data-testid="policy-editor"], .policy-editor');

    // Policy list
    this.policyList = page.locator('[data-testid="policy-list"], .policy-list');
    this.policyCard = (policyId: string) =>
      page.locator(`[data-policy-id="${policyId}"]`).or(
        page.locator('[data-testid="policy-card"]').filter({ hasText: policyId })
      );
    this.createPolicyButton = page.getByRole('button', { name: /create policy|add policy|new policy/i });
    this.emptyState = page.getByText(/no policies|no policies configured/i);

    // Filtering and sorting
    this.searchInput = page.getByPlaceholder(/search|find policy/i);
    this.statusFilter = page.getByLabel(/status|enabled/i);
    this.typeFilter = page.getByLabel(/type|category/i);
    this.sortBySelect = page.getByLabel(/sort by/i);

    // Policy card elements
    this.policyName = page.locator('[data-testid="policy-name"], .policy-name');
    this.policyDescription = page.locator('[data-testid="policy-description"], .policy-description');
    this.policyPriority = page.locator('[data-testid="policy-priority"], .policy-priority');
    this.policyStatusToggle = page.locator('[data-testid="policy-toggle"], .policy-toggle');
    this.editPolicyButton = (policyId: string) =>
      this.policyCard(policyId).getByRole('button', { name: /edit/i });
    this.deletePolicyButton = (policyId: string) =>
      this.policyCard(policyId).getByRole('button', { name: /delete/i });
    this.duplicatePolicyButton = (policyId: string) =>
      this.policyCard(policyId).getByRole('button', { name: /duplicate|copy/i });

    // Policy form/editor
    this.policyNameInput = page.getByLabel(/policy name|name/i);
    this.policyDescriptionInput = page.getByLabel(/description/i);
    this.policyPriorityInput = page.getByLabel(/priority/i);
    this.policyEnabledToggle = page.getByRole('switch', { name: /enabled|active/i });
    this.savePolicyButton = page.getByRole('button', { name: /save|create|update/i });
    this.cancelPolicyButton = page.getByRole('button', { name: /cancel/i });

    // Policy conditions
    this.conditionsSection = page.locator('[data-testid="conditions-section"], section:has-text("Conditions")');
    this.addConditionButton = page.getByRole('button', { name: /add condition/i });
    this.conditionRow = page.locator('[data-testid="condition-row"]');
    this.conditionTypeSelect = page.getByLabel(/condition type|when/i);
    this.conditionOperatorSelect = page.getByLabel(/operator|is/i);
    this.conditionValueInput = page.getByLabel(/value/i);
    this.removeConditionButton = page.locator('[data-testid="remove-condition"]');

    // Policy actions
    this.actionsSection = page.locator('[data-testid="actions-section"], section:has-text("Actions")');
    this.addActionButton = page.getByRole('button', { name: /add action/i });
    this.actionRow = page.locator('[data-testid="action-row"]');
    this.actionTypeSelect = page.getByLabel(/action type|then/i);
    this.actionValueInput = page.getByLabel(/action value/i);
    this.removeActionButton = page.locator('[data-testid="remove-action"]');

    // Predefined condition checkboxes
    this.colorModeCondition = page.getByLabel(/color mode|color/i);
    this.maxPagesCondition = page.getByLabel(/max pages/i);
    this.minPagesCondition = page.getByLabel(/min pages/i);
    this.userRoleCondition = page.getByLabel(/user role|role/i);
    this.fileSizeCondition = page.getByLabel(/file size/i);
    this.printerTypeCondition = page.getByLabel(/printer type/i);
    this.timeRangeCondition = page.getByLabel(/time range|schedule/i);

    // Predefined action checkboxes
    this.forceDuplexAction = page.getByLabel(/force duplex/i);
    this.forceGrayscaleAction = page.getByLabel(/force grayscale/i);
    this.requireApprovalAction = page.getByLabel(/require approval/i);
    this.blockJobAction = page.getByLabel(/block job/i);
    this.setCopiesLimitAction = page.getByLabel(/set copies limit/i);
    this.setPriorityAction = page.getByLabel(/set priority/i);
    this.routeToPrinterAction = page.getByLabel(/route to printer/i);

    // Policy evaluation
    this.evaluatePolicyButton = page.getByRole('button', { name: /evaluate|test policy/i });
    this.testJobButton = page.getByRole('button', { name: /test job|preview/i });
    this.evaluationResult = page.locator('[data-testid="evaluation-result"]');
    this.evaluationPreview = page.locator('[data-testid="evaluation-preview"]');

    // Policy templates
    this.templatesSection = page.locator('[data-testid="policy-templates"], section:has-text("Templates")');
    this.templateCard = (templateName: string) =>
      page.locator('[data-testid="template-card"]').filter({ hasText: templateName });
    this.useTemplateButton = page.getByRole('button', { name: /use template|apply template/i });

    // Policy history
    this.historyTab = page.getByRole('tab', { name: /history|version/i });
    this.historyList = page.locator('[data-testid="policy-history"]');
    this.restorePolicyButton = page.getByRole('button', { name: /restore|revert/i });

    // Import/export
    this.exportPolicyButton = page.getByRole('button', { name: /export/i });
    this.importPolicyButton = page.getByRole('button', { name: /import/i });
    this.importFileInput = page.locator('input[type="file"]');

    // Confirmation modals
    this.deleteConfirmModal = page.locator('[data-testid="delete-modal"], .modal:has-text("delete")');
    this.deleteConfirmButton = page.getByRole('button', { name: /delete|confirm/i }).filter({ hasText: /delete/i });
    this.deleteCancelButton = page.getByRole('button', { name: /cancel/i });
  }

  /**
   * Navigate to Policies page
   */
  async navigate(): Promise<void> {
    await this.goto('/policies');
  }

  /**
   * Verify Policies page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.page.waitForLoadState('networkidle');
    return await this.heading.isVisible();
  }

  /**
   * Get policy count
   */
  async getPolicyCount(): Promise<number> {
    return await this.policyList.locator('[data-testid="policy-card"], .policy-card').count();
  }

  /**
   * Check if policies list is empty
   */
  async isEmpty(): Promise<boolean> {
    return await this.emptyState.isVisible();
  }

  /**
   * Open create policy modal/form
   */
  async openCreatePolicy(): Promise<void> {
    await this.createPolicyButton.click();
    await expect(this.policyEditorSection).toBeVisible();
  }

  /**
   * Create a new policy
   */
  async createPolicy(policy: {
    name: string;
    description: string;
    priority: number;
    enabled: boolean;
  }): Promise<void> {
    await this.openCreatePolicy();
    await this.policyNameInput.fill(policy.name);
    await this.policyDescriptionInput.fill(policy.description);
    await this.policyPriorityInput.fill(policy.priority.toString());
    if (policy.enabled) {
      await this.policyEnabledToggle.check();
    } else {
      await this.policyEnabledToggle.uncheck();
    }
    await this.savePolicyButton.click();
  }

  /**
   * Edit existing policy
   */
  async editPolicy(policyId: string): Promise<void> {
    await this.editPolicyButton(policyId).click();
    await expect(this.policyEditorSection).toBeVisible();
  }

  /**
   * Delete policy
   */
  async deletePolicy(policyId: string): Promise<void> {
    await this.deletePolicyButton(policyId).click();
    await this.deleteConfirmButton.click();
  }

  /**
   * Duplicate policy
   */
  async duplicatePolicy(policyId: string): Promise<void> {
    await this.duplicatePolicyButton(policyId).click();
  }

  /**
   * Toggle policy enabled status
   */
  async togglePolicyStatus(policyId: string): Promise<void> {
    const card = this.policyCard(policyId);
    await card.locator('[data-testid="policy-toggle"], .policy-toggle').click();
  }

  /**
   * Check if policy is enabled
   */
  async isPolicyEnabled(policyId: string): Promise<boolean> {
    const card = this.policyCard(policyId);
    const toggle = card.locator('[data-testid="policy-toggle"], .policy-toggle');
    const isChecked = await toggle.getAttribute('data-checked');
    return isChecked === 'true';
  }

  /**
   * Search for policies
   */
  async searchPolicies(query: string): Promise<void> {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Filter policies by status
   */
  async filterByStatus(status: 'enabled' | 'disabled' | 'all'): Promise<void> {
    await this.statusFilter.selectOption(status);
  }

  /**
   * Filter policies by type
   */
  async filterByType(type: string): Promise<void> {
    await this.typeFilter.selectOption(type);
  }

  /**
   * Sort policies
   */
  async sortPolicies(by: 'priority' | 'name' | 'created'): Promise<void> {
    await this.sortBySelect.selectOption(by);
  }

  /**
   * Add condition to policy
   */
  async addCondition(condition: {
    type: string;
    operator: string;
    value: string;
  }): Promise<void> {
    await this.addConditionButton.click();
    await this.conditionTypeSelect.last().selectOption(condition.type);
    await this.conditionOperatorSelect.last().selectOption(condition.operator);
    await this.conditionValueInput.last().fill(condition.value);
  }

  /**
   * Remove condition from policy
   */
  async removeCondition(index: number): Promise<void> {
    await this.conditionRow.nth(index).locator('[data-testid="remove-condition"], button:has-text("Remove")').click();
  }

  /**
   * Add action to policy
   */
  async addAction(action: {
    type: string;
    value?: string;
  }): Promise<void> {
    await this.addActionButton.click();
    await this.actionTypeSelect.last().selectOption(action.type);
    if (action.value) {
      await this.actionValueInput.last().fill(action.value);
    }
  }

  /**
   * Remove action from policy
   */
  async removeAction(index: number): Promise<void> {
    await this.actionRow.nth(index).locator('[data-testid="remove-action"], button:has-text("Remove")').click();
  }

  /**
   * Set force duplex action
   */
  async setForceDuplex(enabled: boolean): Promise<void> {
    if (enabled) {
      await this.forceDuplexAction.check();
    } else {
      await this.forceDuplexAction.uncheck();
    }
  }

  /**
   * Set force grayscale action
   */
  async setForceGrayscale(enabled: boolean): Promise<void> {
    if (enabled) {
      await this.forceGrayscaleAction.check();
    } else {
      await this.forceGrayscaleAction.uncheck();
    }
  }

  /**
   * Set require approval action
   */
  async setRequireApproval(enabled: boolean): Promise<void> {
    if (enabled) {
      await this.requireApprovalAction.check();
    } else {
      await this.requireApprovalAction.uncheck();
    }
  }

  /**
   * Use policy template
   */
  async useTemplate(templateName: string): Promise<void> {
    await this.templateCard(templateName).getByRole('button', { name: /use|apply/i }).click();
  }

  /**
   * Evaluate policy against test job
   */
  async evaluatePolicy(testJob: {
    pages: number;
    colorMode: string;
    userRole: string;
  }): Promise<void> {
    await this.evaluatePolicyButton.click();
    // Fill test job details
    await this.page.getByLabel(/pages/i).fill(testJob.pages.toString());
    await this.page.getByLabel(/color/i).selectOption(testJob.colorMode);
    await this.page.getByLabel(/role/i).selectOption(testJob.userRole);
    await this.testJobButton.click();
  }

  /**
   * Get evaluation result
   */
  async getEvaluationResult(): Promise<string> {
    return await this.evaluationResult.textContent() || '';
  }

  /**
   * Export policy
   */
  async exportPolicy(policyId: string): Promise<void> {
    await this.editPolicy(policyId);
    const downloadPromise = this.page.waitForEvent('download');
    await this.exportPolicyButton.click();
    await downloadPromise;
  }

  /**
   * Import policy
   */
  async importPolicy(filePath: string): Promise<void> {
    await this.importPolicyButton.click();
    await this.importFileInput.setInputFiles(filePath);
    await this.page.getByRole('button', { name: /import|upload/i }).click();
  }

  /**
   * View policy history
   */
  async viewHistory(policyId: string): Promise<void> {
    await this.editPolicy(policyId);
    await this.historyTab.click();
  }

  /**
   * Restore policy from history
   */
  async restorePolicy(versionId: string): Promise<void> {
    await this.historyList.locator(`[data-version-id="${versionId}"]`)
      .getByRole('button', { name: /restore/i }).click();
  }

  /**
   * Get policy details
   */
  async getPolicyDetails(policyId: string): Promise<{
    name: string;
    description: string;
    priority: number;
    enabled: boolean;
  }> {
    const card = this.policyCard(policyId);
    return {
      name: await card.locator('[data-testid="policy-name"]').textContent() || '',
      description: await card.locator('[data-testid="policy-description"]').textContent() || '',
      priority: parseInt(await card.locator('[data-testid="policy-priority"]').textContent() || '0'),
      enabled: await this.isPolicyEnabled(policyId),
    };
  }

  /**
   * Verify policy exists
   */
  async hasPolicy(policyId: string): Promise<boolean> {
    const count = await this.policyCard(policyId).count();
    return count > 0;
  }

  /**
   * Cancel policy editing
   */
  async cancelPolicyEdit(): Promise<void> {
    await this.cancelPolicyButton.click();
  }

  /**
   * Get conditions count
   */
  async getConditionsCount(): Promise<number> {
    return await this.conditionRow.count();
  }

  /**
   * Get actions count
   */
  async getActionsCount(): Promise<number> {
    return await this.actionRow.count();
  }

  /**
   * Verify policy form is visible
   */
  async isPolicyFormVisible(): Promise<boolean> {
    return await this.policyEditorSection.isVisible();
  }

  /**
   * Verify policy is selected
   */
  async isPolicySelected(policyId: string): Promise<boolean> {
    const card = this.policyCard(policyId);
    const selectedClass = await card.getAttribute('class');
    return selectedClass?.includes('selected') || false;
  }

  /**
   * Select policy
   */
  async selectPolicy(policyId: string): Promise<void> {
    const card = this.policyCard(policyId);
    await card.click();
  }

  /**
   * Get policy priority
   */
  async getPolicyPriority(policyId: string): Promise<number> {
    const card = this.policyCard(policyId);
    const priorityText = await card.locator('[data-testid="policy-priority"]').textContent() || '';
    return parseInt(priorityText.match(/\d+/)?.[0] || '0');
  }

  /**
   * Reorder policies (drag and drop)
   */
  async reorderPolicy(sourceIndex: number, targetIndex: number): Promise<void> {
    const policies = await this.policyList.locator('[data-testid="policy-card"]').all();
    const source = policies[sourceIndex];
    const target = policies[targetIndex];

    await source.dragTo(target);
  }
}
