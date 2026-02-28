/**
 * Dashboard feature exports
 */

export { DashboardHome } from './DashboardHome';
export { StatCard } from './StatCard';
export { RecentActivity } from './RecentActivity';
export { JobChart } from './JobChart';

export type {
  DashboardStats,
  Activity,
  ActivityType,
  ChartDataPoint,
  JobStatistics,
  QuickAction,
  ActivityConfig,
} from './types';

export { ACTIVITY_CONFIG, jobToActivity } from './types';
