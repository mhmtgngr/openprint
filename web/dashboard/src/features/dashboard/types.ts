/**
 * Dashboard feature types for the OpenPrint Dashboard
 */

import type { JobStatus, PrintJob } from '@/types';

// Dashboard statistics summary
export interface DashboardStats {
  totalJobs: number;
  completedJobs: number;
  queuedJobs: number;
  failedJobs: number;
  activeDevices: number;
  totalDevices: number;
  storageUsed: number;
  storageTotal: number;
  pagesPrintedToday: number;
  pagesPrintedMonth: number;
  period?: {
    start: string;
    end: string;
  };
}

// Recent activity item
export interface Activity {
  id: string;
  type: ActivityType;
  jobId?: string;
  jobName?: string;
  status?: JobStatus;
  userId?: string;
  userName?: string;
  printerId?: string;
  printerName?: string;
  timestamp: string;
  details?: string;
}

export type ActivityType =
  | 'job_created'
  | 'job_completed'
  | 'job_failed'
  | 'job_cancelled'
  | 'printer_online'
  | 'printer_offline'
  | 'agent_connected'
  | 'agent_disconnected'
  | 'user_invited'
  | 'user_joined';

// Chart data point
export interface ChartDataPoint {
  label: string;
  value: number;
  date?: string;
}

// Job statistics for chart
export interface JobStatistics {
  period: 'day' | 'week' | 'month';
  data: ChartDataPoint[];
  total: number;
  completed: number;
  failed: number;
  averagePerDay: number;
}

// Quick action type
export type QuickAction =
  | 'create_job'
  | 'add_printer'
  | 'invite_user'
  | 'view_all_jobs'
  | 'view_all_devices';

// Activity configuration for UI display
export interface ActivityConfig {
  label: string;
  icon: string;
  bgColor: string;
  textColor: string;
  dotColor: string;
}

export const ACTIVITY_CONFIG: Record<ActivityType, ActivityConfig> = {
  job_created: {
    label: 'Job Created',
    icon: 'document-plus',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    textColor: 'text-blue-700 dark:text-blue-300',
    dotColor: 'bg-blue-500',
  },
  job_completed: {
    label: 'Job Completed',
    icon: 'check-circle',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    dotColor: 'bg-green-500',
  },
  job_failed: {
    label: 'Job Failed',
    icon: 'exclamation-circle',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
  },
  job_cancelled: {
    label: 'Job Cancelled',
    icon: 'x-circle',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    textColor: 'text-gray-700 dark:text-gray-300',
    dotColor: 'bg-gray-500',
  },
  printer_online: {
    label: 'Printer Online',
    icon: 'printer',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    dotColor: 'bg-green-500',
  },
  printer_offline: {
    label: 'Printer Offline',
    icon: 'printer',
    bgColor: 'bg-orange-100 dark:bg-orange-900/30',
    textColor: 'text-orange-700 dark:text-orange-300',
    dotColor: 'bg-orange-500',
  },
  agent_connected: {
    label: 'Agent Connected',
    icon: 'server',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    dotColor: 'bg-green-500',
  },
  agent_disconnected: {
    label: 'Agent Disconnected',
    icon: 'server',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
  },
  user_invited: {
    label: 'User Invited',
    icon: 'mail',
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    textColor: 'text-purple-700 dark:text-purple-300',
    dotColor: 'bg-purple-500',
  },
  user_joined: {
    label: 'User Joined',
    icon: 'user-plus',
    bgColor: 'bg-teal-100 dark:bg-teal-900/30',
    textColor: 'text-teal-700 dark:text-teal-300',
    dotColor: 'bg-teal-500',
  },
};

// Convert a PrintJob to an Activity item
export const jobToActivity = (job: PrintJob): Activity => {
  const activityType: ActivityType =
    job.status === 'completed'
      ? 'job_completed'
      : job.status === 'failed'
        ? 'job_failed'
        : job.status === 'cancelled'
          ? 'job_cancelled'
          : 'job_created';

  return {
    id: job.id,
    type: activityType,
    jobId: job.id,
    jobName: job.documentName,
    status: job.status,
    userId: job.userId,
    printerId: job.printerId,
    printerName: job.printer?.name,
    timestamp: job.createdAt,
    details: job.errorMessage || undefined,
  };
};
