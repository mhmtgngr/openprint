/**
 * NotificationCenter - Bell icon with dropdown showing real-time WebSocket notifications
 */
import { useState, useEffect, useRef, useCallback } from 'react';
import { wsService } from '@/services/websocket';
import type { WebSocketMessage } from '@/types';

interface Notification {
  id: string;
  type: string;
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
}

const formatNotification = (msg: WebSocketMessage): Notification | null => {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const timestamp = new Date();

  switch (msg.type) {
    case 'job.completed': {
      const data = msg.data as { jobId?: string; documentName?: string };
      return {
        id,
        type: 'success',
        title: 'Print Complete',
        message: data.documentName
          ? `"${data.documentName}" finished printing`
          : `Job ${data.jobId?.slice(0, 8) || ''} completed`,
        timestamp,
        read: false,
      };
    }
    case 'job.failed': {
      const data = msg.data as { jobId?: string; message?: string; documentName?: string };
      return {
        id,
        type: 'error',
        title: 'Print Failed',
        message: data.message || data.documentName
          ? `"${data.documentName}" failed to print`
          : `Job ${data.jobId?.slice(0, 8) || ''} failed`,
        timestamp,
        read: false,
      };
    }
    case 'job.created': {
      const data = msg.data as { documentName?: string };
      return {
        id,
        type: 'info',
        title: 'New Print Job',
        message: data.documentName
          ? `"${data.documentName}" queued for printing`
          : 'New print job queued',
        timestamp,
        read: false,
      };
    }
    case 'printer.online': {
      const data = msg.data as { printerId?: string; printerName?: string };
      return {
        id,
        type: 'success',
        title: 'Printer Online',
        message: data.printerName || `Printer ${data.printerId?.slice(0, 8) || ''} is now online`,
        timestamp,
        read: false,
      };
    }
    case 'printer.offline': {
      const data = msg.data as { printerId?: string; printerName?: string };
      return {
        id,
        type: 'warning',
        title: 'Printer Offline',
        message: data.printerName
          ? `${data.printerName} went offline`
          : `Printer ${data.printerId?.slice(0, 8) || ''} went offline`,
        timestamp,
        read: false,
      };
    }
    case 'notification': {
      const data = msg.data as { title?: string; message?: string };
      return {
        id,
        type: 'info',
        title: data.title || 'Notification',
        message: data.message || 'You have a new notification',
        timestamp,
        read: false,
      };
    }
    default:
      return null;
  }
};

const formatTimeAgo = (date: Date): string => {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return 'Just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
};

const typeStyles: Record<string, { dot: string; bg: string }> = {
  success: { dot: 'bg-green-500', bg: 'bg-green-50 dark:bg-green-900/20' },
  error: { dot: 'bg-red-500', bg: 'bg-red-50 dark:bg-red-900/20' },
  warning: { dot: 'bg-yellow-500', bg: 'bg-yellow-50 dark:bg-yellow-900/20' },
  info: { dot: 'bg-blue-500', bg: 'bg-blue-50 dark:bg-blue-900/20' },
};

export const NotificationCenter = () => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const unreadCount = notifications.filter((n) => !n.read).length;

  // Subscribe to WebSocket messages
  useEffect(() => {
    const unsubscribe = wsService.subscribe((msg: WebSocketMessage) => {
      const notification = formatNotification(msg);
      if (notification) {
        setNotifications((prev) => [notification, ...prev].slice(0, 50));
      }
    });
    return unsubscribe;
  }, []);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const markAllRead = useCallback(() => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  }, []);

  const clearAll = useCallback(() => {
    setNotifications([]);
    setIsOpen(false);
  }, []);

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Bell button */}
      <button
        onClick={() => {
          setIsOpen(!isOpen);
          if (!isOpen && unreadCount > 0) {
            markAllRead();
          }
        }}
        className="relative p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        aria-label="Notifications"
      >
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex items-center justify-center w-5 h-5 text-xs font-bold text-white bg-red-500 rounded-full">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-96 bg-white dark:bg-gray-800 rounded-xl shadow-xl border border-gray-200 dark:border-gray-700 z-50 overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">Notifications</h3>
            {notifications.length > 0 && (
              <button
                onClick={clearAll}
                className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200"
              >
                Clear all
              </button>
            )}
          </div>

          {/* Notification list */}
          <div className="max-h-96 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="py-12 text-center">
                <svg className="mx-auto w-10 h-10 text-gray-300 dark:text-gray-600 mb-3" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                </svg>
                <p className="text-sm text-gray-500 dark:text-gray-400">No notifications yet</p>
                <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                  You'll see print job updates and alerts here
                </p>
              </div>
            ) : (
              notifications.map((notification) => {
                const styles = typeStyles[notification.type] || typeStyles.info;
                return (
                  <div
                    key={notification.id}
                    className={`px-4 py-3 border-b border-gray-100 dark:border-gray-700/50 last:border-0 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors ${
                      !notification.read ? styles.bg : ''
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      <div className={`w-2 h-2 rounded-full mt-2 flex-shrink-0 ${styles.dot}`} />
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {notification.title}
                        </p>
                        <p className="text-sm text-gray-600 dark:text-gray-400 truncate">
                          {notification.message}
                        </p>
                        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                          {formatTimeAgo(notification.timestamp)}
                        </p>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
};
