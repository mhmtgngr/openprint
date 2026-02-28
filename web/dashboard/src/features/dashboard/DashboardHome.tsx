/**
 * DashboardHome Component - Main dashboard with stats cards, charts, and activity feed
 */

import { useQuery } from '@tanstack/react-query';
import { analyticsApi, jobsApi, agentsApi, printersApi } from '@/services/api';
import { StatCard } from './StatCard';
import { RecentActivity } from './RecentActivity';
import { JobChart } from './JobChart';
import type { DashboardStats, Activity, JobStatistics } from './types';
import { jobToActivity } from './types';

// Icons
const DocumentIcon = () => (
  <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
    />
  </svg>
);

const PrinterIcon = () => (
  <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
    />
  </svg>
);

const PageIcon = () => (
  <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
    />
  </svg>
);

const CheckIcon = () => (
  <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
    />
  </svg>
);

export const DashboardHome = ({ className = '' }: { className?: string }) => {
  // Fetch recent jobs for activity feed
  const { data: jobsData, isLoading: jobsLoading } = useQuery({
    queryKey: ['jobs', { limit: 10 }],
    queryFn: () => jobsApi.list({ limit: 10 }),
    staleTime: 30000,
  });

  // Fetch analytics data for statistics
  const { data: analyticsData, isLoading: analyticsLoading } = useQuery({
    queryKey: ['analytics', 'usage'],
    queryFn: () => analyticsApi.getUsage({ groupBy: 'day' }),
    staleTime: 60000,
  });

  // Fetch environment report for storage and pages
  const { data: envData, isLoading: envLoading } = useQuery({
    queryKey: ['analytics', 'environment'],
    queryFn: () => analyticsApi.getEnvironment('30d'),
    staleTime: 60000,
  });

  // Fetch devices for active count
  const { data: agentsData } = useQuery({
    queryKey: ['agents'],
    queryFn: () => agentsApi.list(),
    staleTime: 60000,
  });

  const { data: printersData } = useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
    staleTime: 60000,
  });

  // Calculate dashboard stats
  const stats: DashboardStats = {
    totalJobs: jobsData?.total || 0,
    completedJobs: jobsData?.data.filter((j) => j.status === 'completed').length || 0,
    queuedJobs: jobsData?.data.filter((j) => j.status === 'queued').length || 0,
    failedJobs: jobsData?.data.filter((j) => j.status === 'failed').length || 0,
    activeDevices:
      (agentsData?.filter((a) => a.status === 'online').length || 0) +
      (printersData?.filter((p) => p.isOnline).length || 0),
    totalDevices:
      (agentsData?.length || 0) + (printersData?.length || 0),
    storageUsed: envData?.pagesPrinted || 0, // Using pages as proxy for storage
    storageTotal: 1000000, // 1M pages as example limit
    pagesPrintedToday: analyticsData?.[0]?.pagesPrinted || 0,
    pagesPrintedMonth: envData?.pagesPrinted || 0,
  };

  // Convert jobs to activities
  const activities: Activity[] = jobsData?.data.map(jobToActivity) || [];

  // Prepare chart data from analytics
  const jobStatistics: JobStatistics = {
    period: 'week',
    data: (analyticsData || []).slice(0, 7).map((stat) => ({
      label: new Date(stat.statDate).toLocaleDateString('en-US', { weekday: 'short' }),
      value: stat.jobsCount,
      date: stat.statDate,
    })),
    total: analyticsData?.reduce((sum, s) => sum + s.jobsCount, 0) || 0,
    completed: analyticsData?.reduce((sum, s) => sum + (s.jobsCompleted || 0), 0) || 0,
    failed: analyticsData?.reduce((sum, s) => sum + (s.jobsFailed || 0), 0) || 0,
    averagePerDay: analyticsData?.reduce((sum, s) => sum + s.jobsCount, 0) || 0 / 7 || 0,
  };

  const loading = jobsLoading || analyticsLoading || envLoading;

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          Dashboard
        </h1>
        <p className="text-gray-600 dark:text-gray-400 mt-1">
          Overview of your print environment
        </p>
      </div>

      {/* Stats Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Jobs"
          value={stats.totalJobs}
          icon={DocumentIcon}
          color="blue"
          loading={loading}
        />
        <StatCard
          title="Active Devices"
          value={stats.activeDevices}
          unit={`/ ${stats.totalDevices}`}
          icon={PrinterIcon}
          color="green"
          loading={loading}
          trend={{
            value: 12,
            label: 'from last week',
            isPositive: true,
          }}
        />
        <StatCard
          title="Pages Today"
          value={stats.pagesPrintedToday}
          icon={PageIcon}
          color="purple"
          loading={loading}
          trend={{
            value: 8,
            label: 'vs yesterday',
            isPositive: true,
          }}
        />
        <StatCard
          title="Completed Jobs"
          value={stats.completedJobs}
          icon={CheckIcon}
          color="green"
          loading={loading}
          trend={{
            value: 5,
            label: 'completion rate',
            isPositive: true,
          }}
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Job Chart - spans 2 columns */}
        <div className="lg:col-span-2">
          <JobChart statistics={jobStatistics} loading={loading} />
        </div>

        {/* Recent Activity */}
        <div>
          <RecentActivity
            activities={activities}
            loading={jobsLoading}
            maxItems={6}
          />
        </div>
      </div>

      {/* Additional Stats Row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <StatCard
          title="Queued Jobs"
          value={stats.queuedJobs}
          color="orange"
          loading={loading}
        />
        <StatCard
          title="Failed Jobs"
          value={stats.failedJobs}
          color="red"
          loading={loading}
          trend={{
            value: 2,
            label: 'from last week',
            isPositive: false,
          }}
        />
        <StatCard
          title="Pages This Month"
          value={stats.pagesPrintedMonth}
          icon={PageIcon}
          color="blue"
          loading={loading}
        />
      </div>
    </div>
  );
};

export default DashboardHome;
