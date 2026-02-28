// Settings feature types

export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  orgId: string;
  isActive: boolean;
  emailVerified: boolean;
  pageQuotaMonthly?: number;
  twoFactorEnabled: boolean;
  createdAt: string;
}

export type UserRole = 'user' | 'admin' | 'owner';

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

export interface OrgMember {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  isActive: boolean;
  emailVerified: boolean;
  createdAt: string;
  lastActive?: string;
}

export interface UpdateProfileRequest {
  name?: string;
  email?: string;
}

export interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
}

export interface UpdateOrganizationRequest {
  name?: string;
  settings?: Record<string, unknown>;
}

export interface InviteMemberRequest {
  email: string;
  role: UserRole;
}

export interface UpdateMemberRoleRequest {
  role: UserRole;
}

export interface PasswordFormValues {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
}

export interface ProfileFormValues {
  name: string;
  email: string;
}

export interface SettingsTab {
  value: 'profile' | 'security' | 'organization';
  label: string;
  icon?: string;
}

export interface NotificationMessage {
  type: 'success' | 'error';
  text: string;
}

export interface FormValidationError {
  field: string;
  message: string;
}
