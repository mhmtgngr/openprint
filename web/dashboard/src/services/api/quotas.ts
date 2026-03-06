import type { UserQuota, QuotaPeriod, UpdateQuotaRequest, PaginatedResponse } from '@/types';
import { httpClient } from '@/services/http';

export const quotasApi = {
  async getUserQuota(userId?: string): Promise<UserQuota> {
    const path = userId ? `/quotas/users/${userId}` : '/quotas/me';
    return httpClient.get<UserQuota>(path);
  },

  async getOrgQuotas(): Promise<UserQuota[]> {
    return httpClient.get<UserQuota[]>('/quotas/organization');
  },

  async updateQuota(data: UpdateQuotaRequest): Promise<UserQuota> {
    return httpClient.patch<UserQuota>('/quotas', data);
  },

  async getPeriods(params?: { limit?: number; offset?: number }): Promise<PaginatedResponse<QuotaPeriod>> {
    return httpClient.get<PaginatedResponse<QuotaPeriod>>('/quotas/periods', params);
  },
};
