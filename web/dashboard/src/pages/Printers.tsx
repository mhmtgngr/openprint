import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { printersApi } from '@/services/api';
import { PrinterCard, PrinterIcon } from '@/components/PrinterCard';
import { PlusIcon, SearchIcon, FilterIcon } from '@/components/icons';

export const Printers = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<'all' | 'online' | 'offline'>('all');
  const [isAdding, setIsAdding] = useState(false);
  const [newPrinterName, setNewPrinterName] = useState('');
  const [newPrinterAgent, setNewPrinterAgent] = useState('');

  const { data: printers, isLoading } = useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; agentId: string }) =>
      fetch('/api/v1/printers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: data.name,
          agentId: data.agentId,
          type: 'network',
          isActive: true,
          isOnline: false,
          capabilities: {
            supportsColor: true,
            supportsDuplex: true,
            supportedPaperSizes: ['A4', 'Letter'],
            resolution: '600x600',
          },
        }),
      }).then(r => r.json()),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['printers'] });
      setIsAdding(false);
      setNewPrinterName('');
      setNewPrinterAgent('');
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      printersApi.update(id, { isActive: !isActive }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['printers'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => printersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['printers'] });
    },
  });

  const handleDeletePrinter = (id: string, name: string) => {
    if (confirm(`Are you sure you want to delete printer "${name}"?`)) {
      deleteMutation.mutate(id);
    }
  };

  const filteredPrinters = printers?.filter((printer) => {
    const matchesSearch = printer.name.toLowerCase().includes(search.toLowerCase());
    const matchesFilter =
      filter === 'all' ||
      (filter === 'online' && printer.isOnline) ||
      (filter === 'offline' && !printer.isOnline);
    return matchesSearch && matchesFilter;
  });

  const onlineCount = printers?.filter((p) => p.isOnline).length || 0;
  const activeCount = printers?.filter((p) => p.isActive).length || 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Printers</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage your organization's printing devices
          </p>
        </div>
        <button onClick={() => setIsAdding(true)} className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium">
          <PlusIcon className="w-5 h-5" />
          Add Printer
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Total Printers</p>
          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {printers?.length || 0}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Online</p>
          <p className="text-2xl font-bold text-green-600 dark:text-green-400">{onlineCount}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Active</p>
          <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">{activeCount}</p>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Search printers"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          />
        </div>
        <div className="flex items-center gap-2">
          <FilterIcon className="w-5 h-5 text-gray-400" />
          <div className="flex gap-2">
            <button
              onClick={() => setFilter('all')}
              className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                filter === 'all'
                  ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
              }`}
            >
              All
            </button>
            <button
              onClick={() => setFilter('online')}
              className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                filter === 'online'
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
              }`}
            >
              Online
            </button>
            <button
              onClick={() => setFilter('offline')}
              className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                filter === 'offline'
                  ? 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
              }`}
            >
              Offline
            </button>
          </div>
        </div>
      </div>

      {/* Printer Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[...Array(6)].map((_, i) => (
            <div
              key={i}
              className="bg-gray-100 dark:bg-gray-800 rounded-lg h-48 animate-pulse"
            />
          ))}
        </div>
      ) : filteredPrinters && filteredPrinters.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredPrinters.map((printer) => (
            <PrinterCard
              key={printer.id}
              printer={printer}
              onToggle={() => toggleMutation.mutate({ id: printer.id, isActive: printer.isActive })}
              onDelete={() => handleDeletePrinter(printer.id, printer.name)}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <PrinterIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            No printers found
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {search || filter !== 'all'
              ? 'Try adjusting your search or filter'
              : !printers || printers.length === 0
                ? 'Add your first printer to get started.'
                : 'Install the OpenPrint Agent on your network to add printers.'}
          </p>
        </div>
      )}

      {/* Agent Installation Notice */}
      {!search && filter === 'all' && (!printers || printers.length === 0) && (
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-6">
          <div className="flex items-start gap-4">
            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
              <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Install the OpenPrint Agent
              </h3>
              <p className="text-gray-600 dark:text-gray-400 mt-1">
                Download and install the agent on a computer connected to your printers. The agent
                will automatically discover and register your printers.
              </p>
              <div className="mt-4 flex gap-3">
                <a
                  href="/downloads/openprint-agent-windows.msi"
                  download
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium inline-block text-center"
                >
                  Download for Windows
                </a>
                <a
                  href="/downloads/openprint-agent-macos.pkg"
                  download
                  className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors font-medium inline-block text-center"
                >
                  Download for macOS
                </a>
                <a
                  href="/downloads/openprint-agent-linux.deb"
                  download
                  className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors font-medium inline-block text-center"
                >
                  Download for Linux
                </a>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Add Printer Modal */}
      {isAdding && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full mx-4">
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Add Printer
              </h3>
            </div>
            <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate({ name: newPrinterName, agentId: newPrinterAgent }); }} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Printer Name
                </label>
                <input
                  type="text"
                  name="name"
                  value={newPrinterName}
                  onChange={(e) => setNewPrinterName(e.target.value)}
                  required
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="e.g., HP LaserJet Pro"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Agent
                </label>
                <select
                  name="agentId"
                  value={newPrinterAgent}
                  onChange={(e) => setNewPrinterAgent(e.target.value)}
                  required
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                >
                  <option value="">Select an agent...</option>
                  <option value="agent-1">WORKSTATION-001</option>
                  <option value="agent-2">WORKSTATION-002</option>
                </select>
              </div>
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => {
                    setIsAdding(false);
                    setNewPrinterName('');
                    setNewPrinterAgent('');
                  }}
                  className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {createMutation.isPending ? 'Adding...' : 'Add Printer'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
