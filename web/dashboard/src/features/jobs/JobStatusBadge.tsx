/**
 * JobStatusBadge Component - Displays a visual badge for job status
 */

import type { JobStatus } from '@/types/jobs';
import { JOB_STATUS_CONFIG } from '@/types/jobs';

interface JobStatusBadgeProps {
  status: JobStatus;
  className?: string;
  showIcon?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

const sizeClasses = {
  sm: 'px-2 py-0.5 text-xs',
  md: 'px-2.5 py-1 text-xs',
  lg: 'px-3 py-1.5 text-sm',
};

const dotSizes = {
  sm: 'w-1 h-1',
  md: 'w-1.5 h-1.5',
  lg: 'w-2 h-2',
};

export const JobStatusBadge = ({
  status,
  className = '',
  showIcon = false,
  size = 'md',
}: JobStatusBadgeProps) => {
  const config = JOB_STATUS_CONFIG[status];
  const sizeClass = sizeClasses[size];
  const dotSize = dotSizes[size];

  return (
    <span
      className={`
        inline-flex items-center gap-1.5 rounded-full font-medium
        ${config.bgColor} ${config.textColor} ${sizeClass} ${className}
      `}
    >
      <span className={`rounded-full ${config.dotColor} ${dotSize}`} />
      {config.label}
      {showIcon && status === 'processing' && (
        <svg
          className="w-3 h-3 animate-spin"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
      )}
    </span>
  );
};

export default JobStatusBadge;
