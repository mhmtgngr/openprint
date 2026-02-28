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
}

export type UserRole = 'user' | 'admin' | 'owner';

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
  settings: Record<string, unknown>;
  maxUsers: number;
  maxPrinters: number;
  createdAt: string;
}

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
