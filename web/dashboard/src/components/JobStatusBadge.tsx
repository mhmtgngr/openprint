import type { JobStatus } from '@/types';

interface JobStatusBadgeProps {
  status: JobStatus;
  className?: string;
}

const statusConfig: Record<
  JobStatus,
  { label: string; bgColor: string; textColor: string; dotColor: string }
> = {
  queued: {
    label: 'Queued',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    textColor: 'text-gray-700 dark:text-gray-300',
    dotColor: 'bg-gray-400',
  },
  processing: {
    label: 'Processing',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    textColor: 'text-blue-700 dark:text-blue-300',
    dotColor: 'bg-blue-500 animate-pulse',
  },
  completed: {
    label: 'Completed',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    dotColor: 'bg-green-500',
  },
  failed: {
    label: 'Failed',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    textColor: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
  },
  cancelled: {
    label: 'Cancelled',
    bgColor: 'bg-gray-100 dark:bg-gray-800',
    textColor: 'text-gray-500 dark:text-gray-400',
    dotColor: 'bg-gray-400',
  },
};

export const JobStatusBadge = ({ status, className = '' }: JobStatusBadgeProps) => {
  const config = statusConfig[status];

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${config.bgColor} ${config.textColor} ${className}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${config.dotColor}`} />
      {config.label}
    </span>
  );
};
