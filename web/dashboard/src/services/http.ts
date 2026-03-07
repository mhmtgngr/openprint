import type { APIError } from '@/types';

// --- Token management ---

let accessToken: string | null = null;
let refreshToken: string | null = null;

export const getAccessToken = (): string | null => {
  if (accessToken) return accessToken;
  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored);
    accessToken = tokens.accessToken;
    refreshToken = tokens.refreshToken;
    return accessToken;
  }
  return null;
};

export const setTokens = (tokens: { accessToken: string; refreshToken: string }): void => {
  accessToken = tokens.accessToken;
  refreshToken = tokens.refreshToken;
  localStorage.setItem('auth_tokens', JSON.stringify(tokens));
};

export const clearTokens = (): void => {
  accessToken = null;
  refreshToken = null;
  localStorage.removeItem('auth_tokens');
};

// --- Auth failure event ---

const AUTH_FAILURE_EVENT = 'openprint:auth_failure';

const fireAuthFailure = (): void => {
  window.dispatchEvent(new CustomEvent(AUTH_FAILURE_EVENT));
};

export const onAuthFailure = (handler: () => void): (() => void) => {
  window.addEventListener(AUTH_FAILURE_EVENT, handler);
  return () => window.removeEventListener(AUTH_FAILURE_EVENT, handler);
};

// --- API error ---

export class APIErrorClass extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(error: APIError) {
    super(error.message);
    this.name = 'APIError';
    this.code = error.code;
    this.details = error.details;
  }
}

// --- Interceptor types ---

export type RequestInterceptor = (url: string, options: RequestInit) => RequestInit;
export type ResponseInterceptor = (response: Response) => Response | Promise<Response>;

// --- HttpClient ---

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

const refreshAccessToken = async (): Promise<boolean> => {
  if (!refreshToken) {
    clearTokens();
    return false;
  }
  try {
    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!response.ok) {
      clearTokens();
      return false;
    }
    const data = await response.json();
    setTokens(data);
    return true;
  } catch {
    clearTokens();
    return false;
  }
};

const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = (await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }))) as APIError;
    throw new APIErrorClass(error);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
};

/**
 * Core HTTP client with automatic auth token injection, 401 refresh retry,
 * and pluggable interceptors.
 */
class HttpClient {
  private baseURL: string;
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];

  constructor(baseURL: string) {
    this.baseURL = baseURL;
  }

  /** Add a request interceptor that can modify URL/options before fetch. */
  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  /** Add a response interceptor that can transform the response. */
  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  private applyRequestInterceptors(url: string, options: RequestInit): RequestInit {
    let opts = options;
    for (const interceptor of this.requestInterceptors) {
      opts = interceptor(url, opts);
    }
    return opts;
  }

  private async applyResponseInterceptors(response: Response): Promise<Response> {
    let res = response;
    for (const interceptor of this.responseInterceptors) {
      res = await interceptor(res);
    }
    return res;
  }

  private async fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
    let token = getAccessToken();

    if (token) {
      options.headers = {
        ...options.headers,
        Authorization: `Bearer ${token}`,
      };
    }

    options = this.applyRequestInterceptors(url, options);
    let response = await fetch(url, options);

    if (response.status === 401 && token) {
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        token = getAccessToken();
        options.headers = {
          ...options.headers,
          Authorization: `Bearer ${token}`,
        };
        response = await fetch(url, options);
      } else {
        fireAuthFailure();
      }
    }

    return this.applyResponseInterceptors(response);
  }

  /** GET request with auth. */
  async get<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
    const searchParams = new URLSearchParams();
    if (params) {
      for (const [key, value] of Object.entries(params)) {
        if (value !== undefined) searchParams.set(key, String(value));
      }
    }
    const qs = searchParams.toString();
    const url = `${this.baseURL}${path}${qs ? `?${qs}` : ''}`;
    const response = await this.fetchWithAuth(url);
    return handleResponse<T>(response);
  }

  /** POST request with auth and JSON body. */
  async post<T>(path: string, body?: unknown): Promise<T> {
    const url = `${this.baseURL}${path}`;
    const options: RequestInit = {
      method: 'POST',
      ...(body !== undefined && {
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    };
    const response = await this.fetchWithAuth(url, options);
    return handleResponse<T>(response);
  }

  /** PATCH request with auth and JSON body. */
  async patch<T>(path: string, body: unknown): Promise<T> {
    const url = `${this.baseURL}${path}`;
    const response = await this.fetchWithAuth(url, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    return handleResponse<T>(response);
  }

  /** DELETE request with auth. */
  async delete<T>(path: string): Promise<T> {
    const url = `${this.baseURL}${path}`;
    const response = await this.fetchWithAuth(url, { method: 'DELETE' });
    return handleResponse<T>(response);
  }

  /** Raw fetch with auth (for non-JSON requests like file uploads). */
  async raw(path: string, options: RequestInit = {}): Promise<Response> {
    const url = `${this.baseURL}${path}`;
    return this.fetchWithAuth(url, options);
  }

  /** Public fetch without auth (for login/register). */
  async publicPost<T>(path: string, body: unknown): Promise<T> {
    const url = `${this.baseURL}${path}`;
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    return handleResponse<T>(response);
  }
}

/** Singleton HTTP client instance. */
export const httpClient = new HttpClient(API_BASE_URL);
