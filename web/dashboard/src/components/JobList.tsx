import { formatDistanceToNow } from 'date-fns';
import type { PrintJob } from '@/types';
import { JobStatusBadge } from './JobStatusBadge';
import { useCancelJob, useRetryJob } from '@/hooks/useJobs';

interface JobListProps {
  jobs: PrintJob[];
  isLoading?: boolean;
  onJobClick?: (job: PrintJob) => void;
  selectedJobs?: Set<string>;
  onJobSelect?: (jobId: string) => void;
}

// Safe date formatting helper
const formatJobDate = (dateString: string | undefined): string => {
  if (!dateString) return 'Unknown';
  const date = new Date(dateString);
  if (isNaN(date.getTime())) return 'Invalid date';
  return formatDistanceToNow(date, { addSuffix: true });
};

export const JobList = ({ jobs, isLoading, onJobClick, selectedJobs = new Set(), onJobSelect }: JobListProps) => {
  const cancelJobMutation = useCancelJob();
  const retryJobMutation = useRetryJob();

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(5)].map((_, i) => (
          <div
            key={i}
            className="bg-gray-100 dark:bg-gray-800 rounded-lg h-24 animate-pulse"
          />
        ))}
      </div>
    );
  }

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
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">No print jobs</h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Get started by submitting a new print job.
        </p>
      </div>
    );
  }

  const handleCancel = async (e: React.MouseEvent, jobId: string) => {
    e.stopPropagation();
    if (confirm('Are you sure you want to cancel this print job?')) {
      await cancelJobMutation.mutateAsync(jobId);
    }
  };

  const handleRetry = async (e: React.MouseEvent, jobId: string) => {
    e.stopPropagation();
    await retryJobMutation.mutateAsync(jobId);
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  return (
    <div className="overflow-hidden">
      <ul className="divide-y divide-gray-200 dark:divide-gray-700">
        {jobs.map((job) => (
          <li
            key={job.id}
            className={`
              py-4 px-4 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors
              ${onJobClick ? 'cursor-pointer' : ''}
            `}
            onClick={() => onJobClick?.(job)}
          >
            <div className="flex items-center gap-3">
              {onJobSelect && (
                <input
                  type="checkbox"
                  checked={selectedJobs.has(job.id)}
                  onChange={() => onJobSelect(job.id)}
                  onClick={(e) => e.stopPropagation()}
                  className="w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              )}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-3">
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                    {job.documentName}
                  </p>
                  <JobStatusBadge status={job.status} />
                </div>
                <div className="mt-1 flex items-center gap-4 text-sm text-gray-500 dark:text-gray-400">
                  {job.printer && (
                    <span className="flex items-center gap-1">
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
                        />
                      </svg>
                      {job.printer.name}
                    </span>
                  )}
                  <span className="flex items-center gap-1">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                      />
                    </svg>
                    {job.pageCount} {job.pageCount === 1 ? 'page' : 'pages'}
                  </span>
                  <span className="flex items-center gap-1">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                      />
                    </svg>
                    {formatFileSize(job.fileSize)}
                  </span>
                  <span className="flex items-center gap-1">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                    {formatJobDate(job.createdAt)}
                  </span>
                </div>
                {job.errorMessage && (
                  <p className="mt-1 text-sm text-red-600 dark:text-red-400">{job.errorMessage}</p>
                )}
              </div>

              <div className="flex items-center gap-2">
                {job.status === 'queued' && (
                  <button
                    onClick={(e) => handleCancel(e, job.id)}
                    className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                    title="Cancel job"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                )}
                {job.status === 'failed' && (
                  <button
                    onClick={(e) => handleRetry(e, job.id)}
                    className="p-2 text-gray-400 hover:text-green-600 dark:hover:text-green-400 transition-colors"
                    title="Retry job"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                      />
                    </svg>
                  </button>
                )}
                <svg
                  className="w-5 h-5 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
};
