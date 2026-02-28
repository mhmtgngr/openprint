/**
 * React hooks for agent management operations
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  agentApi,
  discoveredPrintersApi,
  jobAssignmentsApi,
} from '@/api/agentApi';
import type {
  Agent,
  AgentDetail,
  AgentListParams,
  DiscoveredPrinter,
  JobAssignmentRequest,
} from '@/types/agents';

/**
 * Hook for fetching a list of agents with optional filtering
 */
export const useAgents = (params?: AgentListParams) => {
  return useQuery({
    queryKey: ['agents', params],
    queryFn: () => agentApi.list(params),
    staleTime: 10000, // Consider data fresh for 10 seconds
  });
};

/**
 * Hook for fetching a single agent by ID
 */
export const useAgent = (id: string) => {
  return useQuery({
    queryKey: ['agent', id],
    queryFn: () => agentApi.get(id),
    enabled: !!id,
    staleTime: 5000,
  });
};

/**
 * Hook for fetching detailed agent information including printers and job history
 */
export const useAgentDetail = (id: string) => {
  return useQuery({
    queryKey: ['agent-detail', id],
    queryFn: () => agentApi.getDetail(id),
    enabled: !!id,
    staleTime: 5000,
    refetchInterval: (query) => {
      const detail = query.state.data as AgentDetail | undefined;
      // Poll every 5 seconds for agents that are online
      if (detail && detail.status === 'online') {
        return 5000;
      }
      return false;
    },
  });
};

/**
 * Hook for fetching agent health metrics
 */
export const useAgentHealth = (id: string) => {
  return useQuery({
    queryKey: ['agent-health', id],
    queryFn: () => agentApi.getHealth(id),
    enabled: !!id,
    staleTime: 30000, // Health data changes less frequently
  });
};

/**
 * Hook for fetching printers discovered by an agent
 */
export const useAgentPrinters = (agentId: string) => {
  return useQuery({
    queryKey: ['agent-printers', agentId],
    queryFn: () => agentApi.getPrinters(agentId),
    enabled: !!agentId,
    staleTime: 15000,
  });
};

/**
 * Hook for fetching agent job history
 */
export const useAgentJobHistory = (
  agentId: string,
  params?: { limit?: number; offset?: number }
) => {
  return useQuery({
    queryKey: ['agent-jobs', agentId, params],
    queryFn: () => agentApi.getJobHistory(agentId, params),
    enabled: !!agentId,
    staleTime: 30000,
  });
};

/**
 * Hook for updating agent information
 */
export const useUpdateAgent = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Agent> }) =>
      agentApi.update(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['agent', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['agent-detail', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['agents'] });
    },
  });
};

/**
 * Hook for deleting an agent
 */
export const useDeleteAgent = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => agentApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents'] });
    },
  });
};

/**
 * Hook for triggering printer discovery on an agent
 */
export const useTriggerDiscovery = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (agentId: string) => agentApi.triggerDiscovery(agentId),
    onSuccess: (_, agentId) => {
      queryClient.invalidateQueries({ queryKey: ['agent-printers', agentId] });
      queryClient.invalidateQueries({ queryKey: ['discovered-printers'] });
    },
  });
};

/**
 * Hook for restarting an agent
 */
export const useRestartAgent = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (agentId: string) => agentApi.restart(agentId),
    onSuccess: (_, agentId) => {
      queryClient.invalidateQueries({ queryKey: ['agent', agentId] });
      queryClient.invalidateQueries({ queryKey: ['agent-detail', agentId] });
      queryClient.invalidateQueries({ queryKey: ['agents'] });
    },
  });
};

/**
 * Hook for fetching all discovered printers
 */
export const useDiscoveredPrinters = (params?: {
  agentId?: string;
  status?: string;
  search?: string;
  limit?: number;
  offset?: number;
}) => {
  return useQuery({
    queryKey: ['discovered-printers', params],
    queryFn: () => discoveredPrintersApi.list(params),
    staleTime: 15000,
  });
};

/**
 * Hook for fetching a single discovered printer
 */
export const useDiscoveredPrinter = (id: string) => {
  return useQuery({
    queryKey: ['discovered-printer', id],
    queryFn: () => discoveredPrintersApi.get(id),
    enabled: !!id,
  });
};

/**
 * Hook for updating discovered printer
 */
export const useUpdateDiscoveredPrinter = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<DiscoveredPrinter> }) =>
      discoveredPrintersApi.update(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['discovered-printer', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['discovered-printers'] });
      queryClient.invalidateQueries({ queryKey: ['agent-printers'] });
    },
  });
};

/**
 * Hook for deleting discovered printer
 */
export const useDeleteDiscoveredPrinter = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => discoveredPrintersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['discovered-printers'] });
      queryClient.invalidateQueries({ queryKey: ['agent-printers'] });
    },
  });
};

/**
 * Hook for setting default printer
 */
export const useSetDefaultPrinter = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (printerId: string) => discoveredPrintersApi.setDefault(printerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['discovered-printers'] });
      queryClient.invalidateQueries({ queryKey: ['agent-printers'] });
    },
  });
};

/**
 * Hook for fetching job assignments
 */
export const useJobAssignments = (params?: {
  jobId?: string;
  status?: string;
  agentId?: string;
  userId?: string;
  limit?: number;
  offset?: number;
}) => {
  return useQuery({
    queryKey: ['job-assignments', params],
    queryFn: () => jobAssignmentsApi.list(params),
    staleTime: 10000,
  });
};

/**
 * Hook for creating a job assignment
 */
export const useCreateJobAssignment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: JobAssignmentRequest) => jobAssignmentsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['job-assignments'] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });
};

/**
 * Hook for updating a job assignment
 */
export const useUpdateJobAssignment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<JobAssignmentRequest> }) =>
      jobAssignmentsApi.update(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['job-assignment', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['job-assignments'] });
    },
  });
};

/**
 * Hook for cancelling a job assignment
 */
export const useCancelJobAssignment = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => jobAssignmentsApi.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['job-assignments'] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });
};

/**
 * Hook for reassigning a job
 */
export const useReassignJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: { agentId?: string; userId?: string; printerId?: string };
    }) => jobAssignmentsApi.reassign(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['job-assignment', variables.id] });
      queryClient.invalidateQueries({ queryKey: ['job-assignments'] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });
};
