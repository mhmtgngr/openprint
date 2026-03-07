import type { AuthResponse, LoginRequest, RegisterRequest, User } from '@/types';
import { httpClient, setTokens, clearTokens } from '@/services/http';

export const authApi = {
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const data = await httpClient.publicPost<AuthResponse>('/auth/login', credentials);
    setTokens({ accessToken: data.access_token, refreshToken: data.refresh_token });
    return data;
  },

  async register(data: RegisterRequest): Promise<AuthResponse> {
    const result = await httpClient.publicPost<AuthResponse>('/auth/register', data);
    setTokens({ accessToken: result.access_token, refreshToken: result.refresh_token });
    return result;
  },

  async logout(): Promise<void> {
    await httpClient.post<void>('/auth/logout');
    clearTokens();
  },

  async getCurrentUser(): Promise<User> {
    return httpClient.get<User>('/auth/me');
  },

  async initiateSSO(provider: string, redirectUri: string): Promise<{ sso_url: string }> {
    return httpClient.publicPost<{ sso_url: string }>('/auth/sso/initiate', {
      provider,
      redirect_uri: redirectUri,
    });
  },
};
