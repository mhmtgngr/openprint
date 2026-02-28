/**
 * Device and Printer management types for the OpenPrint Dashboard
 */

import type { Printer, Agent, AgentStatus, PrinterType } from '@/types';

// Re-export types from the main types file that are needed here
export type { Printer, Agent, AgentStatus, PrinterType };

// Combined device type representing either an agent or printer
export interface Device {
  id: string;
  type: 'agent' | 'printer';
  name: string;
  status: DeviceStatus;
  lastSeen?: string;
  createdAt: string;
}

export type DeviceStatus = 'online' | 'offline' | 'error';

// Extended Agent type for UI purposes
export interface DeviceAgent extends Agent {
  printerCount?: number;
  uptime?: string; // Formatted uptime string
}

// Extended Printer type for UI purposes
export interface DevicePrinter extends Printer {
  agentName?: string;
  agentStatus?: AgentStatus;
  queueLength?: number;
  uptime?: string; // Formatted uptime string
}

// Device list filtering and sorting
export interface DeviceListParams {
  status?: DeviceStatus | 'all';
  type?: 'agent' | 'printer' | 'all';
  search?: string;
  limit?: number;
  offset?: number;
}

// Device registration form data
export interface RegisterPrinterFormData {
  name: string;
  type: PrinterType;
  agentId: string;
  driver?: string;
  port?: string;
  capabilities: {
    supportsColor: boolean;
    supportsDuplex: boolean;
    supportedPaperSizes: string[];
    resolution: string;
    maxSheetCount?: number;
  };
}

export interface RegisterAgentFormData {
  name: string;
  platform: string;
  ipAddress?: string;
}

// Device stats
export interface DeviceStats {
  totalAgents: number;
  onlineAgents: number;
  totalPrinters: number;
  onlinePrinters: number;
  offlinePrinters: number;
}

// Device action types
export type DeviceAction = 'enable' | 'disable' | 'restart' | 'delete' | 'refresh';

// Device status configuration for UI display
export interface DeviceStatusConfig {
  label: string;
  bgColor: string;
  textColor: string;
  dotColor: string;
  icon?: string;
}

export const DEVICE_STATUS_CONFIG: Record<DeviceStatus, DeviceStatusConfig> = {
  online: {
    label: 'Online',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    dotColor: 'bg-green-500',
    icon: 'check-circle',
  },
  offline: {
    label: 'Offline',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    textColor: 'text-gray-700 dark:text-gray-300',
    dotColor: 'bg-gray-400',
    icon: 'cloud-off',
  },
  error: {
    label: 'Error',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
    icon: 'exclamation-circle',
  },
};

// Heartbeat interval configuration
export const HEARTBEAT_INTERVALS = {
  ONLINE_THRESHOLD: 60, // seconds - considered offline if no heartbeat within this time
  WARNING_THRESHOLD: 30, // seconds - show warning if heartbeat older than this
};

// Form validation errors
export interface RegisterPrinterFormErrors {
  name?: string;
  type?: string;
  agentId?: string;
  capabilities?: string;
}

export interface RegisterAgentFormErrors {
  name?: string;
  platform?: string;
}
