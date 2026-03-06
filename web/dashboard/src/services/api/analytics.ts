import type {
  UsageStats,
  EnvironmentReport,
  UsageAnalyticsParams,
  PaginatedResponse,
  AuditLog,
} from '@/types';
import { httpClient } from '@/services/http';

export const analyticsApi = {
  async getUsage(params?: UsageAnalyticsParams): Promise<UsageStats[]> {
    return httpClient.get<UsageStats[]>('/analytics/usage', {
      start_date: params?.startDate,
      end_date: params?.endDate,
      group_by: params?.groupBy,
    });
  },

  async getEnvironment(period: string = '30d'): Promise<EnvironmentReport> {
    return httpClient.get<EnvironmentReport>('/analytics/environment', { period });
  },

  async getAuditLogs(params?: {
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<AuditLog>> {
    return httpClient.get<PaginatedResponse<AuditLog>>('/analytics/audit-logs', params);
  },
};
