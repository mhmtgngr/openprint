import { useState } from 'react';
import { useJobs } from '@/hooks/useJobs';
import { JobList } from '@/components/JobList';
import { JobStatusBadge } from '@/components/JobStatusBadge';
import { SearchIcon, FilterIcon, DownloadIcon } from './icons';
import type { JobStatus } from '@/types';

const statusFilters: { value: JobStatus | 'all'; label: string }[] = [
  { value: 'all', label: 'All Jobs' },
  { value: 'queued', label: 'Queued' },
  { value: 'processing', label: 'Processing' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
  { value: 'cancelled', label: 'Cancelled' },
];

export const Jobs = () => {
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<JobStatus | 'all'>('all');
  const [selectedJobs, setSelectedJobs] = useState<Set<string>>(new Set());

  const { data, isLoading, refetch } = useJobs({
    status: statusFilter === 'all' ? undefined : statusFilter,
    limit: 50,
  });

  const jobs = data?.data || [];

  const handleSelectAll = () => {
    if (selectedJobs.size === jobs.length) {
      setSelectedJobs(new Set());
    } else {
      setSelectedJobs(new Set(jobs.map((j) => j.id)));
    }
  };

  const handleSelectJob = (jobId: string) => {
    const newSelected = new Set(selectedJobs);
    if (newSelected.has(jobId)) {
      newSelected.delete(jobId);
    } else {
      newSelected.add(jobId);
    }
    setSelectedJobs(newSelected);
  };

  const filteredJobs = jobs.filter((job) =>
    job.documentName.toLowerCase().includes(search.toLowerCase()) ||
    job.printer?.name.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Print Jobs</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            View and manage your print job history
          </p>
        </div>
        <div className="flex items-center gap-3">
          {selectedJobs.size > 0 && (
            <button className="inline-flex items-center gap-2 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors">
              <DownloadIcon className="w-5 h-5" />
              Download ({selectedJobs.size})
            </button>
          )}
          <button
            onClick={() => refetch()}
            className="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors"
            title="Refresh"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        {statusFilters.map((filter) => {
          const count = filter.value === 'all'
            ? jobs.length
            : jobs.filter((j) => j.status === filter.value).length;

          return (
            <button
              key={filter.value}
              onClick={() => setStatusFilter(filter.value)}
              className={`
                bg-white dark:bg-gray-800 rounded-lg p-4 border-2 transition-colors text-left
                ${statusFilter === filter.value
                  ? 'border-blue-500 dark:border-blue-400'
                  : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                }
              `}
            >
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{count}</p>
              <p className="text-sm text-gray-500 dark:text-gray-400">{filter.label}</p>
            </button>
          );
        })}
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Search jobs by name or printer..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          />
        </div>
        <div className="flex items-center gap-2">
          <FilterIcon className="w-5 h-5 text-gray-400" />
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as JobStatus | 'all')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            {statusFilters.map((filter) => (
              <option key={filter.value} value={filter.value}>
                {filter.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Jobs List */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
        {filteredJobs.length > 0 && (
          <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex items-center gap-4">
            <input
              type="checkbox"
              checked={selectedJobs.size === filteredJobs.length && filteredJobs.length > 0}
              onChange={handleSelectAll}
              className="w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {selectedJobs.size > 0
                ? `${selectedJobs.size} job${selectedJobs.size > 1 ? 's' : ''} selected`
                : `${filteredJobs.length} total job${filteredJobs.length > 1 ? 's' : ''}`}
            </span>
          </div>
        )}
        <JobList jobs={filteredJobs} isLoading={isLoading} />
      </div>
    </div>
  );
};

const DownloadIcon = ({ className = '' }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
  </svg>
);
