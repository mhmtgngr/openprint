// User & Authentication
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

// Organization
export interface Organization {
  id: string;
  name: string;
  slug: string;
  plan: OrganizationPlan;
  settings: OrganizationSettings;
  maxUsers: number;
  maxPrinters: number;
  createdAt: string;
  // Multi-tenancy extensions
  displayName?: string;
  status?: OrganizationStatus;
  quotas?: ResourceQuota;
  currentUserCount?: number;
  currentPrinterCount?: number;
}

export interface OrganizationSettings {
  branding?: {
    logoUrl?: string;
    primaryColor?: string;
    customDomain?: string;
  };
  security?: {
    requireMFA?: boolean;
    passwordMinLength?: number;
    sessionTimeoutMinutes?: number;
  };
  [key: string]: unknown;
}

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

export type OrganizationStatus = 'active' | 'suspended' | 'deleted' | 'trial';

export type OrganizationPlan = 'free' | 'pro' | 'enterprise';

// Agents
export interface Agent {
  id: string;
  name: string;
  orgId: string;
  status: AgentStatus;
  platform: string;
  platformVersion?: string;
  agentVersion?: string;
  ipAddress?: string;
  lastHeartbeat?: string;
  capabilities: AgentCapabilities;
  createdAt: string;
}

export type AgentStatus = 'online' | 'offline' | 'error';

export interface AgentCapabilities {
  supportedFormats: string[];
  maxJobSize: number;
  supportsColor: boolean;
  supportsDuplex: boolean;
}

// Printers
export interface Printer {
  id: string;
  name: string;
  agentId: string;
  orgId: string;
  type: PrinterType;
  driver?: string;
  port?: string;
  capabilities: PrinterCapabilities;
  isActive: boolean;
  isOnline: boolean;
  lastSeen?: string;
  createdAt: string;
}

export type PrinterType = 'usb' | 'network' | 'virtual';

export interface PrinterCapabilities {
  supportsColor: boolean;
  supportsDuplex: boolean;
  supportedPaperSizes: string[];
  resolution: string;
  maxSheetCount?: number;
}

export interface PrinterPermission {
  id: string;
  printerId: string;
  userId: string;
  permissionType: PermissionType;
  grantedAt: string;
  grantedBy?: string;
}

export type PermissionType = 'print' | 'manage' | 'admin';

// Print Jobs
export interface PrintJob {
  id: string;
  userId: string;
  printerId?: string;
  orgId: string;
  status: JobStatus;
  documentName: string;
  documentType?: string;
  pageCount: number;
  colorPages?: number;
  fileSize: number;
  fileHash?: string;
  storageKey?: string;
  settings: JobSettings;
  errorMessage?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  autoDeleteAt?: string;
  printer?: Printer;
  progress?: number; // 0-100, only present for processing jobs
}

export type JobStatus = 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';

export interface JobSettings {
  color?: boolean;
  duplex?: boolean;
  paperSize?: string;
  copies?: number;
  quality?: string;
  orientation?: 'portrait' | 'landscape';
}

export interface JobHistoryEntry {
  id: string;
  jobId: string;
  status: JobStatus;
  message?: string;
  metadata: Record<string, unknown>;
  timestamp: string;
}

export interface CreateJobRequest {
  printerId: string;
  documentName: string;
  fileData: string; // base64
  settings?: Partial<JobSettings>;
}

// Analytics
export interface UsageStats {
  id: string;
  orgId: string;
  userId?: string;
  printerId?: string;
  statDate: string;
  pagesPrinted: number;
  colorPages: number;
  jobsCount: number;
  jobsCompleted: number;
  jobsFailed: number;
  totalBytes: number;
  estimatedCost: number;
  co2Grams: number;
  treesSaved: number;
}

export interface EnvironmentReport {
  pagesPrinted: number;
  co2Grams: number;
  treesSaved: number;
  period: string;
}

export interface UsageAnalyticsParams {
  startDate?: string;
  endDate?: string;
  groupBy?: 'day' | 'week' | 'month' | 'user' | 'printer';
}

// Audit Logs
export interface AuditLog {
  id: string;
  userId?: string;
  orgId?: string;
  action: string;
  resourceType?: string;
  resourceId?: string;
  details: Record<string, unknown>;
  ipAddress?: string;
  userAgent?: string;
  timestamp: string;
}

// Invitations
export interface Invitation {
  id: string;
  orgId: string;
  email: string;
  role: UserRole;
  invitedBy?: string;
  acceptedBy?: string;
  acceptedAt?: string;
  expiresAt: string;
  createdAt: string;
}

// API Keys
export interface APIKey {
  id: string;
  userId: string;
  orgId: string;
  name: string;
  keyPrefix: string;
  scopes: string[];
  isActive: boolean;
  expiresAt?: string;
  lastUsedAt?: string;
  createdAt: string;
}

// Webhooks
export interface Webhook {
  id: string;
  orgId: string;
  name: string;
  url: string;
  events: string[];
  isActive: boolean;
  lastTriggeredAt?: string;
  failureCount: number;
  createdAt: string;
}

// WebSocket Messages
export interface WebSocketMessage {
  type: WebSocketMessageType;
  data: unknown;
}

export type WebSocketMessageType =
  | 'job.status_update'
  | 'job.created'
  | 'job.completed'
  | 'job.failed'
  | 'printer.online'
  | 'printer.offline'
  | 'agent.connected'
  | 'agent.disconnected'
  | 'notification';

export interface JobStatusUpdateMessage {
  jobId: string;
  status: JobStatus;
  message?: string;
}

// API Response wrappers
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

// Form types
export interface InviteUserRequest {
  email: string;
  role: UserRole;
}

export interface UpdateOrganizationRequest {
  name?: string;
  displayName?: string;
  status?: OrganizationStatus;
  plan?: OrganizationPlan;
  settings?: Record<string, unknown>;
}

export interface UpdateUserRequest {
  name?: string;
  email?: string;
}

export interface CreateWebhookRequest {
  name: string;
  url: string;
  events: string[];
}

// Cost Tracking & Quota Management
export interface UserQuota {
  userId: string;
  orgId: string;
  monthlyPageLimit: number;
  monthlyColorPageLimit?: number;
  currentMonthPages: number;
  currentMonthColorPages: number;
  currentMonthCost: number;
  quotaResetDate: string;
  overageActions: OverageAction[];
}

export type OverageAction = 'block' | 'charge' | 'warn' | 'allow';

export interface QuotaPeriod {
  startDate: string;
  endDate: string;
  totalPages: number;
  totalCost: number;
  breakdowndByUser: UserCostBreakdown[];
}

export interface UserCostBreakdown {
  userId: string;
  userName: string;
  pages: number;
  colorPages: number;
  cost: number;
}

export interface UpdateQuotaRequest {
  userId?: string;
  monthlyPageLimit?: number;
  monthlyColorPageLimit?: number;
  overageActions?: OverageAction[];
}

// Print Policy Engine
export interface PrintPolicy {
  id: string;
  orgId: string;
  name: string;
  description?: string;
  isEnabled: boolean;
  priority: number;
  conditions: PolicyConditions;
  actions: PolicyActions;
  appliesTo: PolicyScope;
  createdAt: string;
  updatedAt: string;
}

export interface PolicyConditions {
  maxPagesPerJob?: number;
  maxPagesPerMonth?: number;
  allowedFileTypes?: string[];
  blockedFileTypes?: string[];
  requireApproval?: boolean;
  timeRestrictions?: TimeRestriction;
  userRestrictions?: string[]; // user IDs
  printerRestrictions?: string[]; // printer IDs
}

export interface TimeRestriction {
  allowedDays?: number[]; // 0-6 (Sunday-Saturday)
  allowedHours?: { start: string; end: string }[]; // "09:00" - "17:00"
}

export interface PolicyActions {
  forceDuplex?: boolean;
  forceColor?: boolean | 'grayscale';
  forceBlackAndWhite?: boolean;
  maxCopies?: number;
  defaultPaperSize?: string;
  requirePinRelease?: boolean;
  requireApproval?: boolean;
  defaultQuality?: 'draft' | 'normal' | 'high';
}

export type PolicyScope = 'all' | 'users' | 'groups' | 'printers';

export interface CreatePolicyRequest {
  name: string;
  description?: string;
  conditions: PolicyConditions;
  actions: PolicyActions;
  appliesTo: PolicyScope;
  targetIds?: string[]; // user/group/printer IDs based on scope
}

// Secure Print Release
export interface PrintRelease {
  id: string;
  jobId: string;
  userId: string;
  printerId: string;
  releaseCode: string; // hashed
  status: 'pending' | 'released' | 'expired' | 'cancelled';
  createdAt: string;
  expiresAt: string;
  releasedAt?: string;
}

export interface ReleaseJobRequest {
  jobId: string;
  pin: string;
  printerId: string;
}

export interface CreateSecureJobRequest extends CreateJobRequest {
  requirePin: boolean;
  pin?: string;
  expiresAt?: string;
}

// Email-to-Print
export interface EmailToPrintConfig {
  id: string;
  orgId: string;
  isEnabled: boolean;
  emailPrefix: string; // e.g., "print@" for print@org.openprint.cloud
  defaultPrinterId?: string;
  allowedSenders?: string[]; // email addresses or domains
  autoRelease?: boolean;
  requireApproval?: boolean;
  maxAttachments?: number;
  allowedFileTypes?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface EmailPrintJob {
  id: string;
  configId: string;
  fromEmail: string;
  subject: string;
  attachmentCount: number;
  status: 'received' | 'processing' | 'completed' | 'failed';
  createdAt: string;
  processedAt?: string;
  errorMessage?: string;
}

export interface UpdateEmailConfigRequest {
  isEnabled?: boolean;
  defaultPrinterId?: string;
  allowedSenders?: string[];
  autoRelease?: boolean;
  requireApproval?: boolean;
  maxAttachments?: number;
  allowedFileTypes?: string[];
}

// Re-export observability types
export * from './observability';

// Re-export organization types for multi-tenancy
export * from './organization';
