/**
 * ComplianceStatusBadge Component
 * Displays compliance status with appropriate styling and icon
 */

import type { ComplianceStatus } from '../types';

export interface ComplianceStatusBadgeProps {
  status: ComplianceStatus;
  showLabel?: boolean;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const statusConfig = {
  compliant: {
    bg: 'bg-green-100 dark:bg-green-900/30',
    border: 'border-green-200 dark:border-green-800',
    text: 'text-green-700 dark:text-green-400',
    iconBg: 'bg-green-200 dark:bg-green-800',
    label: 'Compliant',
  },
  non_compliant: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    border: 'border-red-200 dark:border-red-800',
    text: 'text-red-700 dark:text-red-400',
    iconBg: 'bg-red-200 dark:bg-red-800',
    label: 'Non-Compliant',
  },
  in_progress: {
    bg: 'bg-amber-100 dark:bg-amber-900/30',
    border: 'border-amber-200 dark:border-amber-800',
    text: 'text-amber-700 dark:text-amber-400',
    iconBg: 'bg-amber-200 dark:bg-amber-800',
    label: 'In Progress',
  },
  pending: {
    bg: 'bg-gray-100 dark:bg-gray-700',
    border: 'border-gray-200 dark:border-gray-600',
    text: 'text-gray-700 dark:text-gray-400',
    iconBg: 'bg-gray-200 dark:bg-gray-600',
    label: 'Pending',
  },
  not_applicable: {
    bg: 'bg-slate-100 dark:bg-slate-700',
    border: 'border-slate-200 dark:border-slate-600',
    text: 'text-slate-600 dark:text-slate-400',
    iconBg: 'bg-slate-200 dark:bg-slate-600',
    label: 'N/A',
  },
  unknown: {
    bg: 'bg-gray-100 dark:bg-gray-700',
    border: 'border-gray-200 dark:border-gray-600',
    text: 'text-gray-600 dark:text-gray-400',
    iconBg: 'bg-gray-200 dark:bg-gray-600',
    label: 'Unknown',
  },
};

const sizeConfig = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-3 py-1 text-sm',
  lg: 'px-4 py-1.5 text-base',
};

const iconSizeConfig = {
  sm: 'w-3 h-3',
  md: 'w-4 h-4',
  lg: 'w-5 h-5',
};

export const ComplianceStatusBadge = ({
  status,
  showLabel = true,
  size = 'md',
  className = '',
}: ComplianceStatusBadgeProps) => {
  const config = statusConfig[status];
  const sizeClass = sizeConfig[size];
  const iconSize = iconSizeConfig[size];

  return (
    <span
      className={`
        inline-flex items-center gap-1.5 rounded-full border font-medium
        ${config.bg} ${config.border} ${config.text} ${sizeClass}
        ${className}
      `}
      data-testid={`compliance-status-${status}`}
    >
      <StatusIcon status={status} className={iconSize} />
      {showLabel && <span>{config.label}</span>}
    </span>
  );
};

const StatusIcon = ({ status, className }: { status: ComplianceStatus; className?: string }) => {
  switch (status) {
    case 'compliant':
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
      );
    case 'non_compliant':
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
            clipRule="evenodd"
          />
        </svg>
      );
    case 'in_progress':
      return (
        <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
      );
    case 'pending':
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
            clipRule="evenodd"
          />
        </svg>
      );
    default:
      return (
        <svg className={className} fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
            clipRule="evenodd"
          />
        </svg>
      );
  }
};

export default ComplianceStatusBadge;
