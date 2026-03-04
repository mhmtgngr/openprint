import { useQuery, useMutation, useQueryClient, type UseQueryResult } from '@tanstack/react-query';
import { policyApi } from './api';
import type { PrintPolicy, CreatePolicyRequest } from './types';

/**
 * Hook for fetching policies list
 */
export function usePolicies() {
  return useQuery({
    queryKey: ['policies'],
    queryFn: () => policyApi.list(),
  });
}

/**
 * Hook for fetching a single policy
 */
export function usePolicy(id: string, enabled = true): UseQueryResult<PrintPolicy, Error> {
  return useQuery({
    queryKey: ['policies', id],
    queryFn: () => policyApi.get(id),
    enabled: enabled && !!id,
  });
}

/**
 * Hook for creating a policy
 */
export function useCreatePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreatePolicyRequest) => policyApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });
}

/**
 * Hook for updating a policy
 */
export function useUpdatePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreatePolicyRequest> }) =>
      policyApi.update(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      queryClient.invalidateQueries({ queryKey: ['policies', variables.id] });
    },
  });
}

/**
 * Hook for deleting a policy
 */
export function useDeletePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => policyApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });
}

/**
 * Hook for toggling policy enabled state
 */
export function useTogglePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, isEnabled }: { id: string; isEnabled: boolean }) =>
      policyApi.toggle(id, isEnabled),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      queryClient.invalidateQueries({ queryKey: ['policies', variables.id] });
    },
  });
}

/**
 * Hook for reordering policies
 */
export function useReorderPolicies() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (policyIds: string[]) => policyApi.reorder(policyIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });
}

/**
 * Hook for duplicating a policy
 */
export function useDuplicatePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, name }: { id: string; name?: string }) => policyApi.duplicate(id, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });
}

/**
 * Hook for importing/exporting policies
 */
export function usePolicyImportExport() {
  const queryClient = useQueryClient();

  const importPolicy = useMutation({
    mutationFn: (file: File) => policyApi.import(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });

  const exportPolicy = async (id: string, filename?: string) => {
    const blob = await policyApi.export(id);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename || `policy-${id}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return {
    importPolicy,
    exportPolicy,
    isImporting: importPolicy.isPending,
  };
}
