/**
 * JobAssignments Page
 * Admin interface for managing job assignments
 */

import { JobAssignment } from '@/features/jobs';

export const JobAssignmentsPage = () => {
  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="md:flex md:items-center md:justify-between mb-6">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Job Assignments
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Assign and manage print job routing to agents and users
          </p>
        </div>
      </div>

      <JobAssignment />
    </div>
  );
};
