/**
 * DiscoveredPrinterList Component
 * View all agent-discovered printers across the organization
 */

import { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { useDiscoveredPrinters, useDeleteDiscoveredPrinter, useSetDefaultPrinter } from '../agents/useAgents';
import { getPrinterTypeColor } from '@/api/agentApi';

interface DiscoveredPrinterListProps {
  agentFilter?: string;
  statusFilter?: string;
}

export const DiscoveredPrinterList = ({
  agentFilter,
  statusFilter,
}: DiscoveredPrinterListProps) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState<'all' | 'local' | 'network' | 'shared'>('all');

  const { data: printersData, isLoading, error } = useDiscoveredPrinters({
    agentId: agentFilter,
    status: statusFilter,
    search: searchQuery || undefined,
  });

  const deleteMutation = useDeleteDiscoveredPrinter();
  const setDefaultMutation = useSetDefaultPrinter();

  const handleDelete = async (printerId: string, printerName: string) => {
    if (confirm(`Remove printer "${printerName}" from the registry?`)) {
      await deleteMutation.mutateAsync(printerId);
    }
  };

  const handleSetDefault = async (printerId: string) => {
    await setDefaultMutation.mutateAsync(printerId);
  };

  const printers = printersData?.printers || [];

  // Apply client-side filters
  const filteredPrinters = printers.filter((printer) => {
    if (typeFilter !== 'all' && printer.type !== typeFilter) return false;
    if (
      searchQuery &&
      !printer.name.toLowerCase().includes(searchQuery.toLowerCase()) &&
      !printer.driver.toLowerCase().includes(searchQuery.toLowerCase())
    )
      return false;
    return true;
  });

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'local':
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
            />
          </svg>
        );
      case 'network':
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0"
            />
          </svg>
        );
      case 'shared':
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
            />
          </svg>
        );
      default:
        return null;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'available':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
            Available
          </span>
        );
      case 'offline':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300">
            <span className="w-1.5 h-1.5 rounded-full bg-gray-400" />
            Offline
          </span>
        );
      case 'error':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
            Error
          </span>
        );
      default:
        return null;
    }
  };

  const typeCounts = {
    local: printers.filter((p) => p.type === 'local').length,
    network: printers.filter((p) => p.type === 'network').length,
    shared: printers.filter((p) => p.type === 'shared').length,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Discovered Printers
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {printers.length} {printers.length === 1 ? 'printer' : 'printers'} discovered across
            all agents
          </p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-600 dark:text-gray-400">Total Printers</p>
          <p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {printers.length}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-blue-500" />
            <p className="text-sm text-gray-600 dark:text-gray-400">Local</p>
          </div>
          <p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {typeCounts.local}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-purple-500" />
            <p className="text-sm text-gray-600 dark:text-gray-400">Network</p>
          </div>
          <p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {typeCounts.network}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-2">
            <div className="w-3 h-3 rounded-full bg-amber-500" />
            <p className="text-sm text-gray-600 dark:text-gray-400">Shared</p>
          </div>
          <p className="mt-1 text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {typeCounts.shared}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="flex-1">
          <input
            type="text"
            placeholder="Search printers by name or driver..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full px-4 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-500 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setTypeFilter('all')}
            className={`px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              typeFilter === 'all'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            All Types
          </button>
          <button
            onClick={() => setTypeFilter('local')}
            className={`px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              typeFilter === 'local'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            Local
          </button>
          <button
            onClick={() => setTypeFilter('network')}
            className={`px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              typeFilter === 'network'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            Network
          </button>
          <button
            onClick={() => setTypeFilter('shared')}
            className={`px-3 py-2 text-sm font-medium rounded-lg transition-colors ${
              typeFilter === 'shared'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            Shared
          </button>
        </div>
      </div>

      {/* Printer List */}
      {isLoading ? (
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <div
              key={i}
              className="bg-gray-100 dark:bg-gray-800 rounded-lg h-20 animate-pulse"
            />
          ))}
        </div>
      ) : error ? (
        <div className="text-center py-12">
          <svg
            className="mx-auto h-12 w-12 text-red-400"
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
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            Error loading printers
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {(error as Error).message}
          </p>
        </div>
      ) : filteredPrinters.length === 0 ? (
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
              d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            No printers found
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Try adjusting your search or filter criteria.
          </p>
        </div>
      ) : (
        <div className="overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Printer
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Capabilities
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Last Seen
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
              {filteredPrinters.map((printer) => (
                <tr key={printer.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div
                        className="flex-shrink-0 h-8 w-8 rounded-full flex items-center justify-center"
                        style={{ backgroundColor: getPrinterTypeColor(printer.type) + '20' }}
                      >
                        <div
                          className="text-xs"
                          style={{ color: getPrinterTypeColor(printer.type) }}
                        >
                          {getTypeIcon(printer.type)}
                        </div>
                      </div>
                      <div className="ml-4">
                        <div className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
                          {printer.name}
                          {printer.isDefault && (
                            <span className="px-1.5 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300 rounded">
                              Default
                            </span>
                          )}
                        </div>
                        <div className="text-sm text-gray-500 dark:text-gray-400">
                          {printer.driver}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className="px-2 py-1 text-xs font-medium rounded capitalize"
                      style={{
                        backgroundColor: getPrinterTypeColor(printer.type) + '20',
                        color: getPrinterTypeColor(printer.type),
                      }}
                    >
                      {printer.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                    <div className="flex flex-wrap gap-1">
                      {printer.capabilities?.supportsColor && (
                        <span className="px-2 py-0.5 text-xs font-medium bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300 rounded">
                          Color
                        </span>
                      )}
                      {printer.capabilities?.supportsDuplex && (
                        <span className="px-2 py-0.5 text-xs font-medium bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-300 rounded">
                          Duplex
                        </span>
                      )}
                      <span className="px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-700 dark:bg-gray-700 dark:text-gray-300 rounded">
                        {printer.capabilities?.resolution || 'Unknown'}
                      </span>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">{getStatusBadge(printer.status)}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                    {formatDistanceToNow(new Date(printer.lastSeen), { addSuffix: true })}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex items-center justify-end gap-2">
                      {!printer.isDefault && printer.status === 'available' && (
                        <button
                          onClick={() => handleSetDefault(printer.id)}
                          disabled={setDefaultMutation.isPending}
                          className="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300 disabled:opacity-50"
                        >
                          Set Default
                        </button>
                      )}
                      <button
                        onClick={() => handleDelete(printer.id, printer.name)}
                        disabled={deleteMutation.isPending}
                        className="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50"
                      >
                        Remove
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
