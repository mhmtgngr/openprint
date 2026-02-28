/**
 * Devices Component - Main listing page for devices/agents/printers
 * Features table view, filtering, and device management
 */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { formatDistanceToNow } from 'date-fns';
import type { DeviceAgent, DevicePrinter, DeviceListParams } from './types';
import { DEVICE_STATUS_CONFIG } from './types';
import { devicesApi, isAgent } from './api';
import { DeviceCard } from './DeviceCard';

interface RegisterDeviceFormData {
  device_name: string;
  location: string;
  printer_type: string;
}

interface DevicesProps {
  onDeviceClick?: (device: DeviceAgent | DevicePrinter) => void;
}

export const Devices = ({ onDeviceClick }: DevicesProps) => {
  const queryClient = useQueryClient();
  const [viewMode, setViewMode] = useState<'table' | 'grid'>('table');
  const [filters, setFilters] = useState<DeviceListParams>({
    status: 'all',
    type: 'all',
    search: '',
  });
  const [isRegisterModalOpen, setIsRegisterModalOpen] = useState(false);
  const [registerForm, setRegisterForm] = useState<RegisterDeviceFormData>({
    device_name: '',
    location: '',
    printer_type: 'local',
  });

  // Fetch devices
  const {
    data: devicesData,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['devices', filters],
    queryFn: () => devicesApi.list(filters),
    refetchInterval: 30000, // Refresh every 30 seconds for status updates
  });

  // Delete device mutation
  const deleteMutation = useMutation({
    mutationFn: ({ id, type }: { id: string; type: 'agent' | 'printer' }) =>
      devicesApi.delete(id, type),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
    },
  });

  // Toggle printer active status mutation
  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      devicesApi.setPrinterActive(id, !isActive),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
    },
  });

  // Register device mutation
  const registerMutation = useMutation({
    mutationFn: (data: RegisterDeviceFormData) =>
      fetch(`${import.meta.env.VITE_API_URL || '/api/v1'}/devices/register`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('auth_tokens') ? JSON.parse(localStorage.getItem('auth_tokens')!).accessToken : ''}`,
        },
        body: JSON.stringify(data),
      }).then((res) => {
        if (!res.ok) throw new Error('Failed to register device');
        return res.json();
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
      setIsRegisterModalOpen(false);
      setRegisterForm({ device_name: '', location: '', printer_type: 'local' });
    },
  });

  const handleDelete = (id: string, type: 'agent' | 'printer', name: string) => {
    if (
      confirm(
        `Are you sure you want to delete ${type === 'agent' ? 'agent' : 'printer'} "${name}"?`
      )
    ) {
      deleteMutation.mutate({ id, type });
    }
  };

  const handleToggle = (id: string, isActive: boolean) => {
    toggleMutation.mutate({ id, isActive });
  };

  const openRegisterDeviceModal = () => {
    setIsRegisterModalOpen(true);
  };

  const closeRegisterDeviceModal = () => {
    setIsRegisterModalOpen(false);
    setRegisterForm({ device_name: '', location: '', printer_type: 'local' });
  };

  const handleRegisterSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    registerMutation.mutate(registerForm);
  };

  const handleFilterChange = (key: keyof DeviceListParams, value: string) => {
    setFilters((prev) => ({
      ...prev,
      [key]: value === 'all' ? undefined : value,
    }));
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
          <div className="flex gap-2">
            <div className="h-10 w-24 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
            <div className="h-10 w-32 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-100 dark:bg-gray-700 rounded animate-pulse" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
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
          Error loading devices
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {(error as Error).message || 'Unable to load devices. Please try again.'}
        </p>
      </div>
    );
  }

  const { agents, printers, stats } = devicesData || {
    agents: [],
    printers: [],
    stats: {
      totalAgents: 0,
      onlineAgents: 0,
      totalPrinters: 0,
      onlinePrinters: 0,
      offlinePrinters: 0,
    },
  };

  // Combine agents and printers for table view
  const allDevices = [...agents, ...printers];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Devices</h1>
        <div className="flex items-center gap-2">
          <button
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
            onClick={openRegisterDeviceModal}
          >
            <PlusIcon className="w-5 h-5" />
            Add Device
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard
          label="Total Agents"
          value={stats.totalAgents}
          online={stats.onlineAgents}
          icon="server"
        />
        <StatCard
          label="Total Printers"
          value={stats.totalPrinters}
          online={stats.onlinePrinters}
          icon="printer"
        />
        <StatCard
          label="Online"
          value={stats.onlineAgents + stats.onlinePrinters}
          icon="signal"
          color="green"
        />
        <StatCard
          label="Offline"
          value={stats.totalAgents + stats.totalPrinters - (stats.onlineAgents + stats.onlinePrinters)}
          icon="signal-off"
          color="red"
        />
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center">
        {/* Search */}
        <div className="relative flex-1 max-w-md">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Search devices..."
            value={filters.search || ''}
            onChange={(e) => handleFilterChange('search', e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Status filter */}
        <select
          value={filters.status || 'all'}
          onChange={(e) => handleFilterChange('status', e.target.value)}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Status</option>
          <option value="online">Online</option>
          <option value="offline">Offline</option>
          <option value="error">Error</option>
        </select>

        {/* Type filter */}
        <select
          value={filters.type || 'all'}
          onChange={(e) => handleFilterChange('type', e.target.value)}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Types</option>
          <option value="agent">Agents</option>
          <option value="printer">Printers</option>
        </select>

        {/* View toggle */}
        <div className="flex items-center border border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden">
          <button
            onClick={() => setViewMode('table')}
            className={`px-3 py-2 transition-colors ${
              viewMode === 'table'
                ? 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100'
                : 'bg-white dark:bg-gray-800 text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
            }`}
          >
            <TableIcon className="w-5 h-5" />
          </button>
          <button
            onClick={() => setViewMode('grid')}
            className={`px-3 py-2 transition-colors ${
              viewMode === 'grid'
                ? 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100'
                : 'bg-white dark:bg-gray-800 text-gray-500 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
            }`}
          >
            <GridIcon className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Content */}
      {allDevices.length === 0 ? (
        <EmptyState onAddDevice={openRegisterDeviceModal} />
      ) : viewMode === 'table' ? (
        <TableView
          devices={allDevices}
          onDeviceClick={onDeviceClick}
          onDelete={handleDelete}
          onToggle={handleToggle}
          isDeleting={deleteMutation.isPending}
          isToggling={toggleMutation.isPending}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {allDevices.map((device) => (
            <DeviceCard
              key={device.id}
              device={device}
              onClick={() => onDeviceClick?.(device)}
              onToggleStatus={
                !isAgent(device)
                  ? () => handleToggle(device.id, (device as DevicePrinter).isActive)
                  : undefined
              }
              onDelete={() =>
                handleDelete(
                  device.id,
                  isAgent(device) ? 'agent' : 'printer',
                  device.name
                )
              }
              isToggling={toggleMutation.isPending}
            />
          ))}
        </div>
      )}

      {/* Register Device Modal */}
      {isRegisterModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full">
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Register New Device
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Add a new printer or device to your organization
              </p>
            </div>
            <form onSubmit={handleRegisterSubmit} className="p-6 space-y-4">
              <div>
                <label htmlFor="device_name" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Device Name
                </label>
                <input
                  type="text"
                  id="device_name"
                  required
                  value={registerForm.device_name}
                  onChange={(e) => setRegisterForm({ ...registerForm, device_name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                  placeholder="e.g. Office Printer 1"
                />
              </div>
              <div>
                <label htmlFor="location" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Location
                </label>
                <input
                  type="text"
                  id="location"
                  required
                  value={registerForm.location}
                  onChange={(e) => setRegisterForm({ ...registerForm, location: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                  placeholder="e.g. First Floor, Room 101"
                />
              </div>
              <div>
                <label htmlFor="printer_type" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Printer Type
                </label>
                <select
                  id="printer_type"
                  value={registerForm.printer_type}
                  onChange={(e) => setRegisterForm({ ...registerForm, printer_type: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                >
                  <option value="local">Local Printer</option>
                  <option value="network">Network Printer</option>
                  <option value="shared">Shared Printer</option>
                  <option value="usb">USB Printer</option>
                </select>
              </div>
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={closeRegisterDeviceModal}
                  className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors font-medium"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={registerMutation.isPending}
                  className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {registerMutation.isPending ? 'Registering...' : 'Register Device'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

// Table View Component
interface TableViewProps {
  devices: (DeviceAgent | DevicePrinter)[];
  onDeviceClick?: (device: DeviceAgent | DevicePrinter) => void;
  onDelete?: (id: string, type: 'agent' | 'printer', name: string) => void;
  onToggle?: (id: string, isActive: boolean) => void;
  isDeleting?: boolean;
  isToggling?: boolean;
}

const TableView = ({
  devices,
  onDeviceClick,
  onDelete,
  onToggle,
  isDeleting = false,
  isToggling = false,
}: TableViewProps) => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-900">
          <tr>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Name
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Type
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Status
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Last Seen
            </th>
            <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
          {devices.map((device) => {
            const isAgentDevice = isAgent(device);
            const status = isAgentDevice
              ? (device.status as 'online' | 'offline' | 'error')
              : device.isOnline
                ? 'online'
                : 'offline';
            const statusConfig = DEVICE_STATUS_CONFIG[status];

            return (
              <tr
                key={device.id}
                className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              >
                <td className="px-4 py-4">
                  <div className="flex items-center gap-3">
                    <div
                      className={`w-8 h-8 rounded-lg flex items-center justify-center ${statusConfig.bgColor}`}
                    >
                      {isAgentDevice ? (
                        <AgentIcon className={`w-4 h-4 ${statusConfig.textColor}`} />
                      ) : (
                        <PrinterIcon className={`w-4 h-4 ${statusConfig.textColor}`} />
                      )}
                    </div>
                    <div>
                      <div
                        className="text-sm font-medium text-gray-900 dark:text-gray-100 cursor-pointer hover:text-blue-600 dark:hover:text-blue-400"
                        onClick={() => onDeviceClick?.(device)}
                      >
                        {device.name}
                      </div>
                      {isAgentDevice ? (
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {(device as DeviceAgent).platform} · {(device as DeviceAgent).agentVersion || 'Unknown'}
                        </div>
                      ) : (
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {(device as DevicePrinter).agentName || 'Unknown Agent'}
                        </div>
                      )}
                    </div>
                  </div>
                </td>
                <td className="px-4 py-4">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200">
                    {isAgentDevice ? 'Agent' : 'Printer'}
                  </span>
                </td>
                <td className="px-4 py-4">
                  <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${statusConfig.textColor}`}>
                    <span className={`w-2 h-2 rounded-full ${statusConfig.dotColor}`} />
                    {statusConfig.label}
                  </span>
                </td>
                <td className="px-4 py-4 text-sm text-gray-500 dark:text-gray-400">
                  {formatDistanceToNow(
                    new Date(
                      isAgentDevice
                        ? device.createdAt
                        : (device as DevicePrinter).lastSeen || device.createdAt
                    ),
                    {
                      addSuffix: true,
                    }
                  )}
                </td>
                <td className="px-4 py-4" onClick={(e) => e.stopPropagation()}>
                  <div className="flex items-center justify-end gap-1">
                    {!isAgentDevice && (device as DevicePrinter).isActive !== undefined && onToggle && (
                      <button
                        onClick={() => onToggle(device.id, (device as DevicePrinter).isActive)}
                        disabled={isToggling}
                        className={`
                          relative inline-flex h-6 w-11 items-center rounded-full transition-colors
                          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                          ${(device as DevicePrinter).isActive
                            ? 'bg-blue-600 dark:bg-blue-500'
                            : 'bg-gray-200 dark:bg-gray-700'
                          }
                          disabled:opacity-50 disabled:cursor-not-allowed
                        `}
                      >
                        <span
                          className={`
                            inline-block h-4 w-4 transform rounded-full bg-white transition-transform
                            ${(device as DevicePrinter).isActive ? 'translate-x-6' : 'translate-x-1'}
                          `}
                        />
                      </button>
                    )}
                    <button
                      onClick={() =>
                        onDelete?.(
                          device.id,
                          isAgentDevice ? 'agent' : 'printer',
                          device.name
                        )
                      }
                      disabled={isDeleting}
                      className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors disabled:opacity-50"
                      title="Delete"
                    >
                      <TrashIcon className="w-4 h-4" />
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

// Stat Card Component
interface StatCardProps {
  label: string;
  value: number;
  online?: number;
  icon: string;
  color?: 'green' | 'red' | 'blue';
}

const StatCard = ({ label, value, online, icon, color = 'blue' }: StatCardProps) => {
  const colorClasses = {
    green: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400',
    red: 'bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400',
    blue: 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400',
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400">{label}</p>
          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
            {value}
            {online !== undefined && (
              <span className="text-sm font-normal text-green-600 dark:text-green-400 ml-2">
                ({online} online)
              </span>
            )}
          </p>
        </div>
        <div className={`w-12 h-12 rounded-lg flex items-center justify-center ${colorClasses[color]}`}>
          <StatIcon name={icon} className="w-6 h-6" />
        </div>
      </div>
    </div>
  );
};

// Empty State Component
interface EmptyStateProps {
  onAddDevice?: () => void;
}

const EmptyState = ({ onAddDevice }: EmptyStateProps) => (
  <div className="text-center py-16 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
    <svg
      className="mx-auto h-16 w-16 text-gray-400"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={1.5}
        d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
      />
    </svg>
    <h3 className="mt-4 text-lg font-medium text-gray-900 dark:text-gray-100">
      No devices found
    </h3>
    <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
      Get started by adding your first printer or agent.
    </p>
    <div className="mt-6">
      <button
        onClick={onAddDevice}
        className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
      >
        <PlusIcon className="w-5 h-5" />
        Add Device
      </button>
    </div>
  </div>
);

// Icons
const PlusIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
  </svg>
);

const SearchIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
    />
  </svg>
);

const TableIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M3 14h18m-9-4v8m-7-4h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2z" />
  </svg>
);

const GridIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
  </svg>
);

const AgentIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
    />
  </svg>
);

const PrinterIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
    />
  </svg>
);

const TrashIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
    />
  </svg>
);

const StatIcon = ({ name, className }: { name: string; className?: string }) => {
  const icons: Record<string, React.ReactNode> = {
    server: (
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
    ),
    printer: (
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
    ),
    signal: (
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
    ),
    'signal-off': (
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 5.636a9 9 0 010 12.728m0 0l-2.829-2.829m2.829 2.829L21 21M15.536 8.464a5 5 0 010 7.072m0 0l-2.829-2.829m-4.243 2.829a4.978 4.978 0 01-1.414-2.83m-1.414 5.658a9 9 0 01-2.167-9.238m7.824 2.167a1 1 0 111.414 1.414m-1.414-1.414L3 3" />
    ),
  };

  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      {icons[name] || icons.signal}
    </svg>
  );
};
