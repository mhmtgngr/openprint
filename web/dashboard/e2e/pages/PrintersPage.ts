/**
 * Printers Page Object
 * Handles printer discovery, management, and configuration
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse, mockPrinters, mockDiscoveredPrinters } from '../helpers';

export class PrintersPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly printersList: Locator;
  readonly printerItems: Locator;
  readonly addPrinterButton: Locator;
  readonly discoverPrintersButton: Locator;
  readonly searchInput: Locator;
  readonly typeFilters: Locator;
  readonly statusFilters: Locator;
  readonly emptyState: Locator;

  // Printer detail
  readonly printerDetailPage: Locator;
  readonly printerDetailName: Locator;
  readonly printerDetailStatus: Locator;
  readonly printerDetailType: Locator;
  readonly printerDetailCapabilities: Locator;
  readonly printerDetailAgent: Locator;
  readonly editPrinterButton: Locator;
  readonly deletePrinterButton: Locator;
  readonly testPrintButton: Locator;

  // Discovery modal
  readonly discoveryModal: Locator;
  readonly discoveryResults: Locator;
  readonly discoveryAddButton: Locator;
  readonly discoveryRefreshButton: Locator;
  readonly discoveryAgentSelect: Locator;

  // Printer configuration form
  readonly configForm: Locator;
  readonly printerNameInput: Locator;
  readonly printerTypeSelect: Locator;
  readonly agentSelect: Locator;
  readonly driverInput: Locator;
  readonly portInput: Locator;
  readonly saveConfigButton: Locator;

  // Agent installation notice
  readonly agentInstallNotice: Locator;
  readonly downloadAgentButton: Locator;
  readonly copyInstallCommandButton: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="printers-heading"]');
    this.printersList = page.locator('[data-testid="printers-list"], .printers-list');
    this.printerItems = page.locator('[data-testid="printer-item"], .printer-item');
    this.addPrinterButton = page.locator('button:has-text("Add Printer"), [data-testid="add-printer-button"]');
    this.discoverPrintersButton = page.locator('button:has-text("Discover"), [data-testid="discover-printers-button"]');
    this.searchInput = page.locator('input[type="search"], [data-testid="search-input"]');
    this.typeFilters = page.locator('[data-testid="type-filters"], .type-filters button');
    this.statusFilters = page.locator('[data-testid="status-filters"], .status-filters button');
    this.emptyState = page.locator('[data-testid="empty-state"], .empty-state');

    // Printer detail
    this.printerDetailPage = page.locator('[data-testid="printer-detail"]');
    this.printerDetailName = page.locator('[data-testid="printer-name"], .printer-name');
    this.printerDetailStatus = page.locator('[data-testid="printer-status"], .printer-status');
    this.printerDetailType = page.locator('[data-testid="printer-type"], .printer-type');
    this.printerDetailCapabilities = page.locator('[data-testid="printer-capabilities"], .printer-capabilities');
    this.printerDetailAgent = page.locator('[data-testid="printer-agent"], .printer-agent');
    this.editPrinterButton = page.locator('button:has-text("Edit"), [data-testid="edit-printer"]');
    this.deletePrinterButton = page.locator('button:has-text("Delete"), [data-testid="delete-printer"]');
    this.testPrintButton = page.locator('button:has-text("Test Print"), [data-testid="test-print"]');

    // Discovery
    this.discoveryModal = page.locator('[data-testid="discovery-modal"], .discovery-modal');
    this.discoveryResults = page.locator('[data-testid="discovery-results"], .discovery-results');
    this.discoveryAddButton = page.locator('button:has-text("Add"), [data-testid="add-discovered-printer"]');
    this.discoveryRefreshButton = page.locator('button:has-text("Refresh"), [data-testid="refresh-discovery"]');
    this.discoveryAgentSelect = page.locator('select[name="agent"], [data-testid="discovery-agent-select"]');

    // Configuration form
    this.configForm = page.locator('[data-testid="printer-config-form"], form[data-type="printer-config"]');
    this.printerNameInput = page.locator('input[name="name"], [data-testid="printer-name-input"]');
    this.printerTypeSelect = page.locator('select[name="type"], [data-testid="printer-type-select"]');
    this.agentSelect = page.locator('select[name="agent"], [data-testid="agent-select"]');
    this.driverInput = page.locator('input[name="driver"], [data-testid="driver-input"]');
    this.portInput = page.locator('input[name="port"], [data-testid="port-input"]');
    this.saveConfigButton = page.locator('button[type="submit"]:has-text("Save"), [data-testid="save-config"]');

    // Agent notice
    this.agentInstallNotice = page.locator('[data-testid="agent-install-notice"], .agent-install-notice');
    this.downloadAgentButton = page.locator('button:has-text("Download Agent"), [data-testid="download-agent"]');
    this.copyInstallCommandButton = page.locator('button:has-text("Copy Command"), [data-testid="copy-command"]');
  }

  /**
   * Navigate to printers page
   */
  async goto() {
    await this.goto('/printers');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for printers page
   */
  async setupMocks() {
    // Mock printers list
    await this.page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    // Mock discovered printers
    await this.page.route('**/api/v1/discovered-printers*', async (route) => {
      await mockApiResponse(route, {
        printers: mockDiscoveredPrinters,
        total: mockDiscoveredPrinters.length,
      });
    });

    // Mock printer detail
    await this.page.route('**/api/v1/printers/*', async (route) => {
      const id = route.request().url().split('/').pop();
      const printer = mockPrinters.find((p) => p.id === id) || mockPrinters[0];
      await mockApiResponse(route, printer);
    });

    // Mock printer update
    await this.page.route('**/api/v1/printers/*', async (route) => {
      if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
        await mockApiResponse(route, { message: 'Printer updated successfully' });
      }
    });

    // Mock printer deletion
    await this.page.route('**/api/v1/printers/*', async (route) => {
      if (route.request().method() === 'DELETE') {
        await mockApiResponse(route, { message: 'Printer deleted successfully' });
      }
    });

    // Mock printer toggle
    await this.page.route('**/api/v1/printers/*/toggle', async (route) => {
      await mockApiResponse(route, { isActive: true, isOnline: true });
    });

    // Mock test print
    await this.page.route('**/api/v1/printers/*/test-print', async (route) => {
      await mockApiResponse(route, {
        jobId: 'test-job-id',
        message: 'Test print job created',
      });
    });

    // Mock agent list
    await this.page.route('**/api/v1/agents*', async (route) => {
      await mockApiResponse(route, [
        {
          id: 'agent-1',
          name: 'WORKSTATION-001',
          status: 'online',
          platform: 'windows',
        },
        {
          id: 'agent-2',
          name: 'WORKSTATION-002',
          status: 'online',
          platform: 'windows',
        },
      ]);
    });
  }

  /**
   * Verify printers page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Get printer count from list
   */
  async getPrinterCount(): Promise<number> {
    return await this.printerItems.count();
  }

  /**
   * Search for printers
   */
  async searchPrinters(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500); // Wait for debounce
  }

  /**
   * Filter printers by type
   */
  async filterByType(type: string) {
    const filter = this.typeFilters.filter({ hasText: type });
    await filter.click();
  }

  /**
   * Filter printers by status
   */
  async filterByStatus(status: string) {
    const filter = this.statusFilters.filter({ hasText: status });
    await filter.click();
  }

  /**
   * Click on a printer to view details
   */
  async viewPrinterDetails(printerName: string) {
    const printerItem = this.printerItems.filter({ hasText: printerName }).first();
    await printerItem.click();
    await expect(this.printerDetailPage).toBeVisible();
  }

  /**
   * Toggle printer status (active/inactive)
   */
  async togglePrinterStatus(printerName: string) {
    const printerItem = this.printerItems.filter({ hasText: printerName });
    const toggleButton = printerItem.locator('button[role="switch"], .toggle-switch');
    await toggleButton.click();
  }

  /**
   * Open discovery modal
   */
  async openDiscoveryModal() {
    await this.discoverPrintersButton.click();
    await expect(this.discoveryModal).toBeVisible();
  }

  /**
   * Run printer discovery
   */
  async runDiscovery(agentId?: string) {
    await this.openDiscoveryModal();

    if (agentId) {
      await this.discoveryAgentSelect.selectOption(agentId);
    }

    await this.discoveryRefreshButton.click();

    // Wait for results
    await this.discoveryResults.waitFor({ state: 'visible' });
  }

  /**
   * Add discovered printer
   */
  async addDiscoveredPrinter(printerName: string) {
    const printerRow = this.discoveryResults.locator(`tr:has-text("${printerName}")`);
    const addButton = printerRow.locator('button:has-text("Add")');
    await addButton.click();
  }

  /**
   * Close discovery modal
   */
  async closeDiscoveryModal() {
    const closeButton = this.discoveryModal.locator('button[aria-label="Close"], .modal-close');
    await closeButton.click();
  }

  /**
   * Open printer configuration form
   */
  async openConfigForm() {
    await this.addPrinterButton.click();
    await expect(this.configForm).toBeVisible();
  }

  /**
   * Fill printer configuration form
   */
  async fillConfigForm(data: {
    name: string;
    type: string;
    agent?: string;
    driver?: string;
    port?: string;
  }) {
    await this.printerNameInput.fill(data.name);
    await this.printerTypeSelect.selectOption(data.type);

    if (data.agent) {
      await this.agentSelect.selectOption(data.agent);
    }

    if (data.driver) {
      await this.driverInput.fill(data.driver);
    }

    if (data.port) {
      await this.portInput.fill(data.port);
    }
  }

  /**
   * Save printer configuration
   */
  async saveConfig() {
    await this.saveConfigButton.click();
    await expect(this.configForm).not.toBeVisible();
  }

  /**
   * Edit printer
   */
  async editPrinter(printerName: string) {
    await this.viewPrinterDetails(printerName);
    await this.editPrinterButton.click();
    await expect(this.configForm).toBeVisible();
  }

  /**
   * Delete printer
   */
  async deletePrinter(printerName: string) {
    await this.viewPrinterDetails(printerName);
    await this.deletePrinterButton.click();

    // Confirm deletion
    const confirmButton = this.page.locator('button:has-text("Confirm"), button:has-text("Delete")');
    await confirmButton.click();
  }

  /**
   * Run test print
   */
  async runTestPrint(printerName: string) {
    await this.viewPrinterDetails(printerName);
    await this.testPrintButton.click();

    // Wait for success message
    await this.verifyToast('Test print sent', 'success');
  }

  /**
   * Verify printer capabilities
   */
  async verifyPrinterCapabilities(printerName: string, capabilities: {
    supportsColor?: boolean;
    supportsDuplex?: boolean;
    paperSizes?: string[];
  }) {
    await this.viewPrinterDetails(printerName);

    const capabilitiesElement = this.printerDetailCapabilities;

    if (capabilities.supportsColor) {
      await expect(capabilitiesElement).toContainText('Color');
    }

    if (capabilities.supportsDuplex) {
      await expect(capabilitiesElement).toContainText('Duplex');
    }

    if (capabilities.paperSizes) {
      for (const size of capabilities.paperSizes) {
        await expect(capabilitiesElement).toContainText(size);
      }
    }
  }

  /**
   * Verify printer status
   */
  async verifyPrinterStatus(printerName: string, status: 'online' | 'offline' | 'error') {
    const printerItem = this.printerItems.filter({ hasText: printerName });
    const statusBadge = printerItem.locator('[data-testid="printer-status"], .status-badge');
    await expect(statusBadge).toHaveText(status);
  }

  /**
   * Verify empty state
   */
  async verifyEmptyState() {
    await expect(this.emptyState).toBeVisible();
    await expect(this.printerItems).toHaveCount(0);
  }

  /**
   * Verify agent installation notice
   */
  async verifyAgentInstallNotice() {
    await expect(this.agentInstallNotice).toBeVisible();
  }

  /**
   * Copy agent install command
   */
  async copyAgentInstallCommand() {
    await this.copyInstallCommandButton.click();
    await this.verifyToast('Copied to clipboard', 'success');
  }

  /**
   * Download agent
   */
  async downloadAgent() {
    const downloadPromise = this.page.waitForEvent('download');
    await this.downloadAgentButton.click();
    await downloadPromise;
  }

  /**
   * Get printer health status
   */
  async getPrinterHealth(printerName: string): Promise<string> {
    const printerItem = this.printerItems.filter({ hasText: printerName });
    const healthIndicator = printerItem.locator('[data-testid="health-indicator"], .health');
    return await healthIndicator.getAttribute('data-health') || 'unknown';
  }

  /**
   * Verify printer type badge
   */
  async verifyPrinterType(printerName: string, type: string) {
    const printerItem = this.printerItems.filter({ hasText: printerName });
    const typeBadge = printerItem.locator('[data-testid="printer-type"], .type-badge');
    await expect(typeBadge).toHaveText(type);
  }

  /**
   * Get all printer names from list
   */
  async getPrinterNames(): Promise<string[]> {
    const names: string[] = [];
    const count = await this.printerItems.count();

    for (let i = 0; i < count; i++) {
      const name = await this.printerItems.nth(i)
        .locator('[data-testid="printer-name"], .printer-name')
        .textContent();
      if (name) names.push(name);
    }

    return names;
  }

  /**
   * Sort printers by name
   */
  async sortByName() {
    const nameHeader = this.page.locator('th:has-text("Name"), [data-testid="sort-name"]');
    await nameHeader.click();
  }

  /**
   * Verify printer connectivity status
   */
  async verifyPrinterConnectivity(printerName: string, isConnected: boolean) {
    const printerItem = this.printerItems.filter({ hasText: printerName });
    const connectivityIndicator = printerItem.locator('[data-testid="connectivity"], .connectivity');

    if (isConnected) {
      await expect(connectivityIndicator).toHaveAttribute('data-status', 'connected');
    } else {
      await expect(connectivityIndicator).toHaveAttribute('data-status', 'disconnected');
    }
  }
}
