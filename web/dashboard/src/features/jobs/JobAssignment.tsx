/**
 * JobAssignment Component
 * Admin interface to assign print jobs to specific users/agents
 */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useJobAssignments, useCreateJobAssignment, useReassignJob, useCancelJobAssignment } from '../agents/useAgents';
import { useAgents } from '../agents/useAgents';
import { jobsApi } from '@/services/api';
import type { JobAssignment as JobAssignmentType } from '@/types/agents';
import type { PrintJob } from '@/types';
import { AgentStatusBadge } from '@/components/AgentStatusBadge';

interface JobAssignmentProps {
  jobId?: string; // If provided, show assignment for a specific job
}

export const JobAssignment = ({ jobId }: JobAssignmentProps) => {
  const [selectedJobId, setSelectedJobId] = useState<string | undefined>(jobId);
  const [selectedAgentId, setSelectedAgentId] = useState<string>('');
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [showAssignModal, setShowAssignModal] = useState(false);

  // Fetch data
  const { data: assignments, isLoading: assignmentsLoading } = useJobAssignments(
    selectedJobId ? { jobId: selectedJobId } : undefined
  );
  const { data: agents } = useAgents();
  const { data: jobsData } = useQuery({
    queryKey: ['jobs'],
    queryFn: () => jobsApi.list({ status: 'queued', limit: 50 }),
  });

  // Mutations
  const createAssignmentMutation = useCreateJobAssignment();
  const reassignMutation = useReassignJob();
  const cancelMutation = useCancelJobAssignment();

  const jobs = jobsData?.data || [];

  const handleCreateAssignment = async () => {
    if (!selectedJobId) return;

    await createAssignmentMutation.mutateAsync({
      jobId: selectedJobId,
      agentId: selectedAgentId || undefined,
      userId: selectedUserId || undefined,
    });

    setShowAssignModal(false);
    setSelectedAgentId('');
    setSelectedUserId('');
  };

  const handleReassign = async (assignmentId: string, agentId?: string, userId?: string) => {
    await reassignMutation.mutateAsync({
      id: assignmentId,
      data: { agentId, userId },
    });
  };

  const handleCancel = async (assignmentId: string) => {
    if (confirm('Are you sure you want to cancel this assignment?')) {
      await cancelMutation.mutateAsync(assignmentId);
    }
  };

  const getAssignmentStatusColor = (status: string) => {
    switch (status) {
      case 'pending':
        return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300';
      case 'assigned':
        return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300';
      case 'in_progress':
        return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300';
      case 'completed':
        return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300';
      case 'failed':
        return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300';
      default:
        return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300';
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Job Assignments
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Assign print jobs to specific agents or users
          </p>
        </div>
        <button
          onClick={() => setShowAssignModal(true)}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
        >
          New Assignment
        </button>
      </div>

      {/* Job Filter */}
      {jobs.length > 0 && !jobId && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Select Job to Assign
          </label>
          <select
            value={selectedJobId || ''}
            onChange={(e) => setSelectedJobId(e.target.value || undefined)}
            className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="">All Queued Jobs</option>
            {jobs.map((job: PrintJob) => (
              <option key={job.id} value={job.id}>
                {job.documentName} ({job.pageCount} pages)
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Assignments List */}
      {assignmentsLoading ? (
        <div className="space-y-3">
          {[...Array(3)].map((_, i) => (
            <div
              key={i}
              className="bg-gray-100 dark:bg-gray-800 rounded-lg h-28 animate-pulse"
            />
          ))}
        </div>
      ) : !assignments || assignments.data.length === 0 ? (
        <div className="text-center py-12">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            No assignments found
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Create a new assignment to route print jobs to specific agents.
          </p>
        </div>
      ) : (
        <div className="overflow-hidden">
          <ul className="divide-y divide-gray-200 dark:divide-gray-700">
            {assignments?.data.map((assignment: JobAssignmentType) => (
              <li
                key={assignment.id}
                className="py-4 px-4 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors"
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        {assignment.job?.documentName || `Job ${assignment.jobId}`}
                      </p>
                      <span
                        className={`px-2 py-1 text-xs font-medium rounded-full ${getAssignmentStatusColor(
                          assignment.status
                        )}`}
                      >
                        {assignment.status.replace('_', ' ')}
                      </span>
                      {assignment.priority > 0 && (
                        <span className="px-2 py-1 text-xs font-medium rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                          Priority: {assignment.priority}
                        </span>
                      )}
                    </div>

                    <div className="mt-2 flex flex-wrap items-center gap-4 text-sm text-gray-600 dark:text-gray-400">
                      {assignment.agent && (
                        <span className="flex items-center gap-1">
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                            />
                          </svg>
                          {assignment.agent.name}
                          <AgentStatusBadge status={assignment.agent.status} showLabel={false} />
                        </span>
                      )}
                      {assignment.user && (
                        <span className="flex items-center gap-1">
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                            />
                          </svg>
                          {assignment.user.name}
                        </span>
                      )}
                      {assignment.assignedAt && (
                        <span>
                          Assigned{' '}
                          {new Date(assignment.assignedAt).toLocaleDateString()} at{' '}
                          {new Date(assignment.assignedAt).toLocaleTimeString()}
                        </span>
                      )}
                      {assignment.errorMessage && (
                        <span className="text-red-600 dark:text-red-400">
                          {assignment.errorMessage}
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="ml-4 flex items-center gap-2">
                    {assignment.status === 'pending' && assignment.agent && (
                      <button
                        onClick={() =>
                          handleReassign(assignment.id, assignment.agent?.id, assignment.user?.id)
                        }
                        className="p-2 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                        title="Reassign"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
                          />
                        </svg>
                      </button>
                    )}
                    {(assignment.status === 'pending' || assignment.status === 'assigned') && (
                      <button
                        onClick={() => handleCancel(assignment.id)}
                        className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                        title="Cancel assignment"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M6 18L18 6M6 6l12 12"
                          />
                        </svg>
                      </button>
                    )}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* New Assignment Modal */}
      {showAssignModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                Create New Assignment
              </h3>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Select Job
                </label>
                <select
                  value={selectedJobId || ''}
                  onChange={(e) => setSelectedJobId(e.target.value || undefined)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  required
                >
                  <option value="">Choose a job...</option>
                  {jobs.map((job: PrintJob) => (
                    <option key={job.id} value={job.id}>
                      {job.documentName} ({job.pageCount} pages)
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Assign to Agent (optional)
                </label>
                <select
                  value={selectedAgentId}
                  onChange={(e) => setSelectedAgentId(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">Select an agent...</option>
                  {agents?.map((agent) => (
                    <option key={agent.id} value={agent.id}>
                      {agent.name} ({agent.status})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Assign to User (optional)
                </label>
                <input
                  type="text"
                  value={selectedUserId}
                  onChange={(e) => setSelectedUserId(e.target.value)}
                  placeholder="Enter user ID..."
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
            <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowAssignModal(false);
                  setSelectedAgentId('');
                  setSelectedUserId('');
                }}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateAssignment}
                disabled={!selectedJobId || createAssignmentMutation.isPending}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {createAssignmentMutation.isPending ? 'Creating...' : 'Create Assignment'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
