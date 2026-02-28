/**
 * React hooks for job management operations
 */

import React from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { jobsApi } from '@/services/api';
import type { CreateJobRequest, JobListParams, PrintJob } from '@/types/jobs';

/**
 * Hook for fetching a paginated list of print jobs
 * @param params - Query parameters for filtering jobs
 */
export const useJobs = (params?: JobListParams) => {
  return useQuery({
    queryKey: ['jobs', params],
    queryFn: () => {
      const filteredParams = params?.status === 'all'
        ? { ...params, status: undefined as never }
        : params;
      return jobsApi.list(filteredParams);
    },
    staleTime: 5000, // Consider data fresh for 5 seconds
  });
};

/**
 * Hook for fetching a single job by ID
 * Automatically polls for jobs in queued or processing status
 * @param id - Job ID
 */
export const useJob = (id: string) => {
  return useQuery({
    queryKey: ['job', id],
    queryFn: () => jobsApi.get(id),
    enabled: !!id,
    refetchInterval: (query) => {
      const job = query.state.data as PrintJob | undefined;
      // Poll every 2 seconds for jobs that are queued or processing
      if (job && (job.status === 'queued' || job.status === 'processing')) {
        return 2000;
      }
      return false; // Stop polling for completed/failed/cancelled jobs
    },
  });
};

/**
 * Hook for fetching job history entries
 * @param jobId - Job ID
 */
export const useJobHistory = (jobId: string) => {
  return useQuery({
    queryKey: ['job-history', jobId],
    queryFn: () => jobsApi.getHistory(jobId),
    enabled: !!jobId,
  });
};

/**
 * Hook for creating a new print job
 * Includes invalidation of jobs list on success
 */
export const useCreateJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateJobRequest) => jobsApi.create(data),
    onSuccess: () => {
      // Invalidate jobs list to refresh
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
    onError: (error) => {
      console.error('Failed to create job:', error);
    },
  });
};

/**
 * Hook for cancelling a print job
 * @returns Mutation function for cancelling a job
 */
export const useCancelJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (jobId: string) => jobsApi.cancel(jobId),
    onSuccess: (_, jobId) => {
      // Invalidate both the specific job and the jobs list
      queryClient.invalidateQueries({ queryKey: ['job', jobId] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
    onError: (error) => {
      console.error('Failed to cancel job:', error);
    },
  });
};

/**
 * Hook for retrying a failed print job
 * @returns Mutation function for retrying a job
 */
export const useRetryJob = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (jobId: string) => jobsApi.retry(jobId),
    onSuccess: (_, jobId) => {
      queryClient.invalidateQueries({ queryKey: ['job', jobId] });
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
    onError: (error) => {
      console.error('Failed to retry job:', error);
    },
  });
};

/**
 * Hook for bulk job operations
 */
export const useBulkJobActions = () => {
  const queryClient = useQueryClient();
  const cancelJob = useCancelJob();

  const bulkCancel = useMutation({
    mutationFn: async (jobIds: string[]) => {
      return Promise.all(jobIds.map((id) => jobsApi.cancel(id)));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
    },
  });

  return {
    bulkCancel,
    cancelJob: cancelJob.mutate,
  };
};

/**
 * Custom hook for file upload progress tracking
 */
export const useFileUpload = () => {
  const [progress, setProgress] = React.useState(0);
  const [isUploading, setIsUploading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const uploadFile = async (
    file: File,
    onProgress?: (progress: number) => void
  ): Promise<string> => {
    setIsUploading(true);
    setProgress(0);
    setError(null);

    return new Promise((resolve, reject) => {
      const reader = new FileReader();

      reader.onprogress = (event) => {
        if (event.lengthComputable) {
          const percentComplete = Math.round((event.loaded / event.total) * 100);
          setProgress(percentComplete);
          onProgress?.(percentComplete);
        }
      };

      reader.onload = () => {
        const base64 = (reader.result as string).split(',')[1];
        setIsUploading(false);
        resolve(base64);
      };

      reader.onerror = () => {
        setIsUploading(false);
        const errorMsg = 'Failed to read file';
        setError(errorMsg);
        reject(new Error(errorMsg));
      };

      reader.readAsDataURL(file);
    });
  };

  return {
    uploadFile,
    progress,
    isUploading,
    error,
    reset: () => {
      setProgress(0);
      setIsUploading(false);
      setError(null);
    },
  };
};
