import { type FC, useState } from 'react';
import { usePolicyEvaluation, useJobTest } from './usePolicyEvaluation';
import { usePolicies } from './usePolicies';
import type { PolicyEvaluationContext, PolicyEvaluationResult, PolicyTestJob } from './types';

interface PolicyEvaluationProps {
  policyId?: string;
  onResult?: (result: PolicyEvaluationResult) => void;
}

export const PolicyEvaluation: FC<PolicyEvaluationProps> = ({
  policyId,
  onResult,
}) => {
  const { data: policies } = usePolicies();
  const evaluateMutation = usePolicyEvaluation();
  const testJobMutation = useJobTest();

  const [selectedPolicyId, setSelectedPolicyId] = useState(policyId || '');
  const [testJob, setTestJob] = useState<PolicyTestJob>({
    documentName: 'Test Document.pdf',
    pageCount: 10,
    colorMode: true,
    copies: 1,
    userRole: 'user',
    printerId: '',
  });

  const [evaluationResult, setEvaluationResult] = useState<PolicyEvaluationResult | null>(null);

  const handleEvaluate = async () => {
    if (!selectedPolicyId) return;

    const context: PolicyEvaluationContext = {
      userRole: testJob.userRole,
      printerId: testJob.printerId || undefined,
      documentType: testJob.documentName.split('.').pop(),
      pageCount: testJob.pageCount,
      colorMode: testJob.colorMode,
      timestamp: new Date().toISOString(),
    };

    try {
      const result = await evaluateMutation.mutateAsync({
        policyId: selectedPolicyId,
        context,
      });
      setEvaluationResult(result);
      onResult?.(result);
    } catch (error) {
      console.error('Evaluation failed:', error);
    }
  };

  const handleTestAll = async () => {
    try {
      const response = await testJobMutation.mutateAsync(testJob);
      // Show all results
      console.log('Test results:', response.results);
    } catch (error) {
      console.error('Test failed:', error);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
        Policy Evaluation
      </h2>

      <div className="space-y-6">
        {/* Policy Selection */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Select Policy to Test
          </label>
          <select
            value={selectedPolicyId}
            onChange={(e) => setSelectedPolicyId(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="">Choose a policy...</option>
            {policies?.map((policy) => (
              <option key={policy.id} value={policy.id}>
                {policy.name} (Priority: {policy.priority})
              </option>
            ))}
          </select>
        </div>

        {/* Test Job Configuration */}
        <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
            Test Job Configuration
          </h3>

          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Document Name
              </label>
              <input
                type="text"
                value={testJob.documentName}
                onChange={(e) => setTestJob({ ...testJob, documentName: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Page Count
              </label>
              <input
                type="number"
                value={testJob.pageCount}
                onChange={(e) => setTestJob({ ...testJob, pageCount: parseInt(e.target.value) || 0 })}
                min={1}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Copies
              </label>
              <input
                type="number"
                value={testJob.copies}
                onChange={(e) => setTestJob({ ...testJob, copies: parseInt(e.target.value) || 1 })}
                min={1}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                User Role
              </label>
              <select
                value={testJob.userRole}
                onChange={(e) => setTestJob({ ...testJob, userRole: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
                <option value="owner">Owner</option>
                <option value="student">Student</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Color Mode
              </label>
              <select
                value={testJob.colorMode ? 'color' : 'grayscale'}
                onChange={(e) => setTestJob({ ...testJob, colorMode: e.target.value === 'color' })}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="color">Color</option>
                <option value="grayscale">Grayscale</option>
              </select>
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Printer ID (Optional)
              </label>
              <input
                type="text"
                value={testJob.printerId || ''}
                onChange={(e) => setTestJob({ ...testJob, printerId: e.target.value })}
                placeholder="Leave empty to test all printers"
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <button
            onClick={handleEvaluate}
            disabled={!selectedPolicyId || evaluateMutation.isPending}
            data-testid="evaluate-policy-button"
            className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {evaluateMutation.isPending ? 'Evaluating...' : 'Evaluate Policy'}
          </button>
          <button
            onClick={handleTestAll}
            disabled={testJobMutation.isPending}
            data-testid="test-job-button"
            className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {testJobMutation.isPending ? 'Testing...' : 'Test All Policies'}
          </button>
        </div>

        {/* Results */}
        {evaluationResult && (
          <div
            data-testid="evaluation-result"
            className="mt-6 p-4 rounded-lg border"
          >
            <div
              className={`flex items-center gap-2 mb-3 ${
                evaluationResult.matched
                  ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800'
                  : 'bg-gray-50 dark:bg-gray-700/50 border-gray-200 dark:border-gray-600'
              }`}
            >
              {evaluationResult.matched ? (
                <svg className="w-5 h-5 text-green-600 dark:text-green-400" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-gray-600 dark:text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
              )}
              <span className="font-medium text-gray-900 dark:text-gray-100">
                {evaluationResult.matched ? 'Policy Matched' : 'Policy Not Matched'}
              </span>
            </div>

            {evaluationResult.policyName && (
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                Policy: <span className="font-medium">{evaluationResult.policyName}</span>
              </p>
            )}

            <p className="text-sm text-gray-700 dark:text-gray-300 mb-3">
              {evaluationResult.message}
            </p>

            {evaluationResult.modifiedSettings && (
              <div className="border-t border-gray-200 dark:border-gray-600 pt-3 mt-3">
                <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                  Modified Settings
                </p>
                <div className="grid grid-cols-2 gap-2 text-sm">
                  {evaluationResult.modifiedSettings.duplex !== undefined && (
                    <div>
                      <span className="text-gray-500">Duplex:</span>{' '}
                      <span className="font-medium">{evaluationResult.modifiedSettings.duplex ? 'Enabled' : 'Disabled'}</span>
                    </div>
                  )}
                  {evaluationResult.modifiedSettings.color !== undefined && (
                    <div>
                      <span className="text-gray-500">Color:</span>{' '}
                      <span className="font-medium">{evaluationResult.modifiedSettings.color ? 'Color' : 'Grayscale'}</span>
                    </div>
                  )}
                  {evaluationResult.modifiedSettings.copies !== undefined && (
                    <div>
                      <span className="text-gray-500">Copies:</span>{' '}
                      <span className="font-medium">{evaluationResult.modifiedSettings.copies}</span>
                    </div>
                  )}
                  {evaluationResult.modifiedSettings.approvalRequired !== undefined && (
                    <div>
                      <span className="text-gray-500">Approval:</span>{' '}
                      <span className="font-medium">{evaluationResult.modifiedSettings.approvalRequired ? 'Required' : 'Not Required'}</span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
