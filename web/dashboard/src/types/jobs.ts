/**
 * Job-related types for the OpenPrint Dashboard
 */

import type { Printer, JobSettings } from './index';

export type JobStatus = 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';

export interface PrintJob {
  id: string;
  userId: string;
  printerId?: string;
  orgId: string;
  status: JobStatus;
  documentName: string;
  documentType?: string;
  pageCount: number;
  colorPages?: number;
  fileSize: number;
  fileHash?: string;
  storageKey?: string;
  settings: JobSettings;
  errorMessage?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  autoDeleteAt?: string;
  printer?: Printer;
  progress?: number; // 0-100, only present for processing jobs
}

export interface CreateJobRequest {
  printerId: string;
  documentName: string;
  fileData: string; // base64 encoded file
  fileName: string;
  fileSize: number;
  settings?: Partial<JobSettings>;
}

export interface CreateJobFormData {
  printerId: string;
  documentName: string;
  file?: File;
  color?: boolean;
  duplex?: boolean;
  paperSize?: string;
  copies?: number;
  quality?: string;
  orientation?: 'portrait' | 'landscape';
}

export interface JobHistoryEntry {
  id: string;
  jobId: string;
  status: JobStatus;
  message?: string;
  metadata: Record<string, unknown>;
  timestamp: string;
}

export interface JobListParams {
  status?: JobStatus | 'all';
  limit?: number;
  offset?: number;
  userId?: string;
  printerId?: string;
}

export interface JobFormData {
  printerId: string;
  documentName: string;
  file?: File;
  settings: Partial<JobSettings>;
}

export interface JobFormErrors {
  printerId?: string;
  documentName?: string;
  file?: string;
  settings?: string;
}

export interface FileUploadProgress {
  loaded: number;
  total: number;
  progress: number; // 0-100
}

export interface JobAction {
  type: 'cancel' | 'retry' | 'delete' | 'download';
  jobId: string;
}

// Job status configuration for UI display
export interface JobStatusConfig {
  label: string;
  bgColor: string;
  textColor: string;
  dotColor: string;
  icon?: string;
}

export const JOB_STATUS_CONFIG: Record<JobStatus, JobStatusConfig> = {
  queued: {
    label: 'Queued',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    textColor: 'text-gray-700 dark:text-gray-300',
    dotColor: 'bg-gray-400',
    icon: 'clock',
  },
  processing: {
    label: 'Processing',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    textColor: 'text-blue-700 dark:text-blue-300',
    dotColor: 'bg-blue-500 animate-pulse',
    icon: 'spinner',
  },
  completed: {
    label: 'Completed',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    dotColor: 'bg-green-500',
    icon: 'check',
  },
  failed: {
    label: 'Failed',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
    icon: 'x-circle',
  },
  cancelled: {
    label: 'Cancelled',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    textColor: 'text-gray-500 dark:text-gray-400',
    dotColor: 'bg-gray-400',
    icon: 'ban',
  },
};
