import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { UserGroup, GroupMember } from '@/types';

const API = '/api/v1';
const authHeader = () => ({
  Authorization: `Bearer ${JSON.parse(localStorage.getItem('auth_tokens') || '{}').accessToken}`,
});

export const UserGroups = () => {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [form, setForm] = useState({ name: '', description: '', color: '#6366F1' });
  const [addMemberEmail, setAddMemberEmail] = useState('');

  const { data: groups = [], isLoading } = useQuery<UserGroup[]>({
    queryKey: ['user-groups'],
    queryFn: async () => {
      const res = await fetch(`${API}/groups`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed to fetch groups');
      const data = await res.json();
      return data.groups || [];
    },
  });

  const { data: members = [] } = useQuery<GroupMember[]>({
    queryKey: ['group-members', selectedGroup],
    enabled: !!selectedGroup,
    queryFn: async () => {
      const res = await fetch(`${API}/groups/${selectedGroup}/members`, { headers: authHeader() });
      if (!res.ok) throw new Error('Failed');
      const data = await res.json();
      return data.members || [];
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: typeof form) => {
      const res = await fetch(`${API}/groups`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify(data),
      });
      if (!res.ok) throw new Error('Failed');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-groups'] });
      setShowCreate(false);
      setForm({ name: '', description: '', color: '#6366F1' });
    },
  });

  const addMemberMutation = useMutation({
    mutationFn: async ({ groupId, email }: { groupId: string; email: string }) => {
      const res = await fetch(`${API}/groups/${groupId}/members`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify({ emails: [email] }),
      });
      if (!res.ok) throw new Error('Failed');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['group-members', selectedGroup] });
      queryClient.invalidateQueries({ queryKey: ['user-groups'] });
      setAddMemberEmail('');
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: async ({ groupId, userId }: { groupId: string; userId: string }) => {
      const res = await fetch(`${API}/groups/${groupId}/members`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify({ user_ids: [userId] }),
      });
      if (!res.ok) throw new Error('Failed');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['group-members', selectedGroup] });
      queryClient.invalidateQueries({ queryKey: ['user-groups'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (groupId: string) => {
      const res = await fetch(`${API}/groups/${groupId}`, {
        method: 'DELETE',
        headers: authHeader(),
      });
      if (!res.ok) throw new Error('Failed');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-groups'] });
      setSelectedGroup(null);
    },
  });

  const colorPresets = ['#6366F1', '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899', '#14B8A6'];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">User Groups</h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1">Organize users into groups for policy and quota management.</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          Create Group
        </button>
      </div>

      {/* Create Group Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Create User Group</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Group Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  placeholder="e.g., Marketing, Engineering"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <textarea
                  value={form.description}
                  onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  rows={2}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Color</label>
                <div className="flex gap-2">
                  {colorPresets.map(c => (
                    <button
                      key={c}
                      onClick={() => setForm(f => ({ ...f, color: c }))}
                      className={`w-8 h-8 rounded-full border-2 transition-transform ${form.color === c ? 'border-gray-900 dark:border-white scale-110' : 'border-transparent'}`}
                      style={{ backgroundColor: c }}
                    />
                  ))}
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg">Cancel</button>
              <button
                onClick={() => createMutation.mutate(form)}
                disabled={!form.name || createMutation.isPending}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                Create Group
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Groups List */}
        <div className="lg:col-span-1 space-y-3">
          {isLoading ? (
            <div className="bg-white dark:bg-gray-800 rounded-xl p-8 text-center text-gray-500 border border-gray-200 dark:border-gray-700">
              Loading...
            </div>
          ) : groups.length === 0 ? (
            <div className="bg-white dark:bg-gray-800 rounded-xl p-8 text-center text-gray-500 dark:text-gray-400 border border-gray-200 dark:border-gray-700">
              No groups created yet.
            </div>
          ) : (
            groups.map(group => (
              <button
                key={group.id}
                onClick={() => setSelectedGroup(group.id)}
                className={`w-full text-left bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border transition-colors ${
                  selectedGroup === group.id
                    ? 'border-blue-500 ring-1 ring-blue-500'
                    : 'border-gray-200 dark:border-gray-700 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center gap-3">
                  <div className="w-4 h-4 rounded-full" style={{ backgroundColor: group.color }} />
                  <div className="flex-1">
                    <p className="font-medium text-gray-900 dark:text-gray-100">{group.name}</p>
                    {group.description && (
                      <p className="text-sm text-gray-500 dark:text-gray-400 truncate">{group.description}</p>
                    )}
                  </div>
                  <span className="text-sm text-gray-500">{group.memberCount || 0} members</span>
                </div>
              </button>
            ))
          )}
        </div>

        {/* Group Detail */}
        <div className="lg:col-span-2">
          {selectedGroup ? (
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-4 h-4 rounded-full" style={{ backgroundColor: groups.find(g => g.id === selectedGroup)?.color }} />
                  <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {groups.find(g => g.id === selectedGroup)?.name}
                  </h2>
                </div>
                <button
                  onClick={() => deleteMutation.mutate(selectedGroup)}
                  className="text-sm text-red-600 hover:text-red-700"
                >
                  Delete Group
                </button>
              </div>

              {/* Add Member */}
              <div className="p-4 border-b border-gray-200 dark:border-gray-700">
                <div className="flex gap-2">
                  <input
                    type="email"
                    placeholder="Add member by email..."
                    value={addMemberEmail}
                    onChange={e => setAddMemberEmail(e.target.value)}
                    className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                  <button
                    onClick={() => addMemberMutation.mutate({ groupId: selectedGroup, email: addMemberEmail })}
                    disabled={!addMemberEmail || addMemberMutation.isPending}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                  >
                    Add
                  </button>
                </div>
              </div>

              {/* Members List */}
              <div className="divide-y divide-gray-200 dark:divide-gray-700">
                {members.length === 0 ? (
                  <div className="p-8 text-center text-gray-500 dark:text-gray-400">
                    No members in this group. Add users by email above.
                  </div>
                ) : (
                  members.map(member => (
                    <div key={member.userId} className="p-4 flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-gray-200 dark:bg-gray-700 rounded-full flex items-center justify-center text-sm font-semibold text-gray-600 dark:text-gray-300">
                          {(member.userName || member.userEmail || '?').charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <p className="font-medium text-gray-900 dark:text-gray-100">{member.userName || 'Unknown'}</p>
                          <p className="text-sm text-gray-500">{member.userEmail || ''}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <span className="text-sm text-gray-400">Added {new Date(member.addedAt).toLocaleDateString()}</span>
                        <button
                          onClick={() => removeMemberMutation.mutate({ groupId: selectedGroup, userId: member.userId })}
                          className="text-sm text-red-600 hover:text-red-700"
                        >
                          Remove
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          ) : (
            <div className="bg-white dark:bg-gray-800 rounded-xl p-8 text-center text-gray-500 dark:text-gray-400 border border-gray-200 dark:border-gray-700">
              Select a group to view and manage its members.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
