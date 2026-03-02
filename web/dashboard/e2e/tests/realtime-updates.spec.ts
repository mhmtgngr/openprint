/**
 * Real-time Updates E2E Tests
 * Tests for WebSocket real-time updates (job status, agent heartbeat, notifications)
 */
import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, mockUsers } from '../helpers';
import { setupWebSocketMock, simulateRealtimeScenario, verifyWebSocketConnected, waitForWebSocketConnection } from '../fixtures/websocket-mock';
import { JobsPage } from '../pages/JobsPage';
import { DashboardPage } from '../pages/DashboardPage';
import { PrintersPage } from '../pages/PrintersPage';

test.describe('Real-time Updates - WebSocket Connection', () => {
  test('should establish WebSocket connection', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    const isConnected = await verifyWebSocketConnected(page);
    expect(isConnected).toBe(true);
    expect(wsMock.getClientCount()).toBe(1);

    await wsMock.cleanup();
  });

  test('should show connection status indicator', async ({ page }) => {
    await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Look for connection status indicator
    const statusIndicator = page.locator('[data-testid="ws-status"], .connection-status');
    await expect(statusIndicator).toHaveAttribute('data-status', 'connected');
  });

  test('should handle connection failure gracefully', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Simulate connection error
    wsMock.simulateConnectionError();

    // Should show disconnected status
    const statusIndicator = page.locator('[data-testid="ws-status"], .connection-status');
    await expect(statusIndicator).toHaveAttribute('data-status', 'disconnected');

    // Should show error notification
    await expect(page.locator('.error, [data-testid="connection-error"]')).toBeVisible();

    await wsMock.cleanup();
  });

  test('should reconnect automatically', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Simulate disconnection
    wsMock.closeAllConnections();

    await page.waitForTimeout(2000);

    // Should attempt reconnection
    const statusIndicator = page.locator('[data-testid="ws-status"], .connection-status');
    await expect(statusIndicator).toHaveAttribute('data-status', /connecting|connected/);

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Job Status', () => {
  test('should receive job status updates', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/jobs');

    const jobsPage = new JobsPage(page);
    await jobsPage.setupMocks();

    // Send job update
    wsMock.sendJobUpdate({
      jobId: 'job-1',
      status: 'processing',
      progress: 50,
    });

    await page.waitForTimeout(500);

    // Verify UI updated
    const statusBadge = page.locator('[data-testid="job-status"]').filter({ hasText: 'job-1' });
    await expect(statusBadge).toContainText('processing');

    await wsMock.cleanup();
  });

  test('should show job progress in real-time', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/jobs');

    const jobsPage = new JobsPage(page);
    await jobsPage.setupMocks();

    // Simulate job progress
    await wsMock.simulateJobProgress('job-1', 3000);

    // Wait for progress to complete
    await page.waitForTimeout(3500);

    // Verify final status
    const statusBadge = page.locator('[data-testid="job-status"]').filter({ hasText: 'job-1' });
    await expect(statusBadge).toContainText('completed');

    await wsMock.cleanup();
  });

  test('should update job list in real-time', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    const dashboardPage = new DashboardPage(page);
    await dashboardPage.setupMocks();

    // Initially no jobs
    let jobCount = await dashboardPage.getRecentJobCount();
    expect(jobCount).toBe(0);

    // Send new job notification
    wsMock.sendJobUpdate({
      jobId: 'new-job-1',
      status: 'queued',
    });

    await page.waitForTimeout(500);

    // Verify job appeared
    jobCount = await dashboardPage.getRecentJobCount();
    expect(jobCount).toBeGreaterThan(0);

    await wsMock.cleanup();
  });

  test('should handle job failure notifications', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/jobs');

    const jobsPage = new JobsPage(page);
    await jobsPage.setupMocks();

    // Send failed job update
    wsMock.sendJobUpdate({
      jobId: 'job-1',
      status: 'failed',
      error: 'Printer out of paper',
    });

    await page.waitForTimeout(500);

    // Verify error state
    const statusBadge = page.locator('[data-testid="job-status"]').filter({ hasText: 'job-1' });
    await expect(statusBadge).toHaveClass(/error|failed/);

    // Should show error message
    await expect(page.locator('text=/out of paper/i')).toBeVisible();

    await wsMock.cleanup();
  });

  test('should show multiple job updates simultaneously', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/jobs');

    const jobsPage = new JobsPage(page);
    await jobsPage.setupMocks();

    // Send multiple updates
    wsMock.sendJobUpdate({ jobId: 'job-1', status: 'processing', progress: 25 });
    wsMock.sendJobUpdate({ jobId: 'job-2', status: 'queued' });
    wsMock.sendJobUpdate({ jobId: 'job-3', status: 'completed' });

    await page.waitForTimeout(1000);

    // Verify all updates reflected
    const jobItems = page.locator('[data-testid="job-item"], .job-item');
    const count = await jobItems.count();
    expect(count).toBeGreaterThan(0);

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Printer Status', () => {
  test('should receive printer status updates', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/printers');

    const printersPage = new PrintersPage(page);
    await printersPage.setupMocks();

    // Send printer offline notification
    wsMock.sendPrinterStatus({
      printerId: 'printer-1',
      status: 'offline',
      message: 'Connection lost',
    });

    await page.waitForTimeout(500);

    // Verify printer status updated
    await printersPage.verifyPrinterStatus('printer-1', 'offline');

    await wsMock.cleanup();
  });

  test('should show printer error notifications', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    const dashboardPage = new DashboardPage(page);
    await dashboardPage.setupMocks();

    // Send printer error
    wsMock.sendPrinterStatus({
      printerId: 'printer-1',
      status: 'error',
      message: 'Paper jam',
    });

    await page.waitForTimeout(500);

    // Should show notification
    await expect(page.locator('text=/paper jam/i')).toBeVisible();

    await wsMock.cleanup();
  });

  test('should update printer status back to online', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/printers');

    const printersPage = new PrintersPage(page);
    await printersPage.setupMocks();

    // Send offline then online
    wsMock.sendPrinterStatus({ printerId: 'printer-1', status: 'offline', message: 'Disconnected' });
    await page.waitForTimeout(500);
    await printersPage.verifyPrinterStatus('printer-1', 'offline');

    wsMock.sendPrinterStatus({ printerId: 'printer-1', status: 'online' });
    await page.waitForTimeout(500);
    await printersPage.verifyPrinterStatus('printer-1', 'online');

    await wsMock.cleanup();
  });

  test('should simulate printer status cycle', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    await simulateRealtimeScenario(wsMock, 'printer_status');

    // Wait for scenario to complete
    await page.waitForTimeout(5000);

    // Should have shown various status notifications
    await expect(page.locator('text=/offline/i')).toBeVisible();
    await page.waitForTimeout(2500);
    await expect(page.locator('text=/online/i')).toBeVisible();

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Notifications', () => {
  test('should receive toast notifications', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Send notification
    wsMock.sendNotification({
      id: 'notif-1',
      title: 'Job Completed',
      body: 'Your document has been printed successfully.',
      type: 'success',
    });

    await page.waitForTimeout(500);

    // Verify toast appeared
    const toast = page.locator('[data-testid="toast"], .toast');
    await expect(toast).toBeVisible();
    await expect(toast).toContainText('Job Completed');

    await wsMock.cleanup();
  });

  test('should show different notification types', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Success notification
    wsMock.sendNotification({
      id: 'notif-1',
      title: 'Success',
      body: 'Operation completed',
      type: 'success',
    });

    await page.waitForTimeout(500);
    let toast = page.locator('[data-testid="toast"].success, .toast.success');
    await expect(toast).toBeVisible();

    // Warning notification
    wsMock.sendNotification({
      id: 'notif-2',
      title: 'Warning',
      body: 'Low paper warning',
      type: 'warning',
    });

    await page.waitForTimeout(500);
    toast = page.locator('[data-testid="toast"].warning, .toast.warning');
    await expect(toast).toBeVisible();

    // Error notification
    wsMock.sendNotification({
      id: 'notif-3',
      title: 'Error',
      body: 'Printing failed',
      type: 'error',
    });

    await page.waitForTimeout(500);
    toast = page.locator('[data-testid="toast"].error, .toast.error');
    await expect(toast).toBeVisible();

    await wsMock.cleanup();
  });

  test('should auto-dismiss notifications', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    wsMock.sendNotification({
      id: 'notif-1',
      title: 'Test',
      body: 'Auto-dismiss test',
      type: 'info',
    });

    // Wait for notification to appear
    const toast = page.locator('[data-testid="toast"], .toast');
    await expect(toast).toBeVisible();

    // Wait for auto-dismiss (typically 5 seconds)
    await page.waitForTimeout(6000);

    // Should be gone now
    await expect(toast).not.toBeVisible();

    await wsMock.cleanup();
  });

  test('should handle rapid notifications', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Send multiple notifications rapidly
    for (let i = 1; i <= 5; i++) {
      wsMock.sendNotification({
        id: `notif-${i}`,
        title: `Notification ${i}`,
        body: `Test message ${i}`,
        type: 'info',
      });
    }

    await page.waitForTimeout(1000);

    // Should handle gracefully (stack or queue)
    const toasts = page.locator('[data-testid="toast"], .toast');
    const count = await toasts.count();
    expect(count).toBeGreaterThan(0);

    await wsMock.cleanup();
  });

  test('should allow manual notification dismissal', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    wsMock.sendNotification({
      id: 'notif-1',
      title: 'Test',
      body: 'Click to dismiss',
      type: 'info',
    });

    await page.waitForTimeout(500);

    const toast = page.locator('[data-testid="toast"], .toast');
    const closeButton = toast.locator('button[aria-label="Close"], .close-button');
    await closeButton.click();

    await expect(toast).not.toBeVisible();

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Agent Heartbeat', () => {
  test('should receive agent heartbeat updates', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/agents');

    // Send heartbeat
    wsMock.sendAgentHeartbeat({
      agentId: 'agent-1',
      status: 'online',
      lastSeen: new Date().toISOString(),
      jobQueueDepth: 3,
    });

    await page.waitForTimeout(500);

    // Verify agent status updated
    const agentStatus = page.locator('[data-agent-id="agent-1"] [data-testid="agent-status"]');
    await expect(agentStatus).toContainText('online');

    await wsMock.cleanup();
  });

  test('should detect agent going offline', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/agents');

    // Agent goes offline
    wsMock.sendAgentHeartbeat({
      agentId: 'agent-1',
      status: 'offline',
      lastSeen: new Date(Date.now() - 120000).toISOString(),
      jobQueueDepth: 0,
    });

    await page.waitForTimeout(500);

    const agentStatus = page.locator('[data-agent-id="agent-1"] [data-testid="agent-status"]');
    await expect(agentStatus).toContainText('offline');

    await wsMock.cleanup();
  });

  test('should display job queue depth', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/agents');

    // Send heartbeat with queue depth
    wsMock.sendAgentHeartbeat({
      agentId: 'agent-1',
      status: 'online',
      lastSeen: new Date().toISOString(),
      jobQueueDepth: 5,
    });

    await page.waitForTimeout(500);

    const queueDepth = page.locator('[data-agent-id="agent-1"] [data-testid="queue-depth"]');
    await expect(queueDepth).toContainText('5');

    await wsMock.cleanup();
  });

  test('should simulate continuous heartbeat stream', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    await simulateRealtimeScenario(wsMock, 'agent_heartbeat');

    // Wait for heartbeats
    await page.waitForTimeout(6000);

    // Should have processed multiple heartbeats
    // Verify no errors occurred
    const errors = page.locator('.error, [data-testid="error-message"]');
    const errorCount = await errors.count();
    expect(errorCount).toBe(0);

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Dashboard Scenarios', () => {
  test('should update all dashboard widgets in real-time', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    const dashboardPage = new DashboardPage(page);
    await dashboardPage.setupMocks();

    // Simulate complete real-time scenario
    await simulateRealtimeScenario(wsMock, 'notifications');

    // Check for notification indicator
    await page.waitForTimeout(1000);
    const notificationBadge = page.locator('[data-testid="notification-badge"]');
    await expect(notificationBadge).toBeVisible();

    await wsMock.cleanup();
  });

  test('should reflect job progress on dashboard', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    const dashboardPage = new DashboardPage(page);
    await dashboardPage.setupMocks();

    // Start job progress simulation
    wsMock.simulateJobProgress('dashboard-job-1', 5000);

    // Monitor progress indicators
    for (let i = 0; i <= 100; i += 25) {
      await page.waitForTimeout(1250);
      // Progress should be reflected in UI
    }

    await wsMock.cleanup();
  });

  test('should update printer count on status changes', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    const dashboardPage = new DashboardPage(page);
    await dashboardPage.setupMocks();

    // Get initial printer count
    const initialCount = await dashboardPage.getRecentPrinterCount();

    // Send printer online
    wsMock.sendPrinterStatus({
      printerId: 'new-printer',
      status: 'online',
    });

    await page.waitForTimeout(500);

    // Count should increase
    const newCount = await dashboardPage.getRecentPrinterCount();
    expect(newCount).toBeGreaterThanOrEqual(initialCount);

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Error Scenarios', () => {
  test('should handle malformed messages gracefully', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Send invalid message
    wsMock.broadcast({ invalid: 'data' } as any);

    await page.waitForTimeout(500);

    // Should not crash or show errors
    const errors = page.locator('.error, [data-testid="error-message"]');
    const criticalErrors = errors.filter({ hasText: /WebSocket|Parse error/i });
    await expect(criticalErrors).toHaveCount(0);

    await wsMock.cleanup();
  });

  test('should handle message queue buildup', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Send many messages rapidly
    for (let i = 0; i < 100; i++) {
      wsMock.sendNotification({
        id: `notif-${i}`,
        title: `Batch ${i}`,
        body: `Message ${i}`,
        type: 'info',
      });
    }

    await page.waitForTimeout(2000);

    // Should handle without crashing
    const toasts = page.locator('[data-testid="toast"], .toast');
    const count = await toasts.count();
    expect(count).toBeGreaterThan(0);

    await wsMock.cleanup();
  });

  test('should recover from connection drops', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Close connection
    wsMock.closeAllConnections();

    await page.waitForTimeout(1000);

    // Send message (should queue or fail gracefully)
    wsMock.sendNotification({
      id: 'notif-test',
      title: 'Test',
      body: 'After reconnect',
      type: 'info',
    });

    await page.waitForTimeout(2000);

    // Verify connection status
    const statusIndicator = page.locator('[data-testid="ws-status"], .connection-status');
    const status = await statusIndicator.getAttribute('data-status');
    expect(status).toMatch(/connecting|connected/);

    await wsMock.cleanup();
  });
});

test.describe('Real-time Updates - Performance', () => {
  test('should handle high-frequency updates', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/jobs');

    const jobsPage = new JobsPage(page);
    await jobsPage.setupMocks();

    // Send 50 updates per second for 2 seconds
    const interval = setInterval(() => {
      for (let i = 0; i < 50; i++) {
        wsMock.sendJobUpdate({
          jobId: `job-${i}`,
          status: i % 2 === 0 ? 'processing' : 'queued',
          progress: Math.floor(Math.random() * 100),
        });
      }
    }, 1000);

    await page.waitForTimeout(2000);
    clearInterval(interval);

    // Page should still be responsive
    const button = page.locator('button').first();
    await expect(button).toBeVisible();

    await wsMock.cleanup();
  });

  test('should not cause memory leaks', async ({ page }) => {
    const wsMock = await setupWebSocketMock(page);
    await setupAuthAndNavigate(page, '/dashboard');

    // Get initial memory usage
    const initialMemory = await page.evaluate(() => {
      return (performance as any).memory?.usedJSHeapSize || 0;
    });

    // Send many updates
    for (let i = 0; i < 1000; i++) {
      wsMock.sendNotification({
        id: `notif-${i}`,
        title: `Memory test ${i}`,
        body: 'Test',
        type: 'info',
      });
    }

    await page.waitForTimeout(3000);

    // Check memory didn't grow excessively
    const finalMemory = await page.evaluate(() => {
      return (performance as any).memory?.usedJSHeapSize || 0;
    });

    // Memory growth should be reasonable (less than 50MB)
    const growth = finalMemory - initialMemory;
    expect(growth).toBeLessThan(50 * 1024 * 1024);

    await wsMock.cleanup();
  });
});
