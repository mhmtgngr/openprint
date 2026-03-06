import type { Webhook, CreateWebhookRequest } from '@/types';
import { httpClient } from '@/services/http';

export const webhooksApi = {
  async list(): Promise<Webhook[]> {
    return httpClient.get<Webhook[]>('/webhooks');
  },

  async create(data: CreateWebhookRequest): Promise<Webhook> {
    return httpClient.post<Webhook>('/webhooks', data);
  },

  async update(id: string, data: Partial<Webhook>): Promise<Webhook> {
    return httpClient.patch<Webhook>(`/webhooks/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return httpClient.delete<void>(`/webhooks/${id}`);
  },

  async test(id: string): Promise<{ success: boolean }> {
    return httpClient.post<{ success: boolean }>(`/webhooks/${id}/test`);
  },
};
