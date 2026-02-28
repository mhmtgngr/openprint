import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { policiesApi } from '@/services/api';
import type { PrintPolicy, CreatePolicyRequest } from '@/types';

export const Policies = () => {
  const queryClient = useQueryClient();
  const [isCreating, setIsCreating] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<PrintPolicy | null>(null);

  const { data: policies, isLoading } = useQuery({
    queryKey: ['policies'],
    queryFn: () => policiesApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: (data: CreatePolicyRequest) => policiesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      setIsCreating(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreatePolicyRequest> }) =>
      policiesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      setEditingPolicy(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => policiesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, isEnabled }: { id: string; isEnabled: boolean }) =>
      policiesApi.toggle(id, isEnabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
  });

  const sortedPolicies = policies?.sort((a, b) => b.priority - a.priority) || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            Print Policies
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Define rules that control print job behavior across your organization
          </p>
        </div>
        <button
          onClick={() => setIsCreating(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Create Policy
        </button>
      </div>

      {/* Policies List */}
      <div className="space-y-4">
        {isLoading ? (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            Loading policies...
          </div>
        ) : sortedPolicies.length === 0 ? (
          <div className="bg-white dark:bg-gray-800 rounded-xl p-12 text-center border border-gray-200 dark:border-gray-700">
            <svg className="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
              No policies configured
            </h3>
            <p className="text-gray-500 dark:text-gray-400 mb-4">
              Create your first print policy to enforce printing rules
            </p>
            <button
              onClick={() => setIsCreating(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              Create Policy
            </button>
          </div>
        ) : (
          sortedPolicies.map((policy) => (
            <PolicyCard
              key={policy.id}
              policy={policy}
              onEdit={() => setEditingPolicy(policy)}
              onDelete={() => deleteMutation.mutate(policy.id)}
              onToggle={(enabled) => toggleMutation.mutate({ id: policy.id, isEnabled: enabled })}
              isDeleting={deleteMutation.isPending}
              isToggling={toggleMutation.isPending}
            />
          ))
        )}
      </div>

      {/* Create/Edit Modal */}
      {(isCreating || editingPolicy) && (
        <PolicyFormModal
          policy={editingPolicy}
          onClose={() => {
            setIsCreating(false);
            setEditingPolicy(null);
          }}
          onSave={(data) => {
            if (editingPolicy) {
              updateMutation.mutate({ id: editingPolicy.id, data });
            } else {
              createMutation.mutate(data);
            }
          }}
          isLoading={createMutation.isPending || updateMutation.isPending}
        />
      )}
    </div>
  );
};

interface PolicyCardProps {
  policy: PrintPolicy;
  onEdit: () => void;
  onDelete: () => void;
  onToggle: (enabled: boolean) => void;
  isDeleting: boolean;
  isToggling: boolean;
}

const PolicyCard = ({ policy, onEdit, onDelete, onToggle, isDeleting, isToggling }: PolicyCardProps) => {
  return (
    <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border ${policy.isEnabled ? 'border-gray-200 dark:border-gray-700' : 'border-gray-300 dark:border-gray-600 opacity-75'}`}>
      <div className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div className={`p-3 rounded-lg ${policy.isEnabled ? 'bg-blue-100 dark:bg-blue-900/30' : 'bg-gray-100 dark:bg-gray-700'}`}>
              <svg className={`w-6 h-6 ${policy.isEnabled ? 'text-blue-600 dark:text-blue-400' : 'text-gray-500'}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {policy.name}
              </h3>
              {policy.description && (
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  {policy.description}
                </p>
              )}
              <div className="flex items-center gap-4 mt-2">
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  Priority: {policy.priority}
                </span>
                <span className={`text-xs px-2 py-1 rounded-full ${policy.isEnabled ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'}`}>
                  {policy.isEnabled ? 'Active' : 'Disabled'}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => onToggle(!policy.isEnabled)}
              disabled={isToggling}
              className="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 disabled:opacity-50"
              title={policy.isEnabled ? 'Disable' : 'Enable'}
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                {policy.isEnabled ? (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                )}
              </svg>
            </button>
            <button
              onClick={onEdit}
              className="p-2 text-gray-500 hover:text-blue-600 dark:hover:text-blue-400"
              title="Edit"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </button>
            <button
              onClick={onDelete}
              disabled={isDeleting}
              className="p-2 text-gray-500 hover:text-red-600 dark:hover:text-red-400 disabled:opacity-50"
              title="Delete"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>

        {/* Policy Details */}
        <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <PolicyCondition
            label="Max Pages"
            value={policy.conditions.maxPagesPerJob ? `${policy.conditions.maxPagesPerJob} per job` : 'No limit'}
          />
          <PolicyCondition
            label="Duplex"
            value={policy.actions.forceDuplex ? 'Forced' : 'Optional'}
          />
          <PolicyCondition
            label="Color Mode"
            value={policy.actions.forceColor === true ? 'Color only' : policy.actions.forceColor === 'grayscale' ? 'Grayscale only' : 'Any'}
          />
          <PolicyCondition
            label="Approval"
            value={policy.actions.requireApproval ? 'Required' : 'Not required'}
          />
        </div>
      </div>
    </div>
  );
};

interface PolicyConditionProps {
  label: string;
  value: string;
}

const PolicyCondition = ({ label, value }: PolicyConditionProps) => (
  <div>
    <span className="text-gray-500 dark:text-gray-400">{label}:</span>
    <span className="ml-2 text-gray-900 dark:text-gray-100 font-medium">{value}</span>
  </div>
);

interface PolicyFormModalProps {
  policy: PrintPolicy | null;
  onClose: () => void;
  onSave: (data: CreatePolicyRequest) => void;
  isLoading: boolean;
}

const PolicyFormModal = ({ policy, onClose, onSave, isLoading }: PolicyFormModalProps) => {
  const [name, setName] = useState(policy?.name || '');
  const [description, setDescription] = useState(policy?.description || '');
  const [maxPages, setMaxPages] = useState(policy?.conditions.maxPagesPerJob || 0);
  const [forceDuplex, setForceDuplex] = useState(policy?.actions.forceDuplex || false);
  const [forceGrayscale, setForceGrayscale] = useState(policy?.actions.forceColor === 'grayscale');
  const [requireApproval, setRequireApproval] = useState(policy?.actions.requireApproval || false);
  const [maxCopies, setMaxCopies] = useState(policy?.actions.maxCopies || 1);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      name,
      description,
      conditions: {
        maxPagesPerJob: maxPages || undefined,
        requireApproval,
      },
      actions: {
        forceDuplex,
        forceColor: forceGrayscale ? 'grayscale' : undefined,
        maxCopies,
      },
      appliesTo: 'all',
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 overflow-y-auto">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full mx-4 my-8">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {policy ? 'Edit Policy' : 'Create Policy'}
          </h3>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Policy Name *
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Description
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">Conditions</h4>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Max Pages per Job
                </label>
                <input
                  type="number"
                  value={maxPages}
                  onChange={(e) => setMaxPages(parseInt(e.target.value) || 0)}
                  min={0}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="0 = unlimited"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Max Copies
                </label>
                <input
                  type="number"
                  value={maxCopies}
                  onChange={(e) => setMaxCopies(parseInt(e.target.value) || 1)}
                  min={1}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>

          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">Actions</h4>
            <div className="space-y-3">
              <label className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={forceDuplex}
                  onChange={(e) => setForceDuplex(e.target.checked)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Force duplex (double-sided) printing</span>
              </label>
              <label className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={forceGrayscale}
                  onChange={(e) => setForceGrayscale(e.target.checked)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Force grayscale (black & white only)</span>
              </label>
              <label className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={requireApproval}
                  onChange={(e) => setRequireApproval(e.target.checked)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Require manual approval for print jobs</span>
              </label>
            </div>
          </div>

          <div className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Saving...' : policy ? 'Update' : 'Create'} Policy
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
