import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { printReleaseApi, printersApi, jobsApi } from '@/services/api';
import type { PrintJob } from '@/types';

export const PrintReleasePage = () => {
  const queryClient = useQueryClient();
  const [selectedPrinter, setSelectedPrinter] = useState<string>('');
  const [pin, setPin] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const { isLoading } = useQuery({
    queryKey: ['releases', 'pending'],
    queryFn: () => printReleaseApi.getPendingReleases(),
    refetchInterval: 5000, // Poll every 5 seconds
  });

  const { data: printers } = useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
  });

  const { data: jobs } = useQuery({
    queryKey: ['jobs'],
    queryFn: () => jobsApi.list({ status: 'queued', limit: 100 }),
  });

  const releaseMutation = useMutation({
    mutationFn: (data: { jobId: string; pin: string; printerId: string }) =>
      printReleaseApi.releaseJob(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['releases'] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      setPin('');
      setSuccess('Job released successfully!');
      setTimeout(() => setSuccess(''), 3000);
    },
    onError: (err: Error) => {
      setError(err.message || 'Failed to release job');
    },
  });

  const cancelMutation = useMutation({
    mutationFn: (jobId: string) => printReleaseApi.cancelRelease(jobId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['releases'] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });

  const handleRelease = (jobId: string) => {
    setError('');
    if (!selectedPrinter) {
      setError('Please select a printer');
      return;
    }
    if (!pin) {
      setError('Please enter your PIN');
      return;
    }
    releaseMutation.mutate({ jobId, pin, printerId: selectedPrinter });
  };

  const pendingJobs = jobs?.data.filter(
    (job) => !job.printerId && job.status === 'queued'
  ) || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            Secure Print Release
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Release your secure print jobs at any enabled printer
          </p>
        </div>
      </div>

      {/* Release Station */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Release Station
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Select a printer and enter your PIN to release print jobs
          </p>
        </div>
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Select Printer
              </label>
              <select
                value={selectedPrinter}
                onChange={(e) => setSelectedPrinter(e.target.value)}
                className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="">Choose a printer...</option>
                {printers?.filter((p) => p.isOnline && p.isActive).map((printer) => (
                  <option key={printer.id} value={printer.id}>
                    {printer.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Release PIN
              </label>
              <input
                type="password"
                value={pin}
                onChange={(e) => setPin(e.target.value)}
                maxLength={6}
                placeholder="Enter 6-digit PIN"
                className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent tracking-widest text-center text-lg"
              />
            </div>
            <div className="flex items-end">
              <button
                onClick={() => pendingJobs[0] && handleRelease(pendingJobs[0].id)}
                disabled={!selectedPrinter || !pin || pendingJobs.length === 0 || releaseMutation.isPending}
                className="w-full px-4 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed font-medium"
              >
                {releaseMutation.isPending ? 'Releasing...' : `Release All (${pendingJobs.length})`}
              </button>
            </div>
          </div>

          {error && (
            <div className="p-4 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 rounded-lg flex items-center gap-2 mb-4">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {error}
            </div>
          )}

          {success && (
            <div className="p-4 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-lg flex items-center gap-2">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              {success}
            </div>
          )}
        </div>
      </div>

      {/* Pending Jobs */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Pending Jobs ({pendingJobs.length})
          </h2>
        </div>
        <div className="divide-y divide-gray-200 dark:divide-gray-700">
          {isLoading ? (
            <div className="p-12 text-center text-gray-500 dark:text-gray-400">
              Loading pending jobs...
            </div>
          ) : pendingJobs.length === 0 ? (
            <div className="p-12 text-center">
              <svg className="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                No pending jobs
              </h3>
              <p className="text-gray-500 dark:text-gray-400">
                Secure print jobs will appear here when you send them to the print queue
              </p>
            </div>
          ) : (
            pendingJobs.map((job) => (
              <PendingJobCard
                key={job.id}
                job={job}
                onRelease={() => handleRelease(job.id)}
                onCancel={() => cancelMutation.mutate(job.id)}
                isReleasing={releaseMutation.isPending}
                isCancelling={cancelMutation.isPending}
              />
            ))
          )}
        </div>
      </div>
    </div>
  );
};

interface PendingJobCardProps {
  job: PrintJob;
  onRelease: () => void;
  onCancel: () => void;
  isReleasing: boolean;
  isCancelling: boolean;
}

const PendingJobCard = ({ job, onRelease, onCancel, isReleasing, isCancelling }: PendingJobCardProps) => {
  return (
    <div className="p-6 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-4">
          <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
            <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </div>
          <div>
            <h3 className="text-base font-medium text-gray-900 dark:text-gray-100">
              {job.documentName}
            </h3>
            <div className="flex items-center gap-4 mt-1 text-sm text-gray-500 dark:text-gray-400">
              <span>{job.pageCount} pages</span>
              <span>•</span>
              <span>{(job.fileSize / 1024).toFixed(0)} KB</span>
              {job.settings.color && (
                <>
                  <span>•</span>
                  <span className="text-blue-600 dark:text-blue-400">Color</span>
                </>
              )}
              {job.settings.duplex && (
                <>
                  <span>•</span>
                  <span>Duplex</span>
                </>
              )}
            </div>
            <p className="text-xs text-gray-400 mt-1">
              Queued {new Date(job.createdAt).toLocaleString()}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onCancel}
            disabled={isCancelling}
            className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={onRelease}
            disabled={isReleasing}
            className="px-3 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50"
          >
            Release
          </button>
        </div>
      </div>
    </div>
  );
};
