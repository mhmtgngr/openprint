import type { User, UpdateUserRequest } from '@/types';
import { httpClient } from '@/services/http';

export const userApi = {
  async update(data: UpdateUserRequest): Promise<User> {
    return httpClient.patch<User>('/users/me', data);
  },

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    return httpClient.post<void>('/users/me/password', { currentPassword, newPassword });
  },
};
