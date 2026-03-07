import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi, clearTokens, getAccessToken, onAuthFailure } from '@/services/api';
import { wsService } from '@/services/websocket';
import type { User, UserRole, AuthContextValue } from '@/types/auth';

interface AuthState {
  user: User | null;
  isLoading: boolean;
  error: string | null;
  isAuthenticated: boolean;
}

let authState: AuthState = {
  user: null,
  isLoading: true,
  error: null,
  isAuthenticated: false,
};

const listeners = new Set<() => void>();

const notifyListeners = () => {
  listeners.forEach((listener) => listener());
};

const setAuthState = (updates: Partial<AuthState>) => {
  authState = { ...authState, ...updates };
  notifyListeners();
};

export const useAuth = (): AuthContextValue => {
  const navigate = useNavigate();
  const [, forceUpdate] = useState({});

  useEffect(() => {
    const listener = () => forceUpdate({});
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }, []);

  // Listen for global auth failure (401 after token refresh fails)
  useEffect(() => {
    return onAuthFailure(() => {
      wsService.disconnect();
      setAuthState({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        error: null,
      });
      navigate('/login');
    });
  }, [navigate]);

  useEffect(() => {
    const initAuth = async () => {
      const token = getAccessToken();
      if (token) {
        try {
          const user = await authApi.getCurrentUser();
          setAuthState({
            user,
            isLoading: false,
            isAuthenticated: true,
            error: null,
          });
          wsService.connect(token);
        } catch {
          clearTokens();
          setAuthState({
            user: null,
            isLoading: false,
            isAuthenticated: false,
            error: null,
          });
        }
      } else {
        setAuthState({
          user: null,
          isLoading: false,
          isAuthenticated: false,
          error: null,
        });
      }
    };

    if (authState.isLoading) {
      initAuth();
    }
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    setAuthState({ isLoading: true, error: null });
    try {
      const response = await authApi.login({ email, password });
      const user = await authApi.getCurrentUser();

      setAuthState({
        user,
        isLoading: false,
        isAuthenticated: true,
        error: null,
      });

      wsService.connect(response.access_token);
      navigate('/dashboard');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Login failed';
      setAuthState({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        error: message,
      });
      throw error;
    }
  }, [navigate]);

  const register = useCallback(async (email: string, password: string, name: string) => {
    setAuthState({ isLoading: true, error: null });
    try {
      const response = await authApi.register({ email, password, name });
      const user = await authApi.getCurrentUser();

      setAuthState({
        user,
        isLoading: false,
        isAuthenticated: true,
        error: null,
      });

      wsService.connect(response.access_token);
      navigate('/dashboard');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Registration failed';
      setAuthState({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        error: message,
      });
      throw error;
    }
  }, [navigate]);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // Ignore logout errors
    } finally {
      wsService.disconnect();
      setAuthState({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        error: null,
      });
      navigate('/login');
    }
  }, [navigate]);

  const hasRole = useCallback((roles: UserRole[]): boolean => {
    return authState.user !== null && roles.includes(authState.user.role);
  }, []);

  return {
    ...authState,
    login,
    register,
    logout,
    hasRole,
  };
};

interface RequireAuthReturn {
  isAuthenticated: boolean;
  isLoading: boolean;
}

export const useRequireAuth = (redirectTo: string = '/login'): RequireAuthReturn => {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate(redirectTo);
    }
  }, [isAuthenticated, isLoading, navigate, redirectTo]);

  return { isAuthenticated, isLoading };
};
