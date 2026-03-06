/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_API_BASE_URL: string;
  readonly VITE_WS_URL: string;
  readonly VITE_AUTH_URL: string;
  readonly VITE_STORAGE_URL: string;
  readonly VITE_STORAGE_API_URL: string;
  readonly VITE_JOB_URL: string;
  readonly VITE_REGISTRY_URL: string;
  readonly VITE_NOTIFICATION_URL: string;
  readonly VITE_PROMETHEUS_URL: string;
  readonly VITE_GRAFANA_URL: string;
  readonly VITE_JAEGER_URL: string;
  readonly VITE_MONITORING_API_URL: string;
  readonly VITE_APP_NAME: string;
  readonly VITE_APP_VERSION: string;
  readonly VITE_ENABLE_ANALYTICS: string;
  readonly VITE_ENABLE_AUDIT_LOGS: string;
  readonly VITE_ENABLE_EMAIL_TO_PRINT: string;
  readonly VITE_ENABLE_QUOTAS: string;
  readonly VITE_ENABLE_POLICIES: string;
  readonly VITE_SESSION_TIMEOUT_MINUTES: string;
  readonly VITE_MAX_UPLOAD_SIZE_MB: string;
  readonly VITE_DEFAULT_PAGE_SIZE: string;
  readonly VITE_MAX_PAGE_SIZE: string;
  readonly VITE_WS_RECONNECT_DELAY_MS: string;
  readonly VITE_WS_MAX_RECONNECT_ATTEMPTS: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
