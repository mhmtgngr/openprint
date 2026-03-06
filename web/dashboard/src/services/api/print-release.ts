import type { PrintJob, PrintRelease, ReleaseJobRequest, CreateSecureJobRequest } from '@/types';
import { httpClient } from '@/services/http';

export const printReleaseApi = {
  async createSecureJob(data: CreateSecureJobRequest): Promise<PrintJob> {
    return httpClient.post<PrintJob>('/jobs/secure', data);
  },

  async getPendingReleases(): Promise<PrintRelease[]> {
    return httpClient.get<PrintRelease[]>('/releases/pending');
  },

  async releaseJob(data: ReleaseJobRequest): Promise<PrintJob> {
    return httpClient.post<PrintJob>('/releases/release', data);
  },

  async cancelRelease(jobId: string): Promise<void> {
    return httpClient.delete<void>(`/releases/${jobId}`);
  },
};
