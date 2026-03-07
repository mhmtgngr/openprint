import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { GuestToken } from '@/types';

const API = '/api/v1';

const fetchTokens = async (): Promise<GuestToken[]> => {
  const res = await fetch(`${API}/guest/tokens`, {
    headers: { Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}` },
  });
  if (!res.ok) throw new Error('Failed to fetch tokens');
  const data = await res.json();
  return data.tokens || [];
};

export const GuestPrinting = () => {
  const queryClient = useQueryClient();
  const { data: tokens = [], isLoading } = useQuery({ queryKey: ['guest-tokens'], queryFn: fetchTokens });
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: '', email: '', maxPages: 10, maxJobs: 5, colorAllowed: false, expiresInHours: 24 });

  const createMutation = useMutation({
    mutationFn: async (data: typeof form) => {
      const res = await fetch(`${API}/guest/tokens`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}`,
        },
        body: JSON.stringify({
          name: data.name,
          email: data.email,
          max_pages: data.maxPages,
          max_jobs: data.maxJobs,
          color_allowed: data.colorAllowed,
          expires_in_hours: data.expiresInHours,
        }),
      });
      if (!res.ok) throw new Error('Failed to create token');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['guest-tokens'] });
      setShowCreate(false);
      setForm({ name: '', email: '', maxPages: 10, maxJobs: 5, colorAllowed: false, expiresInHours: 24 });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`${API}/guest/tokens/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}` },
      });
      if (!res.ok) throw new Error('Failed to revoke token');
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['guest-tokens'] }),
  });

  const activeTokens = tokens.filter(t => t.isActive && new Date(t.expiresAt) > new Date());
  const expiredTokens = tokens.filter(t => !t.isActive || new Date(t.expiresAt) <= new Date());

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Guest Printing</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">Create temporary print access tokens for visitors and guests.</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Create Guest Token
        </button>
      </div>

      {/* Create Token Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Create Guest Token</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Guest Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="John Doe"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email (optional)</label>
                <input
                  type="email"
                  value={form.email}
                  onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="guest@example.com"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Pages</label>
                  <input
                    type="number"
                    value={form.maxPages}
                    onChange={e => setForm(f => ({ ...f, maxPages: parseInt(e.target.value) || 0 }))}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Jobs</label>
                  <input
                    type="number"
                    value={form.maxJobs}
                    onChange={e => setForm(f => ({ ...f, maxJobs: parseInt(e.target.value) || 0 }))}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Expires In (hours)</label>
                <select
                  value={form.expiresInHours}
                  onChange={e => setForm(f => ({ ...f, expiresInHours: parseInt(e.target.value) }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <option value={1}>1 hour</option>
                  <option value={4}>4 hours</option>
                  <option value={8}>8 hours (business day)</option>
                  <option value={24}>24 hours</option>
                  <option value={72}>3 days</option>
                  <option value={168}>1 week</option>
                </select>
              </div>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={form.colorAllowed}
                  onChange={e => setForm(f => ({ ...f, colorAllowed: e.target.checked }))}
                  className="rounded border-gray-300"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Allow color printing</span>
              </label>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
              >
                Cancel
              </button>
              <button
                onClick={() => createMutation.mutate(form)}
                disabled={createMutation.isPending}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {createMutation.isPending ? 'Creating...' : 'Create Token'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Active Tokens */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Active Guest Tokens ({activeTokens.length})</h2>
        </div>
        {isLoading ? (
          <div className="p-8 text-center text-gray-500">Loading...</div>
        ) : activeTokens.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">
            No active guest tokens. Create one to allow visitors to print.
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {activeTokens.map(token => (
              <div key={token.id} className="p-4 flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 rounded-full flex items-center justify-center font-semibold">
                      {(token.name || 'G').charAt(0).toUpperCase()}
                    </div>
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100">{token.name || 'Anonymous Guest'}</p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {token.email || 'No email'} &middot; Token: <code className="bg-gray-100 dark:bg-gray-700 px-1 rounded">{token.token?.slice(0, 8)}...</code>
                      </p>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-6 text-sm">
                  <div className="text-center">
                    <p className="font-semibold text-gray-900 dark:text-gray-100">{token.pagesUsed}/{token.maxPages}</p>
                    <p className="text-gray-500 dark:text-gray-400">Pages</p>
                  </div>
                  <div className="text-center">
                    <p className="font-semibold text-gray-900 dark:text-gray-100">{token.jobsUsed}/{token.maxJobs}</p>
                    <p className="text-gray-500 dark:text-gray-400">Jobs</p>
                  </div>
                  <div className="text-center">
                    <p className="text-gray-600 dark:text-gray-300">
                      Expires {new Date(token.expiresAt).toLocaleDateString()}
                    </p>
                  </div>
                  <button
                    onClick={() => revokeMutation.mutate(token.id)}
                    className="px-3 py-1 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg text-sm"
                  >
                    Revoke
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Expired Tokens */}
      {expiredTokens.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-500 dark:text-gray-400">Expired/Revoked Tokens ({expiredTokens.length})</h2>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {expiredTokens.slice(0, 10).map(token => (
              <div key={token.id} className="p-4 flex items-center justify-between opacity-60">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-gray-100 dark:bg-gray-700 text-gray-400 rounded-full flex items-center justify-center font-semibold">
                    {(token.name || 'G').charAt(0).toUpperCase()}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900 dark:text-gray-100">{token.name || 'Anonymous Guest'}</p>
                    <p className="text-sm text-gray-500">{token.pagesUsed} pages, {token.jobsUsed} jobs used</p>
                  </div>
                </div>
                <span className="text-sm text-gray-400">{new Date(token.expiresAt).toLocaleDateString()}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
