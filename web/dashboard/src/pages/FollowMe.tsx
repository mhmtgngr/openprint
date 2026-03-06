import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { FollowMePool, FollowMeJob } from '@/types';

const API = '/api/v1';

const authHeader = () => ({
  Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}`,
});

export const FollowMe = () => {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<'pools' | 'jobs'>('pools');
  const [showCreatePool, setShowCreatePool] = useState(false);
  const [poolForm, setPoolForm] = useState({ name: '', description: '', location: '' });

  const { data: pools = [] } = useQuery<FollowMePool[]>({
    queryKey: ['follow-me-pools'],
    queryFn: async () => {
      const res = await fetch(`${API}/follow-me/pools`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed to fetch pools');
      const data = await res.json();
      return data.pools || [];
    },
  });

  const { data: pendingJobs = [] } = useQuery<FollowMeJob[]>({
    queryKey: ['follow-me-jobs'],
    queryFn: async () => {
      const res = await fetch(`${API}/follow-me/jobs/pending`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed to fetch jobs');
      const data = await res.json();
      return data.jobs || [];
    },
  });

  const createPoolMutation = useMutation({
    mutationFn: async (data: typeof poolForm) => {
      const res = await fetch(`${API}/follow-me/pools`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify(data),
      });
      if (!res.ok) throw new Error('Failed to create pool');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['follow-me-pools'] });
      setShowCreatePool(false);
      setPoolForm({ name: '', description: '', location: '' });
    },
  });

  const cancelJobMutation = useMutation({
    mutationFn: async (jobId: string) => {
      const res = await fetch(`${API}/follow-me/jobs/${jobId}`, {
        method: 'DELETE',
        headers: authHeader(),
      });
      if (!res.ok) throw new Error('Failed to cancel job');
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['follow-me-jobs'] }),
  });

  const statusColors: Record<string, string> = {
    waiting: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300',
    released: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300',
    expired: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300',
    cancelled: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Follow-Me Printing</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">
            Submit jobs to a printer pool and release them at any printer in the pool.
          </p>
        </div>
        <button
          onClick={() => setShowCreatePool(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Create Pool
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-1 w-fit">
        {(['pools', 'jobs'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 shadow-sm'
                : 'text-gray-600 dark:text-gray-400 hover:text-gray-900'
            }`}
          >
            {tab === 'pools' ? 'Printer Pools' : `Pending Jobs (${pendingJobs.length})`}
          </button>
        ))}
      </div>

      {/* Create Pool Modal */}
      {showCreatePool && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Create Printer Pool</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Pool Name</label>
                <input
                  type="text"
                  value={poolForm.name}
                  onChange={e => setPoolForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="Building A - Floor 2"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Location</label>
                <input
                  type="text"
                  value={poolForm.location}
                  onChange={e => setPoolForm(f => ({ ...f, location: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="Floor 2, East Wing"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <textarea
                  value={poolForm.description}
                  onChange={e => setPoolForm(f => ({ ...f, description: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  rows={2}
                />
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowCreatePool(false)} className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg">Cancel</button>
              <button
                onClick={() => createPoolMutation.mutate(poolForm)}
                disabled={!poolForm.name || createPoolMutation.isPending}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                Create Pool
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Pools Tab */}
      {activeTab === 'pools' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {pools.length === 0 ? (
            <div className="col-span-full bg-white dark:bg-gray-800 rounded-xl p-8 text-center text-gray-500 dark:text-gray-400 border border-gray-200 dark:border-gray-700">
              No printer pools configured. Create a pool to enable Follow-Me printing.
            </div>
          ) : (
            pools.map(pool => (
              <div key={pool.id} className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-6">
                <div className="flex items-start justify-between mb-4">
                  <div>
                    <h3 className="font-semibold text-gray-900 dark:text-gray-100">{pool.name}</h3>
                    {pool.location && (
                      <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{pool.location}</p>
                    )}
                  </div>
                  <span className={`text-xs px-2 py-1 rounded-full ${pool.isActive ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-gray-100 text-gray-500'}`}>
                    {pool.isActive ? 'Active' : 'Inactive'}
                  </span>
                </div>
                {pool.description && (
                  <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">{pool.description}</p>
                )}
                <div className="flex gap-4 text-sm">
                  <div>
                    <span className="font-semibold text-gray-900 dark:text-gray-100">{pool.printerCount || 0}</span>
                    <span className="text-gray-500 dark:text-gray-400 ml-1">printers</span>
                  </div>
                  <div>
                    <span className="font-semibold text-gray-900 dark:text-gray-100">{pool.pendingJobs || 0}</span>
                    <span className="text-gray-500 dark:text-gray-400 ml-1">pending</span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {/* Jobs Tab */}
      {activeTab === 'jobs' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          {pendingJobs.length === 0 ? (
            <div className="p-8 text-center text-gray-500 dark:text-gray-400">
              No pending Follow-Me jobs. Submit a job to a printer pool to get started.
            </div>
          ) : (
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {pendingJobs.map(job => (
                <div key={job.id} className="p-4 flex items-center justify-between">
                  <div className="flex-1">
                    <p className="font-medium text-gray-900 dark:text-gray-100">{job.documentName}</p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {job.pageCount} pages &middot; {job.copies} copies
                      {job.color && ' \u00b7 Color'}
                      {job.duplex && ' \u00b7 Duplex'}
                    </p>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className={`text-xs px-2 py-1 rounded-full ${statusColors[job.status] || ''}`}>
                      {job.status}
                    </span>
                    <span className="text-sm text-gray-500">
                      Expires {new Date(job.expiresAt).toLocaleTimeString()}
                    </span>
                    {job.status === 'waiting' && (
                      <button
                        onClick={() => cancelJobMutation.mutate(job.id)}
                        className="text-sm text-red-600 hover:text-red-700"
                      >
                        Cancel
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
