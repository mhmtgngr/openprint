// Multi-tenancy organization types for OpenPrint Cloud

/**
 * Organization entity representing a tenant in the system
 */
export interface Organization {
  id: string;
  name: string;
  slug: string;
  displayName?: string;
  status: OrganizationStatus;
  plan: OrganizationPlan;
  settings: OrganizationSettings;
  quotas: ResourceQuota;
  billingInfo?: BillingInfo;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
}

/**
 * Organization status for lifecycle management
 */
export type OrganizationStatus = 'active' | 'suspended' | 'deleted' | 'trial';

/**
 * Organization plan determining feature access and limits
 */
export type OrganizationPlan = 'free' | 'pro' | 'enterprise';

/**
 * Organization-specific settings
 */
export interface OrganizationSettings {
  branding?: OrganizationBranding;
  ssoConfig?: SSOConfig;
  security: SecuritySettings;
  notificationSettings: NotificationSettings;
  [key: string]: unknown;
}

/**
 * Branding customization for organization
 */
export interface OrganizationBranding {
  logoUrl?: string;
  primaryColor?: string;
  customDomain?: string;
}

/**
 * SSO configuration for organization
 */
export interface SSOConfig {
  enabled: boolean;
  provider?: 'saml' | 'oidc';
  metadataUrl?: string;
  entityId?: string;
}

/**
 * Security settings for organization
 */
export interface SecuritySettings {
  requireMFA: boolean;
  passwordMinLength: number;
  sessionTimeoutMinutes: number;
  ipWhitelist?: string[];
}

/**
 * Notification settings for organization
 */
export interface NotificationSettings {
  emailAlerts: boolean;
  quotaAlerts: boolean;
  weeklyReports: boolean;
}

/**
 * Resource quota limits for organization
 */
export interface ResourceQuota {
  maxUsers: number;
  maxPrinters: number;
  maxStorageGB: number;
  maxJobsPerMonth: number;
  currentUserCount: number;
  currentPrinterCount: number;
  currentStorageGB: number;
  currentJobsThisMonth: number;
  quotaResetDate: string;
}

/**
 * Billing information for organization
 */
export interface BillingInfo {
  billingEmail?: string;
  billingAddress?: string;
  taxId?: string;
  paymentMethod?: string;
  subscriptionId?: string;
}

/**
 * Organization user membership
 */
export interface OrganizationUser {
  id: string;
  userId: string;
  organizationId: string;
  role: OrgRole;
  status: MemberStatus;
  invitedBy?: string;
  joinedAt: string;
  lastActiveAt?: string;
  user?: {
    id: string;
    name: string;
    email: string;
    isActive: boolean;
  };
}

/**
 * Role within organization
 */
export type OrgRole = 'owner' | 'admin' | 'member' | 'viewer';

/**
 * Membership status
 */
export type MemberStatus = 'active' | 'pending' | 'invited' | 'deactivated';

/**
 * Usage report data for organization
 */
export interface UsageReport {
  organizationId: string;
  period: UsagePeriod;
  startDate: string;
  endDate: string;
  totalJobs: number;
  totalPages: number;
  totalStorageUsed: number;
  userBreakdown: UserUsageBreakdown[];
  printerBreakdown: PrinterUsageBreakdown[];
  costEstimate: number;
  trends: UsageTrend[];
}

/**
 * Usage period type
 */
export type UsagePeriod = 'daily' | 'weekly' | 'monthly' | 'yearly';

/**
 * User usage breakdown
 */
export interface UserUsageBreakdown {
  userId: string;
  userName: string;
  userEmail: string;
  jobsCount: number;
  pagesCount: number;
  colorPages: number;
  storageUsed: number;
  cost: number;
}

/**
 * Printer usage breakdown
 */
export interface PrinterUsageBreakdown {
  printerId: string;
  printerName: string;
  jobsCount: number;
  pagesCount: number;
  colorPages: number;
  avgJobSize: number;
}

/**
 * Usage trend data point
 */
export interface UsageTrend {
  date: string;
  jobs: number;
  pages: number;
  users: number;
  storage: number;
}

/**
 * Platform admin view of all organizations
 */
export interface PlatformAdminOrganizationView extends Organization {
  ownerEmail?: string;
  ownerName?: string;
  healthScore: number;
  alertCount: number;
  usagePercentage: number;
  currentUserCount: number;
  currentPrinterCount: number;
}

/**
 * Create organization request
 */
export interface CreateOrganizationRequest {
  name: string;
  slug: string;
  displayName?: string;
  plan: OrganizationPlan;
  ownerId: string;
  quotas?: Partial<ResourceQuota>;
  settings?: Partial<OrganizationSettings>;
}

/**
 * Update organization request
 */
export interface UpdateOrganizationRequest {
  name?: string;
  displayName?: string;
  status?: OrganizationStatus;
  plan?: OrganizationPlan;
  settings?: Partial<OrganizationSettings>;
  billingInfo?: Partial<BillingInfo>;
}

/**
 * Update quota request
 */
export interface UpdateQuotaRequest {
  organizationId: string;
  maxUsers?: number;
  maxPrinters?: number;
  maxStorageGB?: number;
  maxJobsPerMonth?: number;
}

/**
 * Organization invitation
 */
export interface OrganizationInvitation {
  id: string;
  organizationId: string;
  organizationName: string;
  email: string;
  role: OrgRole;
  invitedBy: string;
  invitedByName?: string;
  expiresAt: string;
  createdAt: string;
  status: 'pending' | 'accepted' | 'expired' | 'cancelled';
}

/**
 * Paginated organizations response
 */
export interface PaginatedOrganizationsResponse {
  data: PlatformAdminOrganizationView[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * Organizations list filters
 */
export interface OrganizationsListFilters {
  status?: OrganizationStatus;
  plan?: OrganizationPlan;
  search?: string;
  sortBy?: 'name' | 'createdAt' | 'usage' | 'plan';
  sortOrder?: 'asc' | 'desc';
}

/**
 * Alert for organization
 */
export interface OrganizationAlert {
  id: string;
  organizationId: string;
  type: 'quota' | 'billing' | 'security' | 'system';
  severity: 'info' | 'warning' | 'error' | 'critical';
  title: string;
  message: string;
  createdAt: string;
  resolvedAt?: string;
}
