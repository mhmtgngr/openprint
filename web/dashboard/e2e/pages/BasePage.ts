/**
 * Base Page Object Model class
 * Provides common navigation and authentication methods for all pages
 */
import { Page, Locator, expect } from '@playwright/test';
import { mockApiResponse, mockUsers } from '../helpers';

export class BasePage {
  readonly page: Page;
  readonly baseURL: string;

  // Common locators
  readonly sidebar: Locator;
  readonly logoutButton: Locator;
  readonly userInfo: Locator;
  readonly navigationLinks: Locator;
  readonly spinner: Locator;

  constructor(page: Page, baseURL: string = 'http://localhost:3000') {
    this.page = page;
    this.baseURL = baseURL;

    // Initialize common locators
    this.sidebar = page.locator('nav aside, [data-testid="sidebar"], .sidebar');
    this.logoutButton = page.locator('button:has-text("Logout"), [data-testid="logout-button"]');
    this.userInfo = page.locator('[data-testid="user-info"], .user-info');
    this.navigationLinks = page.locator('nav a, [data-testid="nav-link"]');
    this.spinner = page.locator('.spinner, [data-testid="spinner"], .loading');
  }

  /**
   * Navigate to a specific path
   */
  async goto(path: string = '/', options?: { waitUntil?: 'load' | 'domcontentloaded' | 'networkidle' }) {
    await this.page.goto(`${this.baseURL}${path}`, options);
  }

  /**
   * Navigate to dashboard
   */
  async gotoDashboard() {
    await this.goto('/dashboard');
  }

  /**
   * Navigate to a specific page using sidebar navigation
   */
  async navigateTo(pageName: string) {
    const link = this.navigationLinks.filter({ hasText: pageName }).first();
    await link.click();
  }

  /**
   * Wait for page to be loaded (no spinners)
   */
  async waitForPageLoad() {
    await this.page.waitForLoadState('networkidle');
    await expect(this.spinner).not.toBeVisible({ timeout: 5000 }).catch(() => {
      // Spinner might not exist, that's okay
    });
  }

  /**
   * Wait for API call to complete
   */
  async waitForApiCall(apiPattern: string) {
    await this.page.waitForResponse(
      (response) => response.url().includes(apiPattern) && response.status() === 200
    );
  }

  /**
   * Set up authenticated state
   * Mocks auth endpoints and sets tokens in localStorage
   */
  async setupAuth(user: typeof mockUsers[number] = mockUsers[0]) {
    // Mock auth endpoints
    await this.page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, user);
    });

    await this.page.route('**/api/v1/auth/login', async (route) => {
      await mockApiResponse(route, {
        userId: user.id,
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        org: { id: 'org-1', name: 'Test Org' },
      });
    });

    await this.page.route('**/api/v1/auth/refresh', async (route) => {
      await mockApiResponse(route, {
        access_token: 'mock-refreshed-access-token',
        refresh_token: 'mock-refresh-token',
      });
    });

    // Set tokens in localStorage
    await this.page.addInitScript((tokens) => {
      localStorage.setItem('auth_tokens', JSON.stringify(tokens));
    }, {
      accessToken: 'mock-access-token',
      refreshToken: 'mock-refresh-token',
    });
  }

  /**
   * Login with credentials using the login form
   */
  async login(email: string, password: string) {
    await this.goto('/login');

    // Wait for form to be ready
    await this.page.waitForSelector('input[type="email"]', { state: 'visible' });

    // Fill in credentials
    await this.page.fill('input[type="email"]', email);
    await this.page.fill('input[type="password"]', password);

    // Submit form
    await this.page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await this.page.waitForURL('**/dashboard', { timeout: 10000 });
  }

  /**
   * Login as admin user
   */
  async loginAsAdmin() {
    await this.login('admin@example.com', 'AdminPassword123!');
  }

  /**
   * Login as regular user
   */
  async loginAsUser() {
    await this.login('test@example.com', 'TestPassword123!');
  }

  /**
   * Logout from the application
   */
  async logout() {
    await this.logoutButton.click();
    await this.page.waitForURL('**/login');
  }

  /**
   * Check if user is authenticated
   */
  async isAuthenticated(): Promise<boolean> {
    const tokens = await this.page.evaluate(() => {
      const authTokens = localStorage.getItem('auth_tokens');
      return authTokens !== null;
    });
    return tokens;
  }

  /**
   * Get current user info from localStorage
   */
  async getCurrentUser() {
    return await this.page.evaluate(() => {
      const authTokens = localStorage.getItem('auth_tokens');
      if (authTokens) {
        return JSON.parse(authTokens);
      }
      return null;
    });
  }

  /**
   * Mock API response for a specific pattern
   */
  async mockApi(pattern: string, data: unknown, status: number = 200) {
    await this.page.route(pattern, async (route) => {
      await mockApiResponse(route, data, status);
    });
  }

  /**
   * Wait for and verify toast/notification message
   */
  async verifyToast(message: string, type: 'success' | 'error' | 'info' | 'warning' = 'info') {
    const toast = this.page.locator(
      `[data-testid="toast"], .toast, [role="alert"]`
    ).filter({ hasText: message });

    await expect(toast).toBeVisible();

    // Verify type if specified
    if (type) {
      await expect(toast).toHaveAttribute('data-type', type);
    }
  }

  /**
   * Take screenshot for debugging/visual regression
   */
  async screenshot(options?: { path?: string; fullPage?: boolean }) {
    await this.page.screenshot({
      fullPage: options?.fullPage ?? true,
      path: options?.path,
    });
  }

  /**
   * Wait for element to be visible
   */
  async waitForVisible(selector: string, timeout: number = 5000) {
    await this.page.waitForSelector(selector, { state: 'visible', timeout });
  }

  /**
   * Wait for element to be hidden
   */
  async waitForHidden(selector: string, timeout: number = 5000) {
    await this.page.waitForSelector(selector, { state: 'hidden', timeout });
  }

  /**
   * Click element and wait for navigation
   */
  async clickAndWaitForNavigation(selector: string) {
    await Promise.all([
      this.page.waitForURL(),
      this.page.click(selector),
    ]);
  }

  /**
   * Fill form field by label
   */
  async fillByLabel(label: string, value: string) {
    await this.page.getByLabel(label).fill(value);
  }

  /**
   * Check if element exists
   */
  async exists(selector: string): Promise<boolean> {
    return await this.page.locator(selector).count() > 0;
  }

  /**
   * Get text content of element
   */
  async getText(selector: string): Promise<string> {
    return await this.page.locator(selector).textContent() || '';
  }

  /**
   * Reload current page
   */
  async reload() {
    await this.page.reload();
  }

  /**
   * Go back in browser history
   */
  async back() {
    await this.page.goBack();
  }

  /**
   * Verify current URL contains expected path
   */
  async verifyUrl(path: string) {
    await this.page.waitForURL(`**${path}`);
    expect(this.page.url()).toContain(path);
  }

  /**
   * Get current URL
   */
  getCurrentUrl(): string {
    return this.page.url();
  }

  /**
   * Set viewport size (useful for mobile testing)
   */
  async setViewport(width: number, height: number) {
    await this.page.setViewportSize({ width, height });
  }

  /**
   * Hover over element
   */
  async hover(selector: string) {
    await this.page.locator(selector).hover();
  }

  /**
   * Scroll to element
   */
  async scrollToElement(selector: string) {
    await this.page.locator(selector).scrollIntoViewIfNeeded();
  }

  /**
   * Press keyboard key
   */
  async press(key: string) {
    await this.page.keyboard.press(key);
  }

  /**
   * Upload file
   */
  async uploadFile(selector: string, filePath: string) {
    await this.page.locator(selector).setInputFiles(filePath);
  }

  /**
   * Select option from dropdown
   */
  async selectOption(selector: string, value: string) {
    await this.page.selectOption(selector, value);
  }

  /**
   * Check checkbox
   */
  async checkCheckbox(selector: string) {
    await this.page.check(selector);
  }

  /**
   * Uncheck checkbox
   */
  async uncheckCheckbox(selector: string) {
    await this.page.uncheck(selector);
  }

  /**
   * Verify page title
   */
  async verifyPageTitle(title: string) {
    await expect(this.page).toHaveTitle(new RegExp(title, 'i'));
  }

  /**
   * Verify breadcrumb contains expected text
   */
  async verifyBreadcrumb(text: string) {
    const breadcrumb = this.page.locator('[data-testid="breadcrumb"], .breadcrumb, nav[aria-label="breadcrumb"]');
    await expect(breadcrumb).toContainText(text);
  }

  /**
   * Mock common dashboard APIs
   */
  async mockCommonAPIs() {
    // Mock printers
    await this.page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, {
        printers: [
          {
            id: 'printer-1',
            name: 'HP LaserJet Pro',
            isActive: true,
            isOnline: true,
          },
          {
            id: 'printer-2',
            name: 'Canon PIXMA',
            isActive: true,
            isOnline: false,
          },
        ],
      });
    });

    // Mock jobs
    await this.page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: [],
        total: 0,
        limit: 50,
        offset: 0,
      });
    });

    // Mock environment report
    await this.page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, {
        pagesPrinted: 0,
        co2Grams: 0,
        treesSaved: 0,
        period: '30d',
      });
    });
  }
}
