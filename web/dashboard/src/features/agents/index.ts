/**
 * Agents Feature Module
 * Exports all components, hooks, and types for agent management
 */

// Components
export { AgentList } from './AgentList';
export { AgentDetail } from './AgentDetail';
export { AgentPrinters } from './AgentPrinters';
export { AgentHealthChart } from './AgentHealthChart';

// Hooks
export {
  useAgents,
  useAgent,
  useAgentDetail,
  useAgentHealth,
  useAgentPrinters,
  useAgentJobHistory,
  useUpdateAgent,
  useDeleteAgent,
  useTriggerDiscovery,
  useRestartAgent,
  useDiscoveredPrinters,
  useDiscoveredPrinter,
  useUpdateDiscoveredPrinter,
  useDeleteDiscoveredPrinter,
  useSetDefaultPrinter,
  useJobAssignments,
  useCreateJobAssignment,
  useUpdateJobAssignment,
  useCancelJobAssignment,
  useReassignJob,
} from './useAgents';

// Types
export type {
  Agent,
  AgentDetail as AgentDetailType,
  AgentStatus,
  AgentCapabilities,
  AgentSessionState,
  DiscoveredPrinter,
  DiscoveredPrinterCapabilities,
  AgentJobHistoryEntry,
  AgentHealthMetrics,
  WeeklyJobCount,
  AgentListParams,
  JobAssignmentRequest,
  JobAssignment,
  AgentFormData,
  AgentFormErrors,
  AgentStatusConfig,
  AGENT_STATUS_CONFIG,
} from '@/types/agents';
