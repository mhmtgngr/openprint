import type { AgentStatus } from '@/types/agents';
import { AGENT_STATUS_CONFIG } from '@/types/agents';

interface AgentStatusBadgeProps {
  status: AgentStatus;
  className?: string;
  showLabel?: boolean;
}

export const AgentStatusBadge = ({
  status,
  className = '',
  showLabel = true,
}: AgentStatusBadgeProps) => {
  const config = AGENT_STATUS_CONFIG[status];

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${config.bgColor} ${config.textColor} ${className}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${config.dotColor}`} />
      {showLabel && config.label}
    </span>
  );
};
