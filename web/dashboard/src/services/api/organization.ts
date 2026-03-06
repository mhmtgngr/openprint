import type {
  Organization,
  User,
  Invitation,
  InviteUserRequest,
  UpdateOrganizationRequest,
} from '@/types';
import { httpClient } from '@/services/http';

export const organizationApi = {
  async get(): Promise<Organization> {
    return httpClient.get<Organization>('/organizations');
  },

  async update(data: UpdateOrganizationRequest): Promise<Organization> {
    return httpClient.patch<Organization>('/organizations', data);
  },

  async getUsers(): Promise<User[]> {
    return httpClient.get<User[]>('/organizations/users');
  },

  async inviteUser(data: InviteUserRequest): Promise<Invitation> {
    return httpClient.post<Invitation>('/organizations/invitations', data);
  },

  async removeUser(userId: string): Promise<void> {
    return httpClient.delete<void>(`/organizations/users/${userId}`);
  },

  async updateUserRole(userId: string, role: string): Promise<User> {
    return httpClient.patch<User>(`/organizations/users/${userId}/role`, { role });
  },

  async getInvitations(): Promise<Invitation[]> {
    return httpClient.get<Invitation[]>('/organizations/invitations');
  },

  async cancelInvitation(invitationId: string): Promise<void> {
    return httpClient.delete<void>(`/organizations/invitations/${invitationId}`);
  },
};
