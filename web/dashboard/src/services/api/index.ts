// Re-export all domain API modules from a single entry point.
// Consumers can import from '@/services/api' or from individual modules.
export { authApi } from './auth';
export { printersApi } from './printers';
export { jobsApi } from './jobs';
export { agentsApi } from './agents';
export { organizationApi } from './organization';
export { analyticsApi } from './analytics';
export { webhooksApi } from './webhooks';
export { userApi } from './users';
export { quotasApi } from './quotas';
export { policiesApi } from './policies';
export { printReleaseApi } from './print-release';
export { emailToPrintApi } from './email-to-print';

// Re-export HTTP client utilities for direct consumers
export {
  httpClient,
  getAccessToken,
  setTokens,
  clearTokens,
  onAuthFailure,
  APIErrorClass,
} from '@/services/http';
export type { APIErrorClass as APIError } from '@/services/http';
