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

  const onlinePrinters = printers?.filter((p) => p.isOnline && p.isActive).length || 0;
  const offlinePrinters = printers?.filter((p) => !p.isOnline && p.isActive).length || 0;
  const errorPrinters = printers?.filter((p) => p.isActive && !p.isOnline && p.lastSeen).length || 0;
  const todayJobs = jobsList.filter((j) => {
    const today = new Date().toDateString();
    return new Date(j.createdAt).toDateString() === today;
  });
  const completedToday = todayJobs.filter(j => j.status === 'completed').length;
  const failedToday = todayJobs.filter(j => j.status === 'failed').length;

  const stats = [
    {
      label: 'Active Printers',
      value: onlinePrinters,
      subtitle: `${offlinePrinters} offline`,
      color: 'text-blue-600 dark:text-blue-400',
      bgColor: 'bg-blue-100 dark:bg-blue-900/30',
      icon: PrinterIcon,
      link: '/printers',
    },
    {
      label: 'Jobs Today',
      value: todayJobs.length,
      subtitle: `${completedToday} done, ${failedToday} failed`,
      color: 'text-green-600 dark:text-green-400',
      bgColor: 'bg-green-100 dark:bg-green-900/30',
      icon: () => (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
      link: '/jobs',
    },
    {
      label: 'Total Pages',
      value: jobsList.reduce((sum, j) => sum + (j.pageCount || 0), 0).toLocaleString(),
      subtitle: `${jobsList.filter(j => j.settings?.color).length} color jobs`,
      color: 'text-purple-600 dark:text-purple-400',
      bgColor: 'bg-purple-100 dark:bg-purple-900/30',
      icon: () => (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
        </svg>
      ),
      link: '/analytics',
    },
    {
      label: 'Fleet Health',
      value: printers.length > 0 ? `${Math.round((onlinePrinters / printers.length) * 100)}%` : 'N/A',
      subtitle: `${errorPrinters} need attention`,
      color: onlinePrinters === printers.length ? 'text-green-600 dark:text-green-400' : 'text-orange-600 dark:text-orange-400',
      bgColor: onlinePrinters === printers.length ? 'bg-green-100 dark:bg-green-900/30' : 'bg-orange-100 dark:bg-orange-900/30',
      icon: () => (
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
      link: '/supplies',
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

      {/* Stats - now 4 columns */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Link
              key={stat.label}
              to={stat.link}
              className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-md transition-shadow"
            >
              <div className="flex items-center gap-4">
                <div className={`${stat.bgColor} ${stat.color} p-3 rounded-lg`}>
                  <Icon />
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stat.value}</p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">{stat.label}</p>
                  <p className="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{stat.subtitle}</p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Recent Jobs - takes 2 cols */}
        <div className="lg:col-span-2 bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Recent Print Jobs</h2>
            <Link to="/jobs" className="text-sm text-blue-600 dark:text-blue-400 hover:underline">View all</Link>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {jobsList.length === 0 ? (
              <div className="p-6 text-center text-gray-500 dark:text-gray-400">
                No print jobs yet. <Link to="/printers" className="text-blue-600 dark:text-blue-400 hover:underline">Select a printer</Link> to get started.
              </div>
            ) : (
              jobsList.slice(0, 5).map((job) => (
                <div key={job.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{job.documentName}</p>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        {job.pageCount} pages &middot; {job.printer?.name || 'No printer'} &middot; {new Date(job.createdAt).toLocaleTimeString()}
                      </p>
                    </div>
                    <JobStatusBadge status={job.status} />
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Quick Actions Sidebar */}
        <div className="space-y-6">
          {/* Fleet Status */}
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-4">Fleet Status</h3>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-green-500" />
                  <span className="text-sm text-gray-600 dark:text-gray-300">Online</span>
                </div>
                <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">{onlinePrinters}</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-gray-400" />
                  <span className="text-sm text-gray-600 dark:text-gray-300">Offline</span>
                </div>
                <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">{offlinePrinters}</span>
              </div>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full bg-red-500" />
                  <span className="text-sm text-gray-600 dark:text-gray-300">Errors</span>
                </div>
                <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">{errorPrinters}</span>
              </div>
            </div>
            {printers.length > 0 && (
              <div className="mt-4">
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className="bg-green-500 h-2 rounded-full transition-all"
                    style={{ width: `${(onlinePrinters / printers.length) * 100}%` }}
                  />
                </div>
                <p className="text-xs text-gray-400 mt-1">{Math.round((onlinePrinters / printers.length) * 100)}% availability</p>
              </div>
            )}
          </div>

          {/* Quick Actions */}
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-4">Quick Actions</h3>
            <div className="space-y-2">
              {[
                { label: 'Follow-Me Print', path: '/follow-me', desc: 'Print to any printer' },
                { label: 'Secure Release', path: '/secure-print', desc: 'Release held jobs' },
                { label: 'Guest Token', path: '/guest-printing', desc: 'Create visitor access' },
                { label: 'View Reports', path: '/analytics', desc: 'Usage analytics' },
              ].map(action => (
                <Link
                  key={action.path}
                  to={action.path}
                  className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors group"
                >
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100 group-hover:text-blue-600">{action.label}</p>
                    <p className="text-xs text-gray-400">{action.desc}</p>
                  </div>
                  <svg className="w-4 h-4 text-gray-400 group-hover:text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </Link>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Printers Grid */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Available Printers</h2>
          <Link to="/printers" className="text-sm text-blue-600 dark:text-blue-400 hover:underline">Manage</Link>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-gray-200 dark:divide-gray-700">
          {printers?.length === 0 ? (
            <div className="col-span-full p-6 text-center text-gray-500 dark:text-gray-400">
              No printers configured. Install the OpenPrint Agent to add printers.
            </div>
          ) : (
            printers?.slice(0, 6).map((printer) => (
              <div key={printer.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                <div className="flex items-center gap-3">
                  <div className={`p-2 rounded-lg ${printer.isOnline ? 'bg-green-100 dark:bg-green-900/30 text-green-600' : 'bg-gray-100 dark:bg-gray-700 text-gray-400'}`}>
                    <PrinterIcon className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{printer.name}</p>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className={`text-xs ${printer.isOnline ? 'text-green-600' : 'text-gray-400'}`}>
                        {printer.isOnline ? 'Online' : 'Offline'}
                      </span>
                      <span className="text-xs text-gray-400 capitalize">{printer.type}</span>
                    </div>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Environmental Report */}
      {environment && <EnvironmentReport report={environment} />}
    </div>
  );
};
