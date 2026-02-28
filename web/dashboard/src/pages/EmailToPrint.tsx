import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { emailToPrintApi, printersApi } from '@/services/api';

export const EmailToPrint = () => {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [testEmailSent, setTestEmailSent] = useState(false);

  const { data: config, isLoading } = useQuery({
    queryKey: ['email-to-print', 'config'],
    queryFn: () => emailToPrintApi.getConfig(),
  });

  const { data: printers } = useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
  });

  const { data: jobs } = useQuery({
    queryKey: ['email-to-print', 'jobs'],
    queryFn: () => emailToPrintApi.getJobs({ limit: 20 }),
  });

  const updateMutation = useMutation({
    mutationFn: emailToPrintApi.updateConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email-to-print'] });
      setIsEditing(false);
    },
  });

  const testMutation = useMutation({
    mutationFn: emailToPrintApi.testEmail,
    onSuccess: () => {
      setTestEmailSent(true);
      setTimeout(() => setTestEmailSent(false), 5000);
    },
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            Email-to-Print
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Configure email-based printing and manage incoming print jobs
          </p>
        </div>
        <button
          onClick={() => setIsEditing(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Configure
        </button>
      </div>

      {/* Configuration Status Card */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Configuration Status
          </h2>
        </div>
        <div className="p-6">
          {isLoading ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              Loading configuration...
            </div>
          ) : config ? (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Status</p>
                  <p className={`text-lg font-medium ${config.isEnabled ? 'text-green-600 dark:text-green-400' : 'text-gray-500'}`}>
                    {config.isEnabled ? 'Enabled' : 'Disabled'}
                  </p>
                </div>
                <div className={`px-4 py-2 rounded-lg ${config.isEnabled ? 'bg-green-100 dark:bg-green-900/30' : 'bg-gray-100 dark:bg-gray-700'}`}>
                  {config.isEnabled ? (
                    <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  ) : (
                    <svg className="w-8 h-8 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                    </svg>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Email Address</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-gray-100">
                    {config.emailPrefix}@org.openprint.cloud
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Default Printer</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-gray-100">
                    {config.defaultPrinterId
                      ? printers?.find((p) => p.id === config.defaultPrinterId)?.name || config.defaultPrinterId
                      : 'Not set'}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Auto Release</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-gray-100">
                    {config.autoRelease ? 'Yes' : 'No (Manual approval required)'}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Max Attachments</p>
                  <p className="text-lg font-medium text-gray-900 dark:text-gray-100">
                    {config.maxAttachments || 10}
                  </p>
                </div>
              </div>

              {config.allowedSenders && config.allowedSenders.length > 0 && (
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Allowed Senders</p>
                  <div className="flex flex-wrap gap-2">
                    {config.allowedSenders.map((sender, i) => (
                      <span
                        key={i}
                        className="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded-full text-sm"
                      >
                        {sender}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              <div className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
                <button
                  onClick={() => setIsEditing(true)}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                >
                  Edit Configuration
                </button>
                <button
                  onClick={() => testMutation.mutate()}
                  disabled={testMutation.isPending}
                  className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
                >
                  {testMutation.isPending ? 'Sending...' : 'Send Test Email'}
                </button>
              </div>

              {testEmailSent && (
                <div className="p-4 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-lg flex items-center gap-2">
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  Test email sent successfully!
                </div>
              )}
            </div>
          ) : null}
        </div>
      </div>

      {/* Recent Email Print Jobs */}
      {jobs && jobs.data.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Recent Email Print Jobs
            </h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-700/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    From
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Subject
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Attachments
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Received
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {jobs.data.map((job) => (
                  <tr key={job.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">
                      {job.fromEmail}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400 max-w-xs truncate">
                      {job.subject}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
                      {job.attachmentCount}
                    </td>
                    <td className="px-6 py-4">
                      <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                        job.status === 'completed'
                          ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                        : job.status === 'failed'
                          ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'
                        : job.status === 'processing'
                          ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                      }`}>
                        {job.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
                      {new Date(job.createdAt).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {isEditing && config && (
        <ConfigModal
          config={config}
          printers={printers || []}
          onClose={() => setIsEditing(false)}
          onSave={(data) => updateMutation.mutate(data)}
          isLoading={updateMutation.isPending}
        />
      )}
    </div>
  );
};

interface ConfigModalProps {
  config: import('@/types').EmailToPrintConfig;
  printers: import('@/types').Printer[];
  onClose: () => void;
  onSave: (data: import('@/types').UpdateEmailConfigRequest) => void;
  isLoading: boolean;
}

const ConfigModal = ({ config, printers, onClose, onSave, isLoading }: ConfigModalProps) => {
  const [isEnabled, setIsEnabled] = useState(config.isEnabled);
  const [defaultPrinterId, setDefaultPrinterId] = useState(config.defaultPrinterId || '');
  const [autoRelease, setAutoRelease] = useState(config.autoRelease || false);
  const [requireApproval, setRequireApproval] = useState(config.requireApproval || false);
  const [maxAttachments, setMaxAttachments] = useState(config.maxAttachments || 10);
  const [allowedSenders, setAllowedSenders] = useState(config.allowedSenders?.join(', ') || '');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      isEnabled,
      defaultPrinterId: defaultPrinterId || undefined,
      autoRelease,
      requireApproval,
      maxAttachments,
      allowedSenders: allowedSenders
        ? allowedSenders.split(',').map((s) => s.trim()).filter(Boolean)
        : undefined,
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-lg w-full mx-4">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Configure Email-to-Print
          </h3>
        </div>
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div>
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
                className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Enable Email-to-Print</span>
            </label>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Default Printer
            </label>
            <select
              value={defaultPrinterId}
              onChange={(e) => setDefaultPrinterId(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="">No default</option>
              {printers.map((printer) => (
                <option key={printer.id} value={printer.id}>
                  {printer.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Allowed Senders
            </label>
            <input
              type="text"
              value={allowedSenders}
              onChange={(e) => setAllowedSenders(e.target.value)}
              placeholder="user@example.com, @example.com"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Comma-separated email addresses or domains (leave empty for any sender)
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Max Attachments per Email
            </label>
            <input
              type="number"
              value={maxAttachments}
              onChange={(e) => setMaxAttachments(parseInt(e.target.value) || 1)}
              min={1}
              max={50}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div className="space-y-2">
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={autoRelease}
                onChange={(e) => setAutoRelease(e.target.checked)}
                className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Auto-release print jobs</span>
            </label>
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={requireApproval}
                onChange={(e) => setRequireApproval(e.target.checked)}
                className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Require admin approval</span>
            </label>
          </div>

          <div className="flex gap-3 pt-4">
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
              {isLoading ? 'Saving...' : 'Save Configuration'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
