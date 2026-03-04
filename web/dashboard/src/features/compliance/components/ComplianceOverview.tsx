/**
 * ComplianceOverview Component
 * Displays compliance overview statistics and framework statuses
 */

import type { ComplianceOverview as OverviewData, ComplianceFramework } from '../types';
import { ComplianceStatusBadge } from './ComplianceStatusBadge';
import { StatCard } from './StatCard';

export interface ComplianceOverviewProps {
  data?: OverviewData;
  isLoading?: boolean;
  error?: string | null;
}

const frameworkConfig: Record<
  ComplianceFramework,
  { name: string; color: 'blue' | 'green' | 'purple' | 'orange'; description: string }
> = {
  fedramp: {
    name: 'FedRAMP',
    color: 'blue',
    description: 'Federal Risk and Authorization Management Program',
  },
  hipaa: {
    name: 'HIPAA',
    color: 'green',
    description: 'Health Insurance Portability and Accountability Act',
  },
  gdpr: {
    name: 'GDPR',
    color: 'purple',
    description: 'General Data Protection Regulation',
  },
  soc2: {
    name: 'SOC 2',
    color: 'orange',
    description: 'Service Organization Control 2',
  },
  all: {
    name: 'All',
    color: 'blue',
    description: 'All compliance frameworks',
  },
};

export const ComplianceOverviewComponent = ({
  data,
  isLoading = false,
  error = null,
}: ComplianceOverviewProps) => {
  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-6" data-testid="compliance-overview-loading">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div
              key={i}
              className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 animate-pulse"
            >
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mb-2" />
              <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-2" />
              <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-2/3" />
            </div>
          ))}
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => (
            <div
              key={i}
              className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 animate-pulse"
            >
              <div className="h-12 bg-gray-200 dark:bg-gray-700 rounded w-12 mb-4" />
              <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mb-2" />
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div
        className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6"
        data-testid="compliance-overview-error"
      >
        <div className="flex items-start gap-3">
          <svg
            className="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <div>
            <h3 className="text-sm font-medium text-red-800 dark:text-red-400">
              Failed to load compliance overview
            </h3>
            <p className="text-sm text-red-700 dark:text-red-300 mt-1">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  // Default data for development
  const overview: OverviewData = data || {
    fedramp: { status: 'compliant', last_audit: '2024-01-15' },
    hipaa: { status: 'compliant', last_audit: '2024-01-15' },
    gdpr: { status: 'compliant', last_audit: '2024-01-15' },
    soc2: { status: 'in_progress', last_audit: '2024-01-15' },
    total_logs: 1523,
    compliant_standards: 3,
    pending_actions: 5,
  };

  return (
    <div className="space-y-6" data-testid="compliance-overview">
      {/* Framework Status Cards */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          Compliance Frameworks
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {(Object.keys(frameworkConfig) as Array<ComplianceFramework>)
            .filter((key) => key !== 'all')
            .map((framework) => {
              const config = frameworkConfig[framework];
              const status = overview[framework];
              return (
                <FrameworkCard
                  key={framework}
                  framework={framework}
                  name={config.name}
                  description={config.description}
                  status={status.status}
                  lastAudit={status.last_audit}
                />
              );
            })}
        </div>
      </div>

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard
          title="Total Audit Logs"
          value={overview.total_logs}
          icon={DocumentIcon}
          color="blue"
        />
        <StatCard
          title="Compliant Standards"
          value={overview.compliant_standards}
          icon={ShieldIcon}
          color="green"
        />
        <StatCard
          title="Pending Actions"
          value={overview.pending_actions}
          icon={AlertIcon}
          color="orange"
        />
      </div>
    </div>
  );
};

interface FrameworkCardProps {
  framework: ComplianceFramework;
  name: string;
  description: string;
  status: ComplianceOverviewData['status'];
  lastAudit: string;
}

type ComplianceOverviewData = {
  status: import('../types').ComplianceStatus;
  last_audit: string;
};

const FrameworkCard = ({
  name,
  description,
  status,
  lastAudit,
}: FrameworkCardProps) => (
  <div
    className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg border border-gray-200 dark:border-gray-700 hover:shadow-sm transition-shadow"
    data-testid={`framework-card-${name.toLowerCase()}`}
  >
    <div className="flex items-center justify-between mb-2">
      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{name}</span>
      <div data-testid={`${name.toLowerCase()}-status`}>
        <ComplianceStatusBadge status={status} size="sm" />
      </div>
    </div>
    <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{description}</p>
    <p className="text-xs text-gray-500 dark:text-gray-400">
      Last audit: {new Date(lastAudit).toLocaleDateString()}
    </p>
  </div>
);

// Icons
const DocumentIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
    />
  </svg>
);

const ShieldIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
    />
  </svg>
);

const AlertIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
    />
  </svg>
);

export default ComplianceOverviewComponent;
