import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { PrintDriver } from '@/types';

const API = '/api/v1';
const authHeader = () => ({
  Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}`,
});

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

export const DriverManagement = () => {
  const [filterOS, setFilterOS] = useState<string>('all');
  const [filterMfg, setFilterMfg] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const { data: drivers = [], isLoading } = useQuery<PrintDriver[]>({
    queryKey: ['print-drivers'],
    queryFn: async () => {
      const res = await fetch(`${API}/drivers`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed to fetch drivers');
      const data = await res.json();
      return data.drivers || [];
    },
  });

  const manufacturers = [...new Set(drivers.map(d => d.manufacturer))].sort();
  const osOptions = [...new Set(drivers.map(d => d.os))].sort();

  const filtered = drivers.filter(d => {
    if (filterOS !== 'all' && d.os !== filterOS) return false;
    if (filterMfg !== 'all' && d.manufacturer !== filterMfg) return false;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      return d.name.toLowerCase().includes(q) || d.manufacturer.toLowerCase().includes(q) || (d.modelPattern || '').toLowerCase().includes(q);
    }
    return true;
  });

  const osIcons: Record<string, string> = {
    windows: 'W',
    macos: 'M',
    linux: 'L',
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Driver Management</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">Manage and distribute printer drivers across your fleet.</p>
        </div>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
          Upload Driver
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Total Drivers', value: drivers.length },
          { label: 'Windows', value: drivers.filter(d => d.os === 'windows').length },
          { label: 'macOS', value: drivers.filter(d => d.os === 'macos').length },
          { label: 'Universal', value: drivers.filter(d => d.isUniversal).length },
        ].map(stat => (
          <div key={stat.label} className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
            <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stat.value}</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">{stat.label}</p>
          </div>
        ))}
      </div>

      {/* Filters */}
      <div className="flex gap-4 items-center">
        <input
          type="text"
          placeholder="Search drivers..."
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 w-64"
        />
        <select
          value={filterOS}
          onChange={e => setFilterOS(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
        >
          <option value="all">All OS</option>
          {osOptions.map(os => <option key={os} value={os}>{os}</option>)}
        </select>
        <select
          value={filterMfg}
          onChange={e => setFilterMfg(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
        >
          <option value="all">All Manufacturers</option>
          {manufacturers.map(m => <option key={m} value={m}>{m}</option>)}
        </select>
      </div>

      {/* Drivers Table */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-gray-500">Loading drivers...</div>
        ) : filtered.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">
            {searchQuery || filterOS !== 'all' || filterMfg !== 'all'
              ? 'No drivers match your filters.'
              : 'No drivers uploaded yet. Upload a driver package to get started.'
            }
          </div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-50 dark:bg-gray-700/50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Driver</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">OS</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Version</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Size</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Status</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {filtered.map(driver => (
                <tr key={driver.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-4 py-3">
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100">{driver.name}</p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {driver.manufacturer} {driver.modelPattern && `\u00b7 ${driver.modelPattern}`}
                      </p>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-sm">
                      <span className="font-mono font-bold text-xs">{osIcons[driver.os] || driver.os.charAt(0).toUpperCase()}</span>
                      {driver.os} {driver.architecture}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-900 dark:text-gray-100">{driver.version}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">{driver.fileSizeBytes ? formatBytes(driver.fileSizeBytes) : '-'}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {driver.isLatest && (
                        <span className="text-xs px-2 py-0.5 bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300 rounded-full">Latest</span>
                      )}
                      {driver.isUniversal && (
                        <span className="text-xs px-2 py-0.5 bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300 rounded-full">Universal</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button className="text-sm text-blue-600 hover:text-blue-700 mr-3">Download</button>
                    <button className="text-sm text-red-600 hover:text-red-700">Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};
