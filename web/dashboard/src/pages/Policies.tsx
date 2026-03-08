import { useState } from 'react';
import {
  PolicyList,
  PolicyForm,
  PolicyTemplates,
  PolicyEvaluation,
  PolicyHistory,
} from '@/features/policies';
import { policyApi } from '@/features/policies/api';
import type { PrintPolicy } from '@/types';

export const Policies = () => {
  const [isCreating, setIsCreating] = useState(false);
  const [editingPolicyId, setEditingPolicyId] = useState<string | null>(null);
  const [viewingHistoryForId, setViewingHistoryForId] = useState<string | null>(null);
  const [showTemplates, setShowTemplates] = useState(false);
  const [showEvaluation, setShowEvaluation] = useState(false);
  const [exportingPolicyId, setExportingPolicyId] = useState<string | null>(null);

  const handleCreatePolicy = () => {
    setIsCreating(true);
  };

  const handleEditPolicy = (policyId: string) => {
    setEditingPolicyId(policyId);
  };

  const handleCloseForm = () => {
    setIsCreating(false);
    setEditingPolicyId(null);
  };

  const handleSavePolicy = () => {
    setIsCreating(false);
    setEditingPolicyId(null);
  };

  const handleViewHistory = (policyId: string) => {
    setViewingHistoryForId(policyId);
  };

  const handleExportPolicy = async (policyId: string) => {
    setExportingPolicyId(policyId);
    try {
      const blob = await policyApi.export(policyId);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `policy-${policyId}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to export policy:', error);
    } finally {
      setExportingPolicyId(null);
    }
  };

  const handleExportAllPolicies = async () => {
    try {
      const policies = await policyApi.list();
      const blob = new Blob([JSON.stringify(policies, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const timestamp = new Date().toISOString().slice(0, 10);
      a.download = `print-policies-${timestamp}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to export policies:', error);
    }
  };

  return (
    <div className="space-y-6" data-testid="policies-page">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1
            data-testid="policies-heading"
            className="text-3xl font-bold text-gray-900 dark:text-gray-100"
          >
            Print Policies
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Configure print policies to control printing behavior
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleExportAllPolicies}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors flex items-center gap-2"
            title="Export all policies as JSON"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            Export All
          </button>
          <button
            onClick={() => setShowEvaluation(!showEvaluation)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors flex items-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Test Policy
          </button>
          <button
            onClick={() => setShowTemplates(!showTemplates)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors flex items-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            Templates
          </button>
          <button
            onClick={handleCreatePolicy}
            data-testid="create-policy-button"
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Policy
          </button>
        </div>
      </div>

      {/* Evaluation Section */}
      {showEvaluation && (
        <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
          <button
            onClick={() => setShowEvaluation(false)}
            className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 mb-4"
          >
            ← Hide Evaluation
          </button>
          <PolicyEvaluation policyId={editingPolicyId || undefined} />
        </div>
      )}

      {/* Templates Section */}
      {showTemplates && (
        <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
          <button
            onClick={() => setShowTemplates(false)}
            className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 mb-4"
          >
            ← Hide Templates
          </button>
          <PolicyTemplates
            onSelectTemplate={() => {
              setShowTemplates(false);
              setIsCreating(true);
            }}
          />
        </div>
      )}

      {/* Policies List */}
      <div data-testid="policies-list-section">
        <PolicyList
          onCreatePolicy={handleCreatePolicy}
          onEditPolicy={handleEditPolicy}
          onExportPolicy={handleExportPolicy}
          onViewHistory={handleViewHistory}
          exportingPolicyId={exportingPolicyId}
        />
      </div>

      {/* Create/Edit Modal */}
      {(isCreating || editingPolicyId) && (
        <PolicyForm
          policy={editingPolicyId ? ({ id: editingPolicyId } as PrintPolicy) : null}
          onClose={handleCloseForm}
          onSave={handleSavePolicy}
          mode={editingPolicyId ? 'edit' : 'create'}
        />
      )}

      {/* History Modal */}
      {viewingHistoryForId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 overflow-y-auto">
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full mx-4 my-8 p-6">
            <PolicyHistory
              policyId={viewingHistoryForId}
              onClose={() => setViewingHistoryForId(null)}
            />
          </div>
        </div>
      )}
    </div>
  );
};
