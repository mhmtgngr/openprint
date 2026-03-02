// Authentication types for OpenPrint Cloud

export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  orgId: string;
  isActive: boolean;
  emailVerified: boolean;
  pageQuotaMonthly?: number;
  createdAt: string;
  isPlatformAdmin?: boolean;
}

export type UserRole = 'user' | 'admin' | 'owner' | 'platform_admin';

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
}

export interface AuthResponse {
  userId: string;
  access_token: string;
  refresh_token: string;
  org?: Organization;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  plan: OrganizationPlan;
  settings: Record<string, unknown>;
  maxUsers: number;
  maxPrinters: number;
  createdAt: string;
}

export type OrganizationPlan = 'free' | 'pro' | 'enterprise';

// Form component props
export interface LoginFormProps {
  onSubmit: (email: string, password: string) => Promise<void>;
  isLoading?: boolean;
  error?: string | null;
}

export interface RegisterFormProps {
  onSubmit: (name: string, email: string, password: string) => Promise<void>;
  isLoading?: boolean;
  error?: string | null;
}

// Protected route props
export interface ProtectedRouteProps {
  children: React.ReactNode;
  redirectTo?: string;
  requiredRoles?: UserRole[];
}

// Auth context value interface
export interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  error: string | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  hasRole: (roles: UserRole[]) => boolean;
}

// Auth state interface
export interface AuthState {
  user: User | null;
  isLoading: boolean;
  error: string | null;
  isAuthenticated: boolean;
}
