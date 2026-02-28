/**
 * JobList Component - Displays print jobs in a table with status badges
 */

import { formatDistanceToNow } from 'date-fns';
import type { PrintJob } from '@/types/jobs';
import { JobStatusBadge } from './JobStatusBadge';
import { useCancelJob, useRetryJob } from './useJobs';

interface JobListProps {
  jobs: PrintJob[];
  isLoading?: boolean;
  error?: string | null;
  onJobClick?: (job: PrintJob) => void;
  selectedJobs?: Set<string>;
  onSelectionChange?: (selectedIds: Set<string>) => void;
  emptyMessage?: string;
  emptyDescription?: string;
}

export const JobList = ({
  jobs,
  isLoading = false,
  error = null,
  onJobClick,
  selectedJobs = new Set(),
  onSelectionChange,
  emptyMessage = 'No print jobs',
  emptyDescription = 'Get started by submitting a new print job.',
}: JobListProps) => {
  const cancelJobMutation = useCancelJob();
  const retryJobMutation = useRetryJob();

  // Loading state
  if (isLoading) {
    return (
      <div className="w-full overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th className="px-4 py-3 w-10">
                <div className="h-4 w-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
              </th>
              <th className="px-4 py-3 text-left">
                <div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
              </th>
              <th className="px-4 py-3 text-left">
                <div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
              </th>
              <th className="px-4 py-3 text-left">
                <div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
              </th>
              <th className="px-4 py-3 text-left">
                <div className="h-4 w-24 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
              </th>
              <th className="px-4 py-3 text-right">
                <div className="h-4 w-20 bg-gray-200 dark:bg-gray-700 rounded animate-pulse ml-auto" />
              </th>
            </tr>
          </thead>
          <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
            {[...Array(5)].map((_, i) => (
              <tr key={i}>
                <td className="px-4 py-4">
                  <div className="h-4 w-4 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
                </td>
                <td className="px-4 py-4">
                  <div className="h-5 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
                </td>
                <td className="px-4 py-4">
                  <div className="h-5 w-24 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
                </td>
                <td className="px-4 py-4">
                  <div className="h-5 w-16 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
                </td>
                <td className="px-4 py-4">
                  <div className="h-5 w-20 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
                </td>
                <td className="px-4 py-4">
                  <div className="h-8 w-20 bg-gray-200 dark:bg-gray-700 rounded animate-pulse ml-auto" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-red-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          Error loading jobs
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{error}</p>
      </div>
    );
  }

  // Empty state
  if (jobs.length === 0) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-gray-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          {emptyMessage}
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {emptyDescription}
        </p>
      </div>
    );
  }

  const handleSelectAll = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newSelected = e.target.checked
      ? new Set(jobs.map((j) => j.id))
      : new Set<string>();
    onSelectionChange?.(newSelected);
  };

  const handleSelectJob = (jobId: string, checked: boolean) => {
    const newSelected = new Set(selectedJobs);
    if (checked) {
      newSelected.add(jobId);
    } else {
      newSelected.delete(jobId);
    }
    onSelectionChange?.(newSelected);
  };

  const handleCancel = async (e: React.MouseEvent, jobId: string) => {
    e.stopPropagation();
    if (confirm('Are you sure you want to cancel this print job?')) {
      try {
        await cancelJobMutation.mutateAsync(jobId);
      } catch (err) {
        console.error('Failed to cancel job:', err);
      }
    }
  };

  const handleRetry = async (e: React.MouseEvent, jobId: string) => {
    e.stopPropagation();
    try {
      await retryJobMutation.mutateAsync(jobId);
    } catch (err) {
      console.error('Failed to retry job:', err);
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  const isAllSelected = jobs.length > 0 && selectedJobs.size === jobs.length;
  const isSomeSelected = selectedJobs.size > 0 && !isAllSelected;

  return (
    <div className="w-full overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-800">
          <tr>
            <th className="px-4 py-3 w-10">
              <input
                type="checkbox"
                checked={isAllSelected}
                ref={isSomeSelected ? (input) => {
                  if (input) input.indeterminate = true;
                } : undefined}
                onChange={handleSelectAll}
                className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600"
              />
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Document
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Printer
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Status
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Pages / Size
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Created
            </th>
            <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
          {jobs.map((job) => {
            const isSelected = selectedJobs.has(job.id);
            const canCancel = job.status === 'queued';
            const canRetry = job.status === 'failed';

            return (
              <tr
                key={job.id}
                className={`
                  hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors
                  ${onJobClick ? 'cursor-pointer' : ''}
                  ${isSelected ? 'bg-blue-50 dark:bg-blue-900/20' : ''}
                `}
                onClick={() => onJobClick?.(job)}
              >
                <td className="px-4 py-4" onClick={(e) => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onChange={(e) => handleSelectJob(job.id, e.target.checked)}
                    className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600"
                  />
                </td>
                <td className="px-4 py-4">
                  <div className="flex flex-col">
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate max-w-xs">
                      {job.documentName}
                    </span>
                    {job.errorMessage && (
                      <span className="text-xs text-red-600 dark:text-red-400 truncate max-w-xs mt-1">
                        {job.errorMessage}
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-4">
                  {job.printer ? (
                    <div className="flex items-center gap-2">
                      <PrinterIcon className="w-4 h-4 text-gray-400" />
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        {job.printer.name}
                      </span>
                    </div>
                  ) : (
                    <span className="text-sm text-gray-400 dark:text-gray-500 italic">
                      No printer
                    </span>
                  )}
                </td>
                <td className="px-4 py-4">
                  <JobStatusBadge status={job.status} />
                </td>
                <td className="px-4 py-4">
                  <div className="flex flex-col gap-1">
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      {job.pageCount} {job.pageCount === 1 ? 'page' : 'pages'}
                      {job.colorPages !== undefined && job.colorPages > 0 && (
                        <span className="text-gray-400"> ({job.colorPages} color)</span>
                      )}
                    </span>
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      {formatFileSize(job.fileSize)}
                    </span>
                  </div>
                </td>
                <td className="px-4 py-4">
                  <span className="text-sm text-gray-500 dark:text-gray-400">
                    {formatDistanceToNow(new Date(job.createdAt), { addSuffix: true })}
                  </span>
                </td>
                <td className="px-4 py-4" onClick={(e) => e.stopPropagation()}>
                  <div className="flex items-center justify-end gap-1">
                    {canCancel && (
                      <button
                        onClick={(e) => handleCancel(e, job.id)}
                        disabled={cancelJobMutation.isPending}
                        className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Cancel job"
                      >
                        <XIcon className="w-5 h-5" />
                      </button>
                    )}
                    {canRetry && (
                      <button
                        onClick={(e) => handleRetry(e, job.id)}
                        disabled={retryJobMutation.isPending}
                        className="p-2 text-gray-400 hover:text-green-600 dark:hover:text-green-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        title="Retry job"
                      >
                        <RefreshIcon className="w-5 h-5" />
                      </button>
                    )}
                    {onJobClick && (
                      <button
                        className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
                        title="View details"
                      >
                        <ChevronRightIcon className="w-5 h-5" />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

// Icons
const PrinterIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
    />
  </svg>
);

const XIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const RefreshIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
    />
  </svg>
);

const ChevronRightIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
  </svg>
);
