import type { PrintPolicy, CreatePolicyRequest } from '@/types';
import { httpClient } from '@/services/http';

export const policiesApi = {
  async list(): Promise<PrintPolicy[]> {
    return httpClient.get<PrintPolicy[]>('/policies');
  },

  async get(id: string): Promise<PrintPolicy> {
    return httpClient.get<PrintPolicy>(`/policies/${id}`);
  },

  async create(data: CreatePolicyRequest): Promise<PrintPolicy> {
    return httpClient.post<PrintPolicy>('/policies', data);
  },

  async update(id: string, data: Partial<CreatePolicyRequest>): Promise<PrintPolicy> {
    return httpClient.patch<PrintPolicy>(`/policies/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return httpClient.delete<void>(`/policies/${id}`);
  },

  async toggle(id: string, isEnabled: boolean): Promise<PrintPolicy> {
    return httpClient.post<PrintPolicy>(`/policies/${id}/toggle`, { isEnabled });
  },

  async reorder(policyIds: string[]): Promise<void> {
    return httpClient.post<void>('/policies/reorder', { policyIds });
  },
};
