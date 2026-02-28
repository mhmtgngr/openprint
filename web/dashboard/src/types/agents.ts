/**
 * Agent-related types for the OpenPrint Dashboard
 */

import type { PrintJob } from './index';

export type AgentStatus = 'online' | 'offline' | 'error';

export interface Agent {
  id: string;
  name: string;
  userId?: string; // Associated user for user-specific agents
  orgId: string;
  status: AgentStatus;
  platform: string;
  platformVersion?: string;
  agentVersion?: string;
  ipAddress?: string;
  lastHeartbeat?: string;
  capabilities: AgentCapabilities;
  sessionState?: AgentSessionState;
  printerCount?: number;
  jobQueueDepth?: number;
  createdAt: string;
  associatedUser?: {
    id: string;
    name: string;
    email: string;
  };
}

export interface AgentCapabilities {
  supportedFormats: string[];
  maxJobSize: number;
  supportsColor: boolean;
  supportsDuplex: boolean;
  supportsLargeFormat?: boolean;
}

export type AgentSessionState = 'active' | 'idle' | 'disconnected' | 'error';

export interface DiscoveredPrinter {
  id: string;
  agentId: string;
  name: string;
  driver: string;
  port: string;
  type: 'local' | 'network' | 'shared';
  capabilities: DiscoveredPrinterCapabilities;
  status: 'available' | 'offline' | 'error';
  lastSeen: string;
  discoveredAt: string;
  isDefault?: boolean;
}

export interface DiscoveredPrinterCapabilities {
  supportsColor: boolean;
  supportsDuplex: boolean;
  supportedPaperSizes: string[];
  resolution: string;
  maxSheetCount?: number;
  supportedFormats?: string[];
}

export interface AgentDetail extends Agent {
  printers: DiscoveredPrinter[];
  jobHistory: AgentJobHistoryEntry[];
  healthMetrics: AgentHealthMetrics;
  associatedUser?: {
    id: string;
    name: string;
    email: string;
  };
}

export interface AgentJobHistoryEntry {
  id: string;
  jobId: string;
  documentName: string;
  printerName: string;
  status: PrintJob['status'];
  pages: number;
  timestamp: string;
  errorMessage?: string;
}

export interface AgentHealthMetrics {
  uptime: number; // seconds
  totalJobsProcessed: number;
  successfulJobs: number;
  failedJobs: number;
  averageResponseTime: number; // milliseconds
  lastJobTime?: string;
  successRate: number; // percentage
  weeklyJobCounts: WeeklyJobCount[];
}

export interface WeeklyJobCount {
  date: string;
  count: number;
  success: number;
  failed: number;
}

export interface AgentListParams {
  status?: AgentStatus | 'all';
  userId?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export interface JobAssignmentRequest {
  jobId: string;
  agentId?: string;
  userId?: string;
  printerId?: string;
  priority?: number;
}

export interface JobAssignment {
  id: string;
  jobId: string;
  agentId?: string;
  userId?: string;
  printerId?: string;
  status: 'pending' | 'assigned' | 'in_progress' | 'completed' | 'failed';
  priority: number;
  assignedAt?: string;
  completedAt?: string;
  errorMessage?: string;
  job?: PrintJob;
  agent?: Agent;
  user?: {
    id: string;
    name: string;
    email: string;
  };
}

// Agent status configuration for UI display
export interface AgentStatusConfig {
  label: string;
  bgColor: string;
  textColor: string;
  dotColor: string;
  icon: string;
}

export const AGENT_STATUS_CONFIG: Record<AgentStatus, AgentStatusConfig> = {
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
    icon: 'circle-slash',
  },
  error: {
    label: 'Error',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
    icon: 'alert-circle',
  },
};

export interface AgentFormData {
  name: string;
  userId?: string;
}

export interface AgentFormErrors {
  name?: string;
  userId?: string;
}
