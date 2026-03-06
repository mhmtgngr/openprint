import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { PrinterSupply, MaintenanceTask } from '@/types';

const API = '/api/v1';
const authHeader = () => ({
  Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}`,
});

export const SupplyManagement = () => {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<'supplies' | 'maintenance'>('supplies');
  const [showAddMaintenance, setShowAddMaintenance] = useState(false);
  const [maintenanceForm, setMaintenanceForm] = useState({
    printerId: '', maintenanceType: 'cleaning', description: '', scheduledAt: '', assignedTo: '',
  });

  const { data: lowSupplies = [] } = useQuery<(PrinterSupply & { printerName?: string })[]>({
    queryKey: ['low-supplies'],
    queryFn: async () => {
      const res = await fetch(`${API}/supplies/alerts`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed');
      const data = await res.json();
      return data.alerts || [];
    },
  });

  const { data: maintenanceTasks = [] } = useQuery<MaintenanceTask[]>({
    queryKey: ['maintenance-tasks'],
    queryFn: async () => {
      const res = await fetch(`${API}/maintenance`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed');
      const data = await res.json();
      return data.tasks || [];
    },
  });

  const createMaintenanceMutation = useMutation({
    mutationFn: async (data: typeof maintenanceForm) => {
      const res = await fetch(`${API}/maintenance`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify({
          printer_id: data.printerId,
          maintenance_type: data.maintenanceType,
          description: data.description,
          scheduled_at: data.scheduledAt,
          assigned_to: data.assignedTo,
        }),
      });
      if (!res.ok) throw new Error('Failed');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['maintenance-tasks'] });
      setShowAddMaintenance(false);
    },
  });

  const completeMutation = useMutation({
    mutationFn: async (taskId: string) => {
      const res = await fetch(`${API}/maintenance/${taskId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify({ status: 'completed' }),
      });
      if (!res.ok) throw new Error('Failed');
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['maintenance-tasks'] }),
  });

  const getSupplyColor = (level: number) => {
    if (level <= 5) return 'bg-red-500';
    if (level <= 15) return 'bg-orange-500';
    if (level <= 30) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  const maintenanceStatusColors: Record<string, string> = {
    scheduled: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
    in_progress: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300',
    completed: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
    overdue: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Supply & Maintenance</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">Monitor supply levels and schedule printer maintenance.</p>
        </div>
        <button
          onClick={() => setShowAddMaintenance(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Schedule Maintenance
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Low Supply Alerts', value: lowSupplies.length, color: 'text-red-600', bg: 'bg-red-100 dark:bg-red-900/30' },
          { label: 'Scheduled Tasks', value: maintenanceTasks.filter(t => t.status === 'scheduled').length, color: 'text-blue-600', bg: 'bg-blue-100 dark:bg-blue-900/30' },
          { label: 'In Progress', value: maintenanceTasks.filter(t => t.status === 'in_progress').length, color: 'text-yellow-600', bg: 'bg-yellow-100 dark:bg-yellow-900/30' },
          { label: 'Overdue', value: maintenanceTasks.filter(t => t.status === 'overdue').length, color: 'text-orange-600', bg: 'bg-orange-100 dark:bg-orange-900/30' },
        ].map(stat => (
          <div key={stat.label} className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
            <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">{stat.label}</p>
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-1 w-fit">
        {(['supplies', 'maintenance'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 shadow-sm'
                : 'text-gray-600 dark:text-gray-400'
            }`}
          >
            {tab === 'supplies' ? 'Supply Alerts' : 'Maintenance Schedule'}
          </button>
        ))}
      </div>

      {/* Add Maintenance Modal */}
      {showAddMaintenance && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Schedule Maintenance</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type</label>
                <select
                  value={maintenanceForm.maintenanceType}
                  onChange={e => setMaintenanceForm(f => ({ ...f, maintenanceType: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <option value="cleaning">Cleaning</option>
                  <option value="toner_replacement">Toner Replacement</option>
                  <option value="drum_replacement">Drum Replacement</option>
                  <option value="paper_jam_fix">Paper Jam Fix</option>
                  <option value="firmware_update">Firmware Update</option>
                  <option value="calibration">Calibration</option>
                  <option value="inspection">General Inspection</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <textarea
                  value={maintenanceForm.description}
                  onChange={e => setMaintenanceForm(f => ({ ...f, description: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  rows={2}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Scheduled Date</label>
                <input
                  type="datetime-local"
                  value={maintenanceForm.scheduledAt}
                  onChange={e => setMaintenanceForm(f => ({ ...f, scheduledAt: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Assigned To</label>
                <input
                  type="text"
                  value={maintenanceForm.assignedTo}
                  onChange={e => setMaintenanceForm(f => ({ ...f, assignedTo: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="Technician name"
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowAddMaintenance(false)} className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg">Cancel</button>
              <button
                onClick={() => createMaintenanceMutation.mutate(maintenanceForm)}
                disabled={createMaintenanceMutation.isPending}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                Schedule
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Supplies Tab */}
      {activeTab === 'supplies' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          {lowSupplies.length === 0 ? (
            <div className="p-8 text-center text-gray-500 dark:text-gray-400">
              All printer supplies are at healthy levels.
            </div>
          ) : (
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {lowSupplies.map(supply => (
                <div key={supply.id} className="p-4 flex items-center gap-4">
                  <div className={`w-3 h-3 rounded-full ${getSupplyColor(supply.levelPercent)}`} />
                  <div className="flex-1">
                    <p className="font-medium text-gray-900 dark:text-gray-100">
                      {supply.printerName || 'Unknown Printer'} - {supply.name}
                    </p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {supply.supplyType} &middot; {supply.partNumber || 'No part number'}
                      {supply.estimatedPagesRemaining != null && ` \u00b7 ~${supply.estimatedPagesRemaining} pages remaining`}
                    </p>
                  </div>
                  <div className="w-32">
                    <div className="flex items-center justify-between text-sm mb-1">
                      <span className="text-gray-500">{supply.levelPercent}%</span>
                    </div>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div className={`${getSupplyColor(supply.levelPercent)} h-2 rounded-full transition-all`} style={{ width: `${supply.levelPercent}%` }} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Maintenance Tab */}
      {activeTab === 'maintenance' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          {maintenanceTasks.length === 0 ? (
            <div className="p-8 text-center text-gray-500 dark:text-gray-400">
              No maintenance tasks scheduled.
            </div>
          ) : (
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {maintenanceTasks.map(task => (
                <div key={task.id} className="p-4 flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-gray-900 dark:text-gray-100">{task.maintenanceType.replace(/_/g, ' ')}</p>
                      <span className={`text-xs px-2 py-0.5 rounded-full ${maintenanceStatusColors[task.status] || ''}`}>
                        {task.status.replace(/_/g, ' ')}
                      </span>
                    </div>
                    <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                      {task.printerName || 'Printer'} &middot; {task.assignedTo || 'Unassigned'}
                      {task.description && ` \u00b7 ${task.description}`}
                    </p>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-gray-500">
                      {new Date(task.scheduledAt).toLocaleDateString()}
                    </span>
                    {task.status === 'scheduled' && (
                      <button
                        onClick={() => completeMutation.mutate(task.id)}
                        className="text-sm text-green-600 hover:text-green-700"
                      >
                        Complete
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
