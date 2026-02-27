import { useEffect, useState } from 'react';
import { wsService } from '@/services/websocket';
import type { WebSocketMessage, JobStatus } from '@/types';

type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export const useWebSocket = () => {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [messages, setMessages] = useState<WebSocketMessage[]>([]);

  useEffect(() => {
    const unsubscribe = wsService.onStatusChange((newStatus) => {
      setStatus(newStatus);
    });

    const unsubscribeMessages = wsService.subscribe((message) => {
      setMessages((prev) => [...prev, message]);
    });

    return () => {
      unsubscribe();
      unsubscribeMessages();
    };
  }, []);

  const sendMessage = (message: WebSocketMessage) => {
    wsService.send(message);
  };

  return {
    status,
    messages,
    sendMessage,
    isConnected: status === 'connected',
  };
};

export const useJobUpdates = (jobId?: string) => {
  const [status, setStatus] = useState<JobStatus | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    if (!jobId) return;

    const unsubscribe = wsService.subscribe((wsMessage) => {
      if (wsMessage.type === 'job.status_update') {
        const data = wsMessage.data as { jobId: string; status: JobStatus; message?: string };
        if (data.jobId === jobId) {
          setStatus(data.status);
          if (data.message) setMessage(data.message);
        }
      }
    });

    return unsubscribe;
  }, [jobId]);

  return { status, message };
};

export const usePrinterUpdates = () => {
  const [printerStatus, setPrinterStatus] = useState<Record<string, boolean>>({});

  useEffect(() => {
    const unsubscribe = wsService.subscribe((wsMessage) => {
      if (wsMessage.type === 'printer.online' || wsMessage.type === 'printer.offline') {
        const data = wsMessage.data as { printerId: string };
        setPrinterStatus((prev) => ({
          ...prev,
          [data.printerId]: wsMessage.type === 'printer.online',
        }));
      }
    });

    return unsubscribe;
  }, []);

  return {
    printerStatus,
    isOnline: (printerId: string) => printerStatus[printerId] ?? false,
  };
};
