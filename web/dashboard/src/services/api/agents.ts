import type { Agent } from '@/types';
import { httpClient } from '@/services/http';

export const agentsApi = {
  async list(): Promise<Agent[]> {
    return httpClient.get<Agent[]>('/agents');
  },

  async get(id: string): Promise<Agent> {
    return httpClient.get<Agent>(`/agents/${id}`);
  },

  async update(id: string, data: Partial<Agent>): Promise<Agent> {
    return httpClient.patch<Agent>(`/agents/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return httpClient.delete<void>(`/agents/${id}`);
  },
};
