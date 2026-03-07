import type {
  EmailToPrintConfig,
  EmailPrintJob,
  UpdateEmailConfigRequest,
  PaginatedResponse,
} from '@/types';
import { httpClient } from '@/services/http';

export const emailToPrintApi = {
  async getConfig(): Promise<EmailToPrintConfig> {
    return httpClient.get<EmailToPrintConfig>('/email-to-print/config');
  },

  async updateConfig(data: UpdateEmailConfigRequest): Promise<EmailToPrintConfig> {
    return httpClient.patch<EmailToPrintConfig>('/email-to-print/config', data);
  },

  async getJobs(params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<EmailPrintJob>> {
    return httpClient.get<PaginatedResponse<EmailPrintJob>>('/email-to-print/jobs', params);
  },

  async testEmail(): Promise<{ success: boolean; email: string }> {
    return httpClient.post<{ success: boolean; email: string }>('/email-to-print/test');
  },
};
