import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { PlusIcon, TrashIcon, EditIcon } from '@/components/icons';

interface AgentGroup {
  id: string;
  name: string;
  description: string;
  type: string;
  location: string;
  tags: string[];
  agent_count?: number;
  created_at: string;
}

interface CreateGroupRequest {
  name: string;
  description: string;
  type: string;
  location: string;
  tags: string[];
}

export const AgentGroups = () => {
  const queryClient = useQueryClient();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingGroup, setEditingGroup] = useState<AgentGroup | null>(null);
  const [newGroup, setNewGroup] = useState<CreateGroupRequest>({
    name: '',
    description: '',
    type: 'custom',
    location: '',
    tags: [],
  });
  const [newTag, setNewTag] = useState('');

  // Fetch agent groups
  const { data: groupsData, isLoading } = useQuery({
    queryKey: ['agent-groups'],
    queryFn: async () => {
      const res = await fetch('/api/v1/agent-groups');
      if (!res.ok) throw new Error('Failed to fetch agent groups');
      return res.json();
    },
  });

  const groups = groupsData?.groups || [];

  // Create group mutation
  const createGroupMutation = useMutation({
    mutationFn: async (group: CreateGroupRequest) => {
      const res = await fetch('/api/v1/agent-groups', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(group),
      });
      if (!res.ok) throw new Error('Failed to create group');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-groups'] });
      setShowCreateForm(false);
      setNewGroup({ name: '', description: '', type: 'custom', location: '', tags: [] });
    },
  });

  // Update group mutation
  const updateGroupMutation = useMutation({
    mutationFn: async ({ id, ...group }: Partial<AgentGroup> & { id: string }) => {
      const res = await fetch(`/api/v1/agent-groups/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(group),
      });
      if (!res.ok) throw new Error('Failed to update group');
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-groups'] });
      setEditingGroup(null);
    },
  });

  // Delete group mutation
  const deleteGroupMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`/api/v1/agent-groups/${id}`, {
        method: 'DELETE',
      });
      if (!res.ok) throw new Error('Failed to delete group');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agent-groups'] });
    },
  });

  const handleCreateGroup = () => {
    if (!newGroup.name.trim()) return;
    createGroupMutation.mutate(newGroup);
  };

  const handleUpdateGroup = () => {
    if (!editingGroup || !editingGroup.name.trim()) return;
    updateGroupMutation.mutate(editingGroup);
  };

  const handleDeleteGroup = (id: string) => {
    if (confirm('Are you sure you want to delete this group?')) {
      deleteGroupMutation.mutate(id);
    }
  };

  const addTag = () => {
    if (newTag.trim() && !newGroup.tags.includes(newTag.trim())) {
      setNewGroup({ ...newGroup, tags: [...newGroup.tags, newTag.trim()] });
      setNewTag('');
    }
  };

  const removeTag = (tag: string) => {
    setNewGroup({ ...newGroup, tags: newGroup.tags.filter(t => t !== tag) });
  };

  const groupedByType = groups.reduce((acc: Record<string, AgentGroup[]>, group: AgentGroup) => {
    const type = group.type || 'custom';
    if (!acc[type]) acc[type] = [];
    acc[type].push(group);
    return acc;
  }, {} as Record<string, AgentGroup[]>);

  const typeColors: Record<string, string> = {
    location: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400 border-purple-200 dark:border-purple-800',
    department: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 border-blue-200 dark:border-blue-800',
    custom: 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-700',
    auto: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 border-green-200 dark:border-green-800',
  };

  const typeLabels: Record<string, string> = {
    location: 'Location',
    department: 'Department',
    custom: 'Custom',
    auto: 'Auto',
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Agent Groups</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Organize agents by location, department, or custom criteria
          </p>
        </div>
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
        >
          <PlusIcon className="w-4 h-4" />
          Create Group
        </button>
      </div>

      {/* Create Group Form */}
      {showCreateForm && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
          <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-4">Create New Group</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Group Name *
              </label>
              <input
                type="text"
                value={newGroup.name}
                onChange={(e) => setNewGroup({ ...newGroup, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
                placeholder="e.g., Floor 3 - Engineering"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Type
                </label>
                <select
                  value={newGroup.type}
                  onChange={(e) => setNewGroup({ ...newGroup, type: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
                >
                  <option value="custom">Custom</option>
                  <option value="location">Location</option>
                  <option value="department">Department</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Location
                </label>
                <input
                  type="text"
                  value={newGroup.location}
                  onChange={(e) => setNewGroup({ ...newGroup, location: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
                  placeholder="e.g., Building A, Floor 3"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Description
              </label>
              <textarea
                value={newGroup.description}
                onChange={(e) => setNewGroup({ ...newGroup, description: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
                rows={2}
                placeholder="Optional description of this group"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Tags
              </label>
              <div className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={newTag}
                  onChange={(e) => setNewTag(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), addTag())}
                  className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
                  placeholder="Add a tag"
                />
                <button
                  onClick={addTag}
                  className="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium"
                >
                  Add
                </button>
              </div>
              {newGroup.tags.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {newGroup.tags.map((tag) => (
                    <span
                      key={tag}
                      className="inline-flex items-center gap-1 px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded text-sm"
                    >
                      {tag}
                      <button
                        onClick={() => removeTag(tag)}
                        className="hover:text-blue-900 dark:hover:text-blue-300"
                      >
                        ×
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </div>

            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setShowCreateForm(false)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateGroup}
                disabled={createGroupMutation.isPending || !newGroup.name.trim()}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg text-sm font-medium"
              >
                {createGroupMutation.isPending ? 'Creating...' : 'Create Group'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Group Form */}
      {editingGroup && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
          <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-4">Edit Group</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Group Name *
              </label>
              <input
                type="text"
                value={editingGroup.name}
                onChange={(e) => setEditingGroup({ ...editingGroup, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
              />
            </div>

            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setEditingGroup(null)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium"
              >
                Cancel
              </button>
              <button
                onClick={handleUpdateGroup}
                disabled={updateGroupMutation.isPending || !editingGroup.name.trim()}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg text-sm font-medium"
              >
                {updateGroupMutation.isPending ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Groups List */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500 dark:text-gray-400">
          Loading agent groups...
        </div>
      ) : groups.length === 0 ? (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-12 text-center">
          <p className="text-gray-500 dark:text-gray-400 mb-4">No agent groups yet</p>
          <button
            onClick={() => setShowCreateForm(true)}
            className="text-blue-600 dark:text-blue-400 hover:underline"
          >
            Create your first group
          </button>
        </div>
      ) : (
        <div className="space-y-6">
          {Object.entries(groupedByType).map(([type, typeGroups]) => (
            <div key={type}>
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
                {typeLabels[type] || type} Groups
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {(typeGroups as AgentGroup[]).map((group) => (
                  <div
                    key={group.id}
                    className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-shadow"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex-1">
                        <h4 className="font-medium text-gray-900 dark:text-gray-100">{group.name}</h4>
                        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{group.description || 'No description'}</p>
                      </div>
                      <div className="flex gap-1">
                        <button
                          onClick={() => setEditingGroup(group)}
                          className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                        >
                          <EditIcon className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDeleteGroup(group.id)}
                          className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                        >
                          <TrashIcon className="w-4 h-4" />
                        </button>
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-2 mb-3">
                      <span className={`text-xs px-2 py-1 rounded border ${typeColors[type] || typeColors.custom}`}>
                        {typeLabels[type] || type}
                      </span>
                      {group.location && (
                        <span className="text-xs px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400">
                          {group.location}
                        </span>
                      )}
                    </div>

                    {group.tags && group.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1">
                        {group.tags.map((tag: string) => (
                          <span
                            key={tag}
                            className="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}

                    <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between text-sm">
                      <span className="text-gray-500 dark:text-gray-400">
                        {group.agent_count || 0} agents
                      </span>
                      <a
                        href={`/agents?group=${group.id}`}
                        className="text-blue-600 dark:text-blue-400 hover:underline"
                      >
                        View Agents →
                      </a>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
