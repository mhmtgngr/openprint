/**
 * AuditLogsTable Component
 * Displays audit logs with filtering and export capabilities
 */

import { useState } from 'react';
import { format } from 'date-fns';
import type { AuditEvent, AuditCategory, AuditOutcome } from '../types';

export interface AuditLogsTableProps {
  logs?: AuditEvent[];
  isLoading?: boolean;
  error?: string | null;
  onExport?: (format: 'csv' | 'json' | 'xlsx') => void;
  onRefresh?: () => void;
  totalCount?: number;
}

export const AuditLogsTable = ({
  logs = [],
  isLoading = false,
  error = null,
  onExport,
  onRefresh,
  totalCount = 0,
}: AuditLogsTableProps) => {
  const [search, setSearch] = useState('');
  const [actionFilter, setActionFilter] = useState<string>('all');
  const [userFilter, setUserFilter] = useState<string>('all');
  const [categoryFilter, setCategoryFilter] = useState<string>('all');

  // Get unique values for filters
  const uniqueUsers = Array.from(new Set(logs.map((log) => log.user_name)));
  const uniqueActions = Array.from(new Set(logs.map((log) => log.event_type)));
  const uniqueCategories = Array.from(new Set(logs.map((log) => log.category)));

  // Filter logs
  const filteredLogs = logs.filter((log) => {
    const matchesSearch =
      !search ||
      log.user_name.toLowerCase().includes(search.toLowerCase()) ||
      log.event_type.toLowerCase().includes(search.toLowerCase()) ||
      log.action.toLowerCase().includes(search.toLowerCase()) ||
      log.resource_id.toLowerCase().includes(search.toLowerCase());

    const matchesAction = actionFilter === 'all' || log.event_type === actionFilter;
    const matchesUser = userFilter === 'all' || log.user_id === userFilter;
    const matchesCategory = categoryFilter === 'all' || log.category === categoryFilter;

    return matchesSearch && matchesAction && matchesUser && matchesCategory;
  });

  const outcomeConfig: Record<AuditOutcome, { label: string; className: string }> = {
    success: {
      label: 'Success',
      className: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
    },
    failure: {
      label: 'Failure',
      className: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
    },
    error: {
      label: 'Error',
      className: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
    },
  };

  const categoryConfig: Record<AuditCategory, { label: string; icon: string }> = {
    authentication: { label: 'Authentication', icon: '🔐' },
    authorization: { label: 'Authorization', icon: '🛡️' },
    data_access: { label: 'Data Access', icon: '📄' },
    data_modification: { label: 'Data Modification', icon: '✏️' },
    system: { label: 'System', icon: '⚙️' },
    compliance: { label: 'Compliance', icon: '✔️' },
    security: { label: 'Security', icon: '🔒' },
  };

  if (error) {
    return (
      <div
        className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6"
        data-testid="audit-logs-error"
      >
        <div className="flex items-start gap-3">
          <svg
            className="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <div>
            <h3 className="text-sm font-medium text-red-800 dark:text-red-400">
              Failed to load audit logs
            </h3>
            <p className="text-sm text-red-700 dark:text-red-300 mt-1">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid="audit-logs-section">
      {/* Filters */}
      <div
        className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700"
        data-testid="audit-logs-filters"
      >
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search logs..."
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              data-testid="audit-logs-search"
            />
          </div>
          <select
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            data-testid="audit-logs-category-filter"
          >
            <option value="all">All Categories</option>
            {uniqueCategories.map((cat) => (
              <option key={cat} value={cat}>
                {categoryConfig[cat as AuditCategory]?.label || cat}
              </option>
            ))}
          </select>
          <select
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            data-testid="audit-logs-action-filter"
          >
            <option value="all">All Actions</option>
            {uniqueActions.map((action) => (
              <option key={action} value={action}>
                {action}
              </option>
            ))}
          </select>
          <select
            value={userFilter}
            onChange={(e) => setUserFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            data-testid="audit-logs-user-filter"
          >
            <option value="all">All Users</option>
            {uniqueUsers.map((user) => (
              <option key={user} value={user}>
                {user}
              </option>
            ))}
          </select>
          {onRefresh && (
            <button
              onClick={onRefresh}
              disabled={isLoading}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
              data-testid="audit-logs-refresh"
            >
              <RefreshIcon className="w-5 h-5" />
            </button>
          )}
        </div>
        <div className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          Showing {filteredLogs.length} of {totalCount} logs
        </div>
      </div>

      {/* Audit Logs Table */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        {isLoading ? (
          <div className="p-8 flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : filteredLogs.length === 0 ? (
          <div
            className="p-12 text-center"
            data-testid="audit-logs-empty"
          >
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
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
            <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
              No audit logs found
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {search || actionFilter !== 'all' || userFilter !== 'all'
                ? 'Try adjusting your filters'
                : 'Audit logs will appear here as users interact with the system'}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table
              className="w-full"
              data-testid="audit-logs-table"
            >
              <thead className="bg-gray-50 dark:bg-gray-700/50">
                <tr>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="timestamp-column"
                  >
                    Timestamp
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="user-column"
                  >
                    User
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="action-column"
                  >
                    Action
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="resource-column"
                  >
                    Resource
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="details-column"
                  >
                    Details
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="outcome-column"
                  >
                    Outcome
                  </th>
                  <th
                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
                    data-testid="ip-address-column"
                  >
                    IP Address
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {filteredLogs.map((log) => (
                  <tr
                    key={log.id}
                    className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                    data-testid="audit-log-entry"
                  >
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                      {format(new Date(log.timestamp), 'MMM d, yyyy HH:mm:ss')}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">
                      {log.user_name}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <span className="text-lg">
                          {categoryConfig[log.category]?.icon || '📋'}
                        </span>
                        <span className="text-sm text-gray-900 dark:text-gray-100">
                          {log.event_type}
                        </span>
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                        {log.action}
                      </p>
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <span className="text-gray-600 dark:text-gray-400">
                        {log.resource_type}:
                      </span>{' '}
                      <span className="text-gray-900 dark:text-gray-100 font-mono text-xs">
                        {log.resource_id}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400 max-w-xs truncate">
                      {log.details ? JSON.stringify(log.details) : '-'}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${
                          outcomeConfig[log.outcome]?.className || ''
                        }`}
                      >
                        {outcomeConfig[log.outcome]?.label || log.outcome}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-500 font-mono">
                      {log.ip_address}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Export Actions */}
      {onExport && filteredLogs.length > 0 && (
        <div className="flex justify-end gap-2" data-testid="audit-logs-export-actions">
          <button
            onClick={() => onExport('csv')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-sm"
            data-testid="export-csv-button"
          >
            Export CSV
          </button>
          <button
            onClick={() => onExport('json')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-sm"
            data-testid="export-json-button"
          >
            Export JSON
          </button>
          <button
            onClick={() => onExport('xlsx')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors text-sm"
            data-testid="export-xlsx-button"
          >
            Export XLSX
          </button>
        </div>
      )}
    </div>
  );
};

const RefreshIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
    />
  </svg>
);

export default AuditLogsTable;
