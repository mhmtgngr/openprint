import type { WebSocketMessage, JobStatusUpdateMessage } from '@/types';

type MessageHandler = (message: WebSocketMessage) => void;
type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectTimeout: number | null = null;
  private handlers: Set<MessageHandler> = new Set();
  private statusHandlers: Set<(status: ConnectionStatus) => void> = new Set();
  private connectionStatus: ConnectionStatus = 'disconnected';
  private url: string;

  constructor() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsPath = import.meta.env.VITE_WS_URL || '/ws';

    // Handle relative paths by constructing absolute URL
    if (wsPath.startsWith('/')) {
      this.url = `${wsProtocol}//${window.location.host}${wsPath}`;
    } else {
      this.url = wsPath;
    }
  }

  connect(token: string): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.updateStatus('connecting');

    try {
      const wsUrl = new URL(this.url);
      wsUrl.searchParams.set('token', token);

      this.ws = new WebSocket(wsUrl.toString());

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.updateStatus('connected');
      };

      this.ws.onclose = (event) => {
        this.updateStatus('disconnected');
        if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect(token);
        }
      };

      this.ws.onerror = () => {
        this.updateStatus('error');
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WebSocketMessage;
          this.notifyHandlers(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
    } catch (error) {
      console.error('WebSocket connection error:', error);
      this.updateStatus('error');
      this.scheduleReconnect(token);
    }
  }

  disconnect(): void {
    if (this.reconnectTimeout) {
      window.clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.reconnectAttempts = 0;
    this.updateStatus('disconnected');
  }

  send(message: WebSocketMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  subscribe(handler: MessageHandler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  onStatusChange(handler: (status: ConnectionStatus) => void): () => void {
    this.statusHandlers.add(handler);
    handler(this.connectionStatus);
    return () => this.statusHandlers.delete(handler);
  }

  private notifyHandlers(message: WebSocketMessage): void {
    this.handlers.forEach((handler) => {
      try {
        handler(message);
      } catch (error) {
        console.error('Error in WebSocket message handler:', error);
      }
    });
  }

  private updateStatus(status: ConnectionStatus): void {
    this.connectionStatus = status;
    this.statusHandlers.forEach((handler) => {
      try {
        handler(status);
      } catch (error) {
        console.error('Error in WebSocket status handler:', error);
      }
    });
  }

  private scheduleReconnect(token: string): void {
    if (this.reconnectTimeout) {
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectTimeout = null;
      this.connect(token);
    }, delay);
  }

  getConnectionStatus(): ConnectionStatus {
    return this.connectionStatus;
  }
}

export const wsService = new WebSocketService();

// Convenience hook for job status updates
export const subscribeToJobUpdates = (
  jobId: string,
  onUpdate: (status: JobStatusUpdateMessage) => void
): (() => void) => {
  const handler = (message: WebSocketMessage) => {
    if (message.type === 'job.status_update') {
      const data = message.data as JobStatusUpdateMessage;
      if (data.jobId === jobId) {
        onUpdate(data);
      }
    }
  };

  return wsService.subscribe(handler);
};

// Subscribe to all job updates
export const subscribeToAllJobs = (
  onUpdate: (update: { jobId: string; status: string; message?: string }) => void
): (() => void) => {
  const handler = (message: WebSocketMessage) => {
    if (
      message.type === 'job.status_update' ||
      message.type === 'job.created' ||
      message.type === 'job.completed' ||
      message.type === 'job.failed'
    ) {
      const data = message.data as JobStatusUpdateMessage;
      onUpdate(data);
    }
  };

  return wsService.subscribe(handler);
};

// Subscribe to printer status updates
export const subscribeToPrinterUpdates = (
  onUpdate: (printerId: string, online: boolean) => void
): (() => void) => {
  const handler = (message: WebSocketMessage) => {
    if (message.type === 'printer.online' || message.type === 'printer.offline') {
      const data = message.data as { printerId: string };
      onUpdate(data.printerId, message.type === 'printer.online');
    }
  };

  return wsService.subscribe(handler);
};
