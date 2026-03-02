/**
 * WebSocket Mock Utilities
 * Provides mocking utilities for testing WebSocket real-time updates
 */
import { Page, WebSocketRoute } from '@playwright/test';

export interface WebSocketMessage {
  type: string;
  data: unknown;
  timestamp: string;
}

export interface JobUpdateMessage extends WebSocketMessage {
  type: 'job.update';
  data: {
    jobId: string;
    status: string;
    progress?: number;
    error?: string;
  };
}

export interface PrinterStatusMessage extends WebSocketMessage {
  type: 'printer.status';
  data: {
    printerId: string;
    status: 'online' | 'offline' | 'error';
    message?: string;
  };
}

export interface NotificationMessage extends WebSocketMessage {
  type: 'notification';
  data: {
    id: string;
    title: string;
    body: string;
    type: 'info' | 'success' | 'warning' | 'error';
  };
}

export interface AgentHeartbeatMessage extends WebSocketMessage {
  type: 'agent.heartbeat';
  data: {
    agentId: string;
    status: 'online' | 'offline';
    lastSeen: string;
    jobQueueDepth: number;
  };
}

export class WebSocketMock {
  private page: Page;
  private wsUrl: string;
  private server: WebSocketRoute | null = null;
  private connectedClients: Set<WebSocketRoute> = new Set();
  private messageHandlers: Map<string, (data: unknown) => void> = new Map();

  constructor(page: Page, wsUrl: string = 'ws://localhost:8005/ws') {
    this.page = page;
    this.wsUrl = wsUrl;
  }

  /**
   * Setup WebSocket mock
   */
  async setup() {
    // Mock WebSocket connections
    this.server = await this.page.routeWebSocket(this.wsUrl, (ws) => {
      this.connectedClients.add(ws);

      ws.onMessage((message) => {
        this.handleClientMessage(ws, message);
      });

      ws.onClose(() => {
        this.connectedClients.delete(ws);
      });

      // Send welcome message
      this.sendToClient(ws, {
        type: 'connection.established',
        data: { timestamp: new Date().toISOString() },
      });
    });

    // Mock WebSocket client setup in the page
    await this.page.addInitScript((wsUrl) => {
      // Store original WebSocket
      const OriginalWebSocket = window.WebSocket;

      // Create mock WebSocket that connects to our mock server
      class MockWebSocket extends OriginalWebSocket {
        constructor(url: string, protocols?: string | string[]) {
          super(url, protocols);
          console.log('[WS Mock] Connecting to', url);
        }
      }

      window.WebSocket = MockWebSocket as any;
    }, this.wsUrl);
  }

  /**
   * Cleanup WebSocket mock
   */
  async cleanup() {
    if (this.server) {
      await this.server.unroute?.();
      this.server = null;
    }
    this.connectedClients.clear();
    this.messageHandlers.clear();
  }

  /**
   * Handle incoming message from client
   */
  private handleClientMessage(ws: WebSocketRoute, message: string) {
    try {
      const parsed = JSON.parse(message.toString());
      console.log('[WS Mock] Received from client:', parsed);

      // Handle authentication
      if (parsed.type === 'auth') {
        this.sendToClient(ws, {
          type: 'auth.success',
          data: { userId: 'user-1' },
        });
      }
    } catch (error) {
      console.error('[WS Mock] Error parsing message:', error);
    }
  }

  /**
   * Send message to a specific client
   */
  sendToClient(ws: WebSocketRoute, message: Record<string, unknown>) {
    try {
      ws.send(JSON.stringify(message));
    } catch (error) {
      console.error('[WS Mock] Error sending message:', error);
    }
  }

  /**
   * Broadcast message to all connected clients
   */
  broadcast(message: Record<string, unknown>) {
    for (const client of this.connectedClients) {
      this.sendToClient(client, message);
    }
  }

  /**
   * Send job status update
   */
  sendJobUpdate(data: JobUpdateMessage['data']) {
    this.broadcast({
      type: 'job.update',
      data,
      timestamp: new Date().toISOString(),
    } as JobUpdateMessage);
  }

  /**
   * Send printer status update
   */
  sendPrinterStatus(data: PrinterStatusMessage['data']) {
    this.broadcast({
      type: 'printer.status',
      data,
      timestamp: new Date().toISOString(),
    } as PrinterStatusMessage);
  }

  /**
   * Send notification
   */
  sendNotification(data: NotificationMessage['data']) {
    this.broadcast({
      type: 'notification',
      data,
      timestamp: new Date().toISOString(),
    } as NotificationMessage);
  }

  /**
   * Send agent heartbeat
   */
  sendAgentHeartbeat(data: AgentHeartbeatMessage['data']) {
    this.broadcast({
      type: 'agent.heartbeat',
      data,
      timestamp: new Date().toISOString(),
    } as AgentHeartbeatMessage);
  }

  /**
   * Simulate job progress updates
   */
  async simulateJobProgress(jobId: string, duration: number = 5000) {
    const steps = 10;
    const delay = duration / steps;

    for (let i = 0; i <= steps; i++) {
      await new Promise((resolve) => setTimeout(resolve, delay));

      const progress = Math.min((i / steps) * 100, 100);
      const status = progress >= 100 ? 'completed' : 'processing';

      this.sendJobUpdate({
        jobId,
        status,
        progress: Math.round(progress),
      });
    }
  }

  /**
   * Simulate printer going offline
   */
  simulatePrinterOffline(printerId: string, reason: string = 'Connection lost') {
    this.sendPrinterStatus({
      printerId,
      status: 'offline',
      message: reason,
    });
  }

  /**
   * Simulate printer coming online
   */
  simulatePrinterOnline(printerId: string) {
    this.sendPrinterStatus({
      printerId,
      status: 'online',
    });
  }

  /**
   * Get connected client count
   */
  getClientCount(): number {
    return this.connectedClients.size;
  }

  /**
   * Close all client connections
   */
  closeAllConnections() {
    for (const client of this.connectedClients) {
      try {
        client.close();
      } catch (error) {
        // Ignore errors when closing
      }
    }
    this.connectedClients.clear();
  }

  /**
   * Simulate connection error
   */
  simulateConnectionError() {
    for (const client of this.connectedClients) {
      try {
        client.error(new Error('Connection lost'));
      } catch (error) {
        // Ignore errors
      }
    }
  }

  /**
   * Wait for client to send specific message type
   */
  async waitForClientMessage(type: string, timeout: number = 5000): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const timeoutId = setTimeout(() => {
        this.messageHandlers.delete(type);
        reject(new Error(`Timeout waiting for message type: ${type}`));
      }, timeout);

      this.messageHandlers.set(type, (data) => {
        clearTimeout(timeoutId);
        resolve(data);
      });
    });
  }
}

/**
 * Setup WebSocket mock for a test
 */
export async function setupWebSocketMock(page: Page, wsUrl?: string): Promise<WebSocketMock> {
  const mock = new WebSocketMock(page, wsUrl);
  await mock.setup();
  return mock;
}

/**
 * Simulate real-time updates scenario
 */
export async function simulateRealtimeScenario(
  mock: WebSocketMock,
  scenario: 'job_progress' | 'printer_status' | 'agent_heartbeat' | 'notifications'
) {
  switch (scenario) {
    case 'job_progress':
      await mock.simulateJobProgress('job-test-123', 3000);
      break;

    case 'printer_status':
      mock.sendPrinterStatus({ printerId: 'printer-1', status: 'online' });
      await new Promise((resolve) => setTimeout(resolve, 1000));
      mock.sendPrinterStatus({ printerId: 'printer-1', status: 'offline', message: 'Paper jam' });
      await new Promise((resolve) => setTimeout(resolve, 2000));
      mock.sendPrinterStatus({ printerId: 'printer-1', status: 'online' });
      break;

    case 'agent_heartbeat':
      for (let i = 0; i < 5; i++) {
        mock.sendAgentHeartbeat({
          agentId: 'agent-1',
          status: 'online',
          lastSeen: new Date().toISOString(),
          jobQueueDepth: Math.floor(Math.random() * 5),
        });
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }
      break;

    case 'notifications':
      mock.sendNotification({
        id: 'notif-1',
        title: 'Job Completed',
        body: 'Your document "Report.pdf" has been printed successfully.',
        type: 'success',
      });
      await new Promise((resolve) => setTimeout(resolve, 500));
      mock.sendNotification({
        id: 'notif-2',
        title: 'Printer Offline',
        body: 'HP LaserJet Pro has gone offline.',
        type: 'warning',
      });
      break;
  }
}

/**
 * Intercept WebSocket messages from the page
 */
export async function interceptWebSocketMessages(
  page: Page,
  callback: (message: WebSocketMessage) => void
): Promise<void> {
  await page.addInitScript((callbackString) => {
    const OriginalWebSocket = window.WebSocket;
    const callback = new Function('return ' + callbackString)();

    class InterceptedWebSocket extends OriginalWebSocket {
      constructor(url: string, protocols?: string | string[]) {
        super(url, protocols);
      }

      send(data: string | ArrayBuffer | Blob) {
        if (typeof data === 'string') {
          try {
            const parsed = JSON.parse(data);
            callback(parsed);
          } catch (error) {
            // Not JSON, ignore
          }
        }
        return super.send(data);
      }
    }

    window.WebSocket = InterceptedWebSocket as any;
  }, callback.toString());
}

/**
 * Verify WebSocket connection status
 */
export async function verifyWebSocketConnected(page: Page): Promise<boolean> {
  return await page.evaluate(() => {
    // Check if the app has an active WebSocket connection
    const ws = (window as any).activeWebSocket;
    return ws && ws.readyState === WebSocket.OPEN;
  });
}

/**
 * Wait for WebSocket to connect
 */
export async function waitForWebSocketConnection(page: Page, timeout: number = 5000): Promise<void> {
  await page.waitForFunction(
    () => {
      const ws = (window as any).activeWebSocket;
      return ws && ws.readyState === WebSocket.OPEN;
    },
    { timeout }
  );
}

/**
 * Get WebSocket connection info
 */
export async function getWebSocketInfo(page: Page): Promise<{
  connected: boolean;
  url?: string;
  readyState?: number;
}> {
  return await page.evaluate(() => {
    const ws = (window as any).activeWebSocket;
    if (!ws) {
      return { connected: false };
    }
    return {
      connected: ws.readyState === WebSocket.OPEN,
      url: ws.url,
      readyState: ws.readyState,
    };
  });
}

export default WebSocketMock;
