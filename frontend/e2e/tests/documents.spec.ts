import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Documents Management', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock documents API
    await page.route('**/api/v1/documents*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'doc-1',
              name: 'Quarterly Report.pdf',
              contentType: 'application/pdf',
              size: 1024000,
              pageCount: 15,
              createdAt: '2024-02-27T10:00:00Z',
              isEncrypted: true,
              checksum: 'abc123def456',
            },
            {
              id: 'doc-2',
              name: 'Sales Presentation.pptx',
              contentType: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
              size: 5120000,
              pageCount: 24,
              createdAt: '2024-02-26T14:30:00Z',
              isEncrypted: false,
            },
            {
              id: 'doc-3',
              name: 'Invoice Template.docx',
              contentType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
              size: 256000,
              pageCount: 2,
              createdAt: '2024-02-25T09:15:00Z',
              isEncrypted: true,
            },
            {
              id: 'doc-4',
              name: 'Meeting Notes.txt',
              contentType: 'text/plain',
              size: 4096,
              pageCount: 1,
              createdAt: '2024-02-24T16:00:00Z',
              isEncrypted: false,
            },
          ],
          total: 4,
          limit: 50,
          offset: 0,
        }),
      });
    });

    // Mock document stats API
    await page.route('**/api/v1/documents/stats*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          totalDocuments: 4,
          totalSize: 6403096,
          encryptedCount: 2,
          avgSize: 1600774,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display documents page', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByRole('heading', { name: /documents/i })).toBeVisible();
    await expect(page.getByText('Manage and upload print documents')).toBeVisible();
  });

  test('should display all documents', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByText('Quarterly Report.pdf')).toBeVisible();
    await expect(page.getByText('Sales Presentation.pptx')).toBeVisible();
    await expect(page.getByText('Invoice Template.docx')).toBeVisible();
    await expect(page.getByText('Meeting Notes.txt')).toBeVisible();
  });

  test('should display document cards with metadata', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByText('1 MB')).toBeVisible();
    await expect(page.getByText('15 pages')).toBeVisible();
    await expect(page.getByText('Encrypted')).toBeVisible();
  });

  test('should display upload button', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByRole('button', { name: /upload/i })).toBeVisible();
  });

  test('should open upload modal', async ({ page }) => {
    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    await expect(page.getByText(/upload documents/i)).toBeVisible();
  });

  test('should display upload zone with drag-drop', async ({ page }) => {
    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    await expect(page.getByText(/drag and drop/i)).toBeVisible();
    await expect(page.getByText(/browse files/i)).toBeVisible();
  });

  test('should show supported file types', async ({ page }) => {
    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    await expect(page.getByText(/supported:.*pdf/i)).toBeVisible();
  });

  test('should display document statistics cards', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByText(/total documents/i)).toBeVisible();
    await expect(page.getByText('4')).toBeVisible();
    await expect(page.getByText(/total size/i)).toBeVisible();
  });

  test('should search documents', async ({ page }) => {
    await page.goto('/documents');

    const searchInput = page.getByPlaceholder(/search/i);
    await searchInput.fill('report');

    await expect(page.getByText('Quarterly Report.pdf')).toBeVisible();
  });

  test('should filter documents by type', async ({ page }) => {
    await page.goto('/documents');

    const pdfFilter = page.getByRole('button', { name: /pdf/i });
    await pdfFilter.click();

    await expect(pdfFilter).toHaveClass(/active/);
  });

  test('should display document encryption status', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByText('Encrypted')).toBeVisible();
  });

  test('should show download button on document cards', async ({ page }) => {
    await page.goto('/documents');

    const downloadButton = page.locator('button[title*="Download"], button:has(svg:has-text("Download"))').first();
    await expect(downloadButton).toBeVisible();
  });

  test('should show delete button on document cards', async ({ page }) => {
    await page.goto('/documents');

    const deleteButton = page.locator('button[title*="Delete"], button:has(svg:has-text("Trash"))').first();
    await expect(deleteButton).toBeVisible();
  });

  test('should open document preview on click', async ({ page }) => {
    // Mock document preview API
    await page.route('**/api/v1/documents/*/preview*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: 'data:application/pdf;base64,JVBERi0xL...',
        }),
      });
    });

    await page.goto('/documents');

    await page.getByText('Quarterly Report.pdf').click();

    await expect(page.getByText(/document preview/i)).toBeVisible();
  });

  test('should show empty state when no documents', async ({ page }) => {
    await page.route('**/api/v1/documents*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          total: 0,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await page.goto('/documents');

    await expect(page.getByText(/no documents/i)).toBeVisible();
    await expect(page.getByText(/upload your first document/i)).toBeVisible();
  });

  test('should paginate documents', async ({ page }) => {
    // Mock paginated response
    await page.route('**/api/v1/documents*', (route) => {
      const url = new URL(route.request().url());
      const offset = parseInt(url.searchParams.get('offset') || '0');

      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: `doc-${offset + 1}`,
              name: `Document ${offset + 1}.pdf`,
              contentType: 'application/pdf',
              size: 1024000,
              pageCount: 5,
              createdAt: '2024-02-27T10:00:00Z',
            },
          ],
          total: 25,
          limit: 1,
          offset,
        }),
      });
    });

    await page.goto('/documents');

    await expect(page.getByRole('button', { name: /next/i })).toBeVisible();
  });
});

test.describe('Document Upload', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/documents*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], total: 0, limit: 50, offset: 0 }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show file input for upload', async ({ page }) => {
    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    const fileInput = page.locator('input[type="file"]');
    await expect(fileInput).toBeVisible();
  });

  test('should accept multiple files', async ({ page }) => {
    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    const fileInput = page.locator('input[type="file"]');
    await expect(fileInput).toHaveAttribute('multiple', '');
  });

  test('should validate file size limit', async ({ page }) => {
    await page.route('**/api/v1/documents/upload*', (route) => {
      route.fulfill({
        status: 413,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'File size exceeds limit' }),
      });
    });

    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    // Try to upload a large file (mock)
    const fileInput = page.locator('input[type="file"]');

    // For large file test, mock the validation
    await page.route('**/api/v1/documents/upload*', (route) => {
      route.fulfill({
        status: 413,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'File size exceeds limit' }),
      });
    });

    // Create a mock file - use a buffer for Playwright
    const largeBuffer = Buffer.alloc(50 * 1024 * 1024); // 50MB
    await fileInput.setInputFiles({
      name: 'large.pdf',
      mimeType: 'application/pdf',
      buffer: largeBuffer,
    });

    await expect(page.getByText(/file size/i)).toBeVisible();
  });

  test('should show upload progress', async ({ page }) => {
    await page.route('**/api/v1/documents/upload*', (route) => {
      // Delay response to show progress
      setTimeout(() => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'doc-new',
            name: 'test.pdf',
            size: 1024000,
            status: 'success',
          }),
        });
      }, 1000);
    });

    await page.goto('/documents');

    await page.getByRole('button', { name: /upload/i }).click();

    const fileInput = page.locator('input[type="file"]');
    // Use buffer for Playwright setInputFiles
    await fileInput.setInputFiles({
      name: 'test.pdf',
      mimeType: 'application/pdf',
      buffer: Buffer.from('test content'),
    });

    await expect(page.locator('.progress-bar, .animate-spin')).toBeVisible();
  });
});

test.describe('Document Deletion', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/documents*', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 204,
          contentType: 'application/json',
          body: '',
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'doc-1',
                name: 'To Delete.pdf',
                contentType: 'application/pdf',
                size: 1024000,
                pageCount: 5,
                createdAt: '2024-02-27T10:00:00Z',
              },
            ],
            total: 1,
            limit: 50,
            offset: 0,
          }),
        });
      }
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show delete confirmation', async ({ page }) => {
    await page.goto('/documents');

    const deleteButton = page.locator('button[title*="Delete"], button:has(svg:has-text("Trash"))').first();
    await deleteButton.click();

    await expect(page.getByText(/are you sure/i)).toBeVisible();
    await expect(page.getByText(/delete document/i)).toBeVisible();
  });

  test('should cancel deletion', async ({ page }) => {
    await page.goto('/documents');

    const deleteButton = page.locator('button[title*="Delete"], button:has(svg:has-text("Trash"))').first();
    await deleteButton.click();

    await page.getByRole('button', { name: /cancel/i }).click();

    await expect(page.getByText(/are you sure/i)).not.toBeVisible();
  });

  test('should confirm deletion and remove document', async ({ page }) => {
    await page.goto('/documents');

    const deleteButton = page.locator('button[title*="Delete"], button:has(svg:has-text("Trash"))').first();
    await deleteButton.click();

    await page.getByRole('button', { name: /delete/i }).click();

    // Document should be removed
    await expect(page.getByText('To Delete.pdf')).not.toBeVisible();
  });
});
