/**
 * JobDetails Component - Shows detailed information about a single print job
 */

import { formatDistanceToNow, format } from 'date-fns';
import { useJob, useJobHistory, useCancelJob, useRetryJob } from './useJobs';
import { JobStatusBadge } from './JobStatusBadge';
import type { PrintJob, JobHistoryEntry } from '@/types/jobs';

interface JobDetailsProps {
  jobId: string;
  onClose?: () => void;
  onJobUpdated?: (job: PrintJob) => void;
}

export const JobDetails = ({ jobId, onClose }: JobDetailsProps) => {
  const { data: job, isLoading, error, refetch } = useJob(jobId);
  const { data: history, isLoading: isLoadingHistory } = useJobHistory(jobId);
  const cancelJob = useCancelJob();
  const retryJob = useRetryJob();

  const handleCancel = async () => {
    if (confirm('Are you sure you want to cancel this print job?')) {
      try {
        await cancelJob.mutateAsync(jobId);
        refetch();
      } catch (err) {
        console.error('Failed to cancel job:', err);
      }
    }
  };

  const handleRetry = async () => {
    try {
      await retryJob.mutateAsync(jobId);
      refetch();
    } catch (err) {
      console.error('Failed to retry job:', err);
    }
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
          <div className="h-4 w-64 bg-gray-200 dark:bg-gray-700 rounded" />
          <div className="h-4 w-96 bg-gray-200 dark:bg-gray-700 rounded" />
        </div>
      </div>
    );
  }

  // Error state
  if (error || !job) {
    return (
      <div className="p-6 text-center">
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
          Failed to load job details
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          The job could not be found or an error occurred.
        </p>
      </div>
    );
  }

  const canCancel = job.status === 'queued';
  const canRetry = job.status === 'failed';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            {job.documentName}
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Job ID: <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">{job.id}</code>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <JobStatusBadge status={job.status} size="lg" />
          {onClose && (
            <button
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
            >
              <XIcon className="w-5 h-5" />
            </button>
          )}
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-3">
        {canCancel && (
          <button
            onClick={handleCancel}
            disabled={cancelJob.isPending}
            className="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <XMarkIcon className="w-4 h-4" />
            Cancel Job
          </button>
        )}
        {canRetry && (
          <button
            onClick={handleRetry}
            disabled={retryJob.isPending}
            className="inline-flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <RefreshIcon className="w-4 h-4" />
            Retry Job
          </button>
        )}
        <button
          onClick={() => refetch()}
          className="inline-flex items-center gap-2 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
        >
          <RefreshIcon className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Error message */}
      {job.errorMessage && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <div className="flex items-start gap-3">
            <ExclamationTriangleIcon className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <h4 className="text-sm font-medium text-red-800 dark:text-red-300">Error</h4>
              <p className="text-sm text-red-700 dark:text-red-400 mt-1">{job.errorMessage}</p>
            </div>
          </div>
        </div>
      )}

      {/* Details Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Job Info */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            Job Information
          </h3>
          <div className="p-4 space-y-3">
            <DetailRow label="Document Name" value={job.documentName} />
            <DetailRow label="Document Type" value={job.documentType || 'Unknown'} />
            <DetailRow label="File Size" value={formatFileSize(job.fileSize)} />
            <DetailRow label="File Hash" value={job.fileHash || 'N/A'} truncate />
            <DetailRow label="Storage Key" value={job.storageKey || 'N/A'} truncate />
          </div>
        </div>

        {/* Print Settings */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            Print Settings
          </h3>
          <div className="p-4 space-y-3">
            <DetailRow label="Page Count" value={`${job.pageCount} pages`} />
            <DetailRow
              label="Color Pages"
              value={job.colorPages !== undefined ? `${job.colorPages} pages` : 'N/A'}
            />
            <DetailRow
              label="Color Mode"
              value={job.settings.color ? 'Color' : 'Grayscale'}
            />
            <DetailRow
              label="Duplex"
              value={job.settings.duplex ? 'Double-sided' : 'Single-sided'}
            />
            <DetailRow label="Paper Size" value={job.settings.paperSize || 'Default'} />
            <DetailRow label="Copies" value={job.settings.copies?.toString() || '1'} />
            <DetailRow label="Quality" value={job.settings.quality || 'Standard'} />
            <DetailRow label="Orientation" value={job.settings.orientation || 'portrait'} />
          </div>
        </div>

        {/* Printer Info */}
        {job.printer && (
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
              Printer Information
            </h3>
            <div className="p-4 space-y-3">
              <DetailRow label="Printer Name" value={job.printer.name} />
              <DetailRow label="Printer Type" value={job.printer.type} />
              <DetailRow
                label="Status"
                value={job.printer.isOnline ? 'Online' : 'Offline'}
                valueClass={job.printer.isOnline ? 'text-green-600' : 'text-gray-500'}
              />
              {job.printer.capabilities && (
                <>
                  <DetailRow
                    label="Color Support"
                    value={job.printer.capabilities.supportsColor ? 'Yes' : 'No'}
                  />
                  <DetailRow
                    label="Duplex Support"
                    value={job.printer.capabilities.supportsDuplex ? 'Yes' : 'No'}
                  />
                  <DetailRow
                    label="Resolution"
                    value={job.printer.capabilities.resolution || 'Unknown'}
                  />
                </>
              )}
            </div>
          </div>
        )}

        {/* Timeline */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            Timeline
          </h3>
          <div className="p-4 space-y-3">
            <TimelineItem
              label="Created"
              date={job.createdAt}
              icon="clock"
            />
            {job.startedAt && (
              <TimelineItem
                label="Started"
                date={job.startedAt}
                icon="play"
              />
            )}
            {job.completedAt && (
              <TimelineItem
                label="Completed"
                date={job.completedAt}
                icon="check"
              />
            )}
            {job.autoDeleteAt && (
              <TimelineItem
                label="Auto Delete"
                date={job.autoDeleteAt}
                icon="trash"
              />
            )}
          </div>
        </div>
      </div>

      {/* Job Progress (for processing jobs) */}
      {job.status === 'processing' && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Printing Progress
            </span>
            <span className="text-sm text-gray-500 dark:text-gray-400">
              {job.progress || 0}%
            </span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              className="bg-blue-600 h-2 rounded-full transition-all duration-300"
              style={{ width: `${job.progress || 0}%` }}
            />
          </div>
        </div>
      )}

      {/* Job History */}
      {history && history.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
            Job History
          </h3>
          {isLoadingHistory ? (
            <div className="p-4 text-center text-sm text-gray-500 dark:text-gray-400">
              Loading history...
            </div>
          ) : (
            <div className="divide-y divide-gray-200 dark:divide-gray-700 max-h-64 overflow-y-auto">
              {history.map((entry) => (
                <HistoryEntry key={entry.id} entry={entry} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// Helper Components
const DetailRow = ({
  label,
  value,
  valueClass,
  truncate = false,
}: {
  label: string;
  value: string | number;
  valueClass?: string;
  truncate?: boolean;
}) => (
  <div className="flex justify-between items-start">
    <span className="text-sm text-gray-500 dark:text-gray-400">{label}</span>
    <span
      className={`text-sm text-gray-900 dark:text-gray-100 font-medium ${
        truncate ? 'max-w-xs truncate' : ''
      } ${valueClass || ''}`}
    >
      {value}
    </span>
  </div>
);

const TimelineItem = ({
  label,
  date,
  icon,
}: {
  label: string;
  date: string;
  icon: string;
}) => {
  const dateObj = new Date(date);
  const isPast = dateObj < new Date();

  return (
    <div className="flex items-start gap-3">
      <div className={`p-1.5 rounded-full ${isPast ? 'bg-green-100 dark:bg-green-900/30' : 'bg-gray-100 dark:bg-gray-800'}`}>
        {icon === 'clock' && <ClockIcon className="w-3.5 h-3.5 text-gray-600 dark:text-gray-400" />}
        {icon === 'play' && <PlayIcon className="w-3.5 h-3.5 text-blue-600 dark:text-blue-400" />}
        {icon === 'check' && <CheckIcon className="w-3.5 h-3.5 text-green-600 dark:text-green-400" />}
        {icon === 'trash' && <TrashIcon className="w-3.5 h-3.5 text-gray-600 dark:text-gray-400" />}
      </div>
      <div className="flex-1">
        <p className="text-sm text-gray-900 dark:text-gray-100">{label}</p>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          {format(dateObj, 'PPp')} ({formatDistanceToNow(dateObj, { addSuffix: true })})
        </p>
      </div>
    </div>
  );
};

const HistoryEntry = ({ entry }: { entry: JobHistoryEntry }) => (
  <div className="px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/50">
    <div className="flex items-start gap-3">
      <JobStatusBadge status={entry.status} size="sm" />
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-900 dark:text-gray-100">
          {entry.message || `Status changed to ${entry.status}`}
        </p>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
          {formatDistanceToNow(new Date(entry.timestamp), { addSuffix: true })}
        </p>
      </div>
    </div>
  </div>
);

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
};

// Icons
const XIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const XMarkIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const RefreshIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
  </svg>
);

const ExclamationTriangleIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>
);

const ClockIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const PlayIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const CheckIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
);

const TrashIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
  </svg>
);

export default JobDetails;
