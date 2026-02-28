// Settings API calls

import type {
  User,
  Organization,
  OrgMember,
  UpdateProfileRequest,
  ChangePasswordRequest,
  UpdateOrganizationRequest,
  InviteMemberRequest,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

const getAccessToken = (): string | null => {
  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored) as { accessToken: string; refreshToken: string };
    return tokens.accessToken;
  }
  return null;
};

const fetchWithAuth = async (url: string, options: RequestInit = {}): Promise<Response> => {
  const token = getAccessToken();

  if (token) {
    options.headers = {
      ...options.headers,
      Authorization: `Bearer ${token}`,
    };
  }

  return fetch(url, options);
};

const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }));
    throw new Error(error.message || 'Request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

// Profile API
export const updateProfile = async (data: UpdateProfileRequest): Promise<User> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/user/profile`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return handleResponse<User>(response);
};

export const changePassword = async (data: ChangePasswordRequest): Promise<void> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/user/password`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return handleResponse<void>(response);
};

export const toggleTwoFactor = async (enabled: boolean): Promise<{ success: boolean; qrCode?: string }> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/user/two-factor`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  });
  return handleResponse<{ success: boolean; qrCode?: string }>(response);
};

// Organization API
export const getOrganization = async (): Promise<Organization> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/organization`);
  return handleResponse<Organization>(response);
};

export const updateOrganization = async (data: UpdateOrganizationRequest): Promise<Organization> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/organization`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return handleResponse<Organization>(response);
};

export const getOrganizationMembers = async (): Promise<OrgMember[]> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/organization/members`);
  return handleResponse<OrgMember[]>(response);
};

export const inviteMember = async (data: InviteMemberRequest): Promise<{ invitationId: string; email: string }> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/organization/invitations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return handleResponse<{ invitationId: string; email: string }>(response);
};

export const updateMemberRole = async (memberId: string, role: string): Promise<OrgMember> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/organization/members/${memberId}/role`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  });
  return handleResponse<OrgMember>(response);
};

export const removeMember = async (memberId: string): Promise<void> => {
  const response = await fetchWithAuth(`${API_BASE_URL}/organization/members/${memberId}`, {
    method: 'DELETE',
  });
  return handleResponse<void>(response);
};
