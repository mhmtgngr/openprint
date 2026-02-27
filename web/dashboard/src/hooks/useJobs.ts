import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { jobsApi } from '@/services/api';
import { useJobUpdates } from './useWebSocket';
import type { CreateJobRequest, JobStatus } from '@/types';

export const useJobs = (params?: { status?: string; limit?: number; offset?: number }) => {
  return useQuery({
    queryKey: ['jobs', params],
    queryFn: () => jobsApi.list(params),
  });
};

export const useJob = (id: string) => {
  return useQuery({
    queryKey: ['job', id],
    queryFn: () => jobsApi.get(id),
    enabled: !!id,
    refetchInterval: (query) => {
      const job = query.state.data;
      // Poll for jobs that are queued or processing
      if (job && (job.status === 'queued' || job.status === 'processing')) {
        return 2000; // Poll every 2 seconds
      }
      return false; // Stop polling for completed/failed jobs
    },
  });
};

export const useJobHistory = (jobId: string) => {
  return useQuery({
    queryKey: ['job-history', jobId],
    queryFn: () => jobsApi.getHistory(jobId),
    enabled: !!jobId,
  });
};

export const useCreateJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateJobRequest) => jobsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });
};

export const useCancelJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (jobId: string) => jobsApi.cancel(jobId),
    onSuccess: (_, jobId) => {
      queryClient.invalidateQueries({ queryKey: ['job', jobId] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });
};

export const useRetryJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (jobId: string) => jobsApi.retry(jobId),
    onSuccess: (_, jobId) => {
      queryClient.invalidateQueries({ queryKey: ['job', jobId] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });
};

// Hook for real-time job updates using WebSocket
export const useRealTimeJobs = () => {
  const { status, isConnected } = useWebSocket();

  return {
    connectionStatus: status,
    isRealTimeEnabled: isConnected,
  };
};

// Hook for a single job with real-time updates
export const useJobRealtime = (jobId: string) => {
  const jobQuery = useJob(jobId);
  const { status: wsStatus } = useJobUpdates(jobId);

  return {
    ...jobQuery,
    realtimeStatus: wsStatus,
    isUpdating: wsStatus === 'processing' || wsStatus === 'queued',
  };
};
