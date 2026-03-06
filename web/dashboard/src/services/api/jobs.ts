import type { PrintJob, CreateJobRequest, PaginatedResponse, JobHistoryEntry } from '@/types';
import { httpClient } from '@/services/http';

export const jobsApi = {
  async list(params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<PrintJob>> {
    return httpClient.get<PaginatedResponse<PrintJob>>('/jobs', params);
  },

  async get(id: string): Promise<PrintJob> {
    return httpClient.get<PrintJob>(`/jobs/${id}`);
  },

  async create(data: CreateJobRequest): Promise<PrintJob> {
    return httpClient.post<PrintJob>('/jobs', data);
  },

  async cancel(id: string): Promise<{ success: boolean }> {
    return httpClient.post<{ success: boolean }>(`/jobs/${id}/cancel`);
  },

  async retry(id: string): Promise<PrintJob> {
    return httpClient.post<PrintJob>(`/jobs/${id}/retry`);
  },

  async getHistory(id: string): Promise<JobHistoryEntry[]> {
    return httpClient.get<JobHistoryEntry[]>(`/jobs/${id}/history`);
  },
};
