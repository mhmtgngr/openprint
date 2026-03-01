import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { useJobs } from '@/hooks/useJobs';
import { analyticsApi } from '@/services/api';
import { JobStatusBadge } from '@/components/JobStatusBadge';
import { EnvironmentReport } from '@/components/EnvironmentReport';
import { PrinterIcon } from '@/components/icons';
import type { Printer } from '@/types';

export const Dashboard = () => {
  const { user } = useAuth();
  const { data: jobs } = useJobs({ limit: 5 });
  const { data: printersResponse } = useQuery({
    queryKey: ['printers'],
    queryFn: () => fetch('/api/v1/printers').then(r => r.json()),
  });
  const printers = (printersResponse?.printers || []) as Printer[];
  const { data: environment } = useQuery({
    queryKey: ['environment'],
    queryFn: () => analyticsApi.getEnvironment('30d'),
  });

  const jobsList = jobs?.data || [];

  const stats = [
    {
      label: 'Active Printers',
      value: printers?.filter((p) => p.isOnline && p.isActive).length || 0,
      total: printers?.length || 0,
      color: 'text-blue-600 dark:text-blue-400',
      bgColor: 'bg-blue-100 dark:bg-blue-900/30',
      icon: PrinterIcon,
    },
    {
      label: 'Jobs Today',
      value: jobsList.filter((j) => {
        const today = new Date().toDateString();
        return new Date(j.createdAt).toDateString() === today;
      }).length || 0,
      total: jobs?.total || 0,
      color: 'text-green-600 dark:text-green-400',
      bgColor: 'bg-green-100 dark:bg-green-900/30',
      icon: () => (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
      ),
    },
    {
      label: 'Pages This Month',
      value: '1,234',
      total: '10,000',
      color: 'text-purple-600 dark:text-purple-400',
      bgColor: 'bg-purple-100 dark:bg-purple-900/30',
      icon: () => (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
          />
        </svg>
      ),
    },
  ];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
          Welcome back, {user?.name?.split(' ')[0]}!
        </h1>
        <p className="text-gray-600 dark:text-gray-400 mt-1">
          Here's what's happening with your print environment today.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div
              key={stat.label}
              className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
            >
              <div className="flex items-center gap-4">
                <div className={`${stat.bgColor} ${stat.color} p-3 rounded-lg`}>
                  <Icon />
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {typeof stat.value === 'number' ? stat.value : stat.value}
                  </p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">{stat.label}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Recent Jobs */}
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Recent Print Jobs
            </h2>
            <Link
              to="/jobs"
              className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
            >
              View all
            </Link>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {jobsList.length === 0 ? (
              <div className="p-6 text-center text-gray-500 dark:text-gray-400">
                No print jobs yet.{' '}
                <Link to="/printers" className="text-blue-600 dark:text-blue-400 hover:underline">
                  Select a printer
                </Link>{' '}
                to get started.
              </div>
            ) : (
              jobsList.slice(0, 5).map((job) => (
                <div
                  key={job.id}
                  className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                        {job.documentName}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        {job.pageCount} pages • {job.printer?.name || 'No printer'}
                      </p>
                    </div>
                    <JobStatusBadge status={job.status} />
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Printers */}
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Available Printers
            </h2>
            <Link
              to="/printers"
              className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
            >
              Manage
            </Link>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {printers?.length === 0 ? (
              <div className="p-6 text-center text-gray-500 dark:text-gray-400">
                No printers configured. Install the OpenPrint Agent to add printers.
              </div>
            ) : (
              printers?.slice(0, 5).map((printer) => (
                <div
                  key={printer.id}
                  className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className={`p-2 rounded-lg ${
                          printer.isOnline
                            ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                            : 'bg-gray-100 dark:bg-gray-700 text-gray-400'
                        }`}
                      >
                        <PrinterIcon className="w-5 h-5" />
                      </div>
                      <div>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {printer.name}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">
                          {printer.type}
                        </p>
                      </div>
                    </div>
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${
                        printer.isOnline
                          ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
                      }`}
                    >
                      {printer.isOnline ? 'Online' : 'Offline'}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Environmental Report */}
      {environment && <EnvironmentReport report={environment} />}
    </div>
  );
};
