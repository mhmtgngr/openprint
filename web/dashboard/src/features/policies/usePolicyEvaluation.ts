import { useMutation, useQuery, useQueryClient, type UseMutationResult } from '@tanstack/react-query';
import { policyApi } from './api';
import type { PolicyEvaluationContext, PolicyEvaluationResult, PolicyTestJob } from './types';

/**
 * Hook for evaluating a policy
 */
export function usePolicyEvaluation() {
  return useMutation({
    mutationFn: ({ policyId, context }: { policyId: string; context: PolicyEvaluationContext }) =>
      policyApi.evaluate(policyId, context),
  });
}

/**
 * Hook for testing a job against all policies
 */
export function useJobTest(): UseMutationResult<
  { results: Array<{ policyId: string; policyName: string; result: PolicyEvaluationResult }> },
  Error,
  PolicyTestJob,
  unknown
> {
  return useMutation({
    mutationFn: (job: PolicyTestJob) => policyApi.testJob(job),
  });
}

/**
 * Hook for getting policy history
 */
export function usePolicyHistory(policyId: string, enabled = true) {
  return useQuery({
    queryKey: ['policy-history', policyId],
    queryFn: () => policyApi.getHistory(policyId),
    enabled: enabled && !!policyId,
  });
}

/**
 * Hook for restoring policy from history
 */
export function useRestorePolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ policyId, versionId }: { policyId: string; versionId: string }) =>
      policyApi.restoreFromHistory(policyId, versionId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      queryClient.invalidateQueries({ queryKey: ['policies', variables.policyId] });
      queryClient.invalidateQueries({ queryKey: ['policy-history', variables.policyId] });
    },
  });
}
