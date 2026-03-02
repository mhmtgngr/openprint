import { useState } from 'react';

type PinType = '4-digit' | '6-digit' | 'custom';

interface SecurePrintConfig {
  enabled: boolean;
  requirePin: boolean;
  pinType: PinType;
  customPinLength: number;
  requireCardRelease: boolean;
  requireAuthAtRelease: boolean;
  pinExpiration: number; // minutes
  maxQueueSize: number;
}

interface QueuedJob {
  id: string;
  documentName: string;
  user: string;
  submittedAt: string;
  status: 'waiting' | 'released' | 'expired';
  pageCount: number;
}

export const SecurePrint = () => {
  const [activeTab, setActiveTab] = useState<'overview' | 'settings' | 'queue'>('overview');
  const [config, setConfig] = useState<SecurePrintConfig>({
    enabled: true,
    requirePin: true,
    pinType: '4-digit',
    customPinLength: 6,
    requireCardRelease: false,
    requireAuthAtRelease: true,
    pinExpiration: 240,
    maxQueueSize: 50,
  });
  const [selectedJobs, setSelectedJobs] = useState<Set<string>>(new Set());

  // Mock queued jobs
  const queuedJobs: QueuedJob[] = [
    {
      id: '1',
      documentName: 'Quarterly Report.pdf',
      user: 'john.doe@example.com',
      submittedAt: '2024-01-15T09:30:00Z',
      status: 'waiting',
      pageCount: 15,
    },
    {
      id: '2',
      documentName: 'Presentation.pptx',
      user: 'jane.smith@example.com',
      submittedAt: '2024-01-15T09:45:00Z',
      status: 'waiting',
      pageCount: 22,
    },
    {
      id: '3',
      documentName: 'Contract Draft.docx',
      user: 'bob.wilson@example.com',
      submittedAt: '2024-01-15T08:00:00Z',
      status: 'expired',
      pageCount: 8,
    },
    {
      id: '4',
      documentName: 'Invoice #12345.pdf',
      user: 'alice.johnson@example.com',
      submittedAt: '2024-01-15T10:00:00Z',
      status: 'waiting',
      pageCount: 2,
    },
  ];

  const handleReleaseSelected = () => {
    console.log('Releasing jobs:', Array.from(selectedJobs));
    setSelectedJobs(new Set());
  };

  const handleClearExpired = () => {
    console.log('Clearing expired jobs');
  };

  const handleJobSelect = (jobId: string) => {
    const newSelected = new Set(selectedJobs);
    if (newSelected.has(jobId)) {
      newSelected.delete(jobId);
    } else {
      newSelected.add(jobId);
    }
    setSelectedJobs(newSelected);
  };

  const handleSelectAll = () => {
    const waitingJobs = queuedJobs.filter(j => j.status === 'waiting');
    if (selectedJobs.size === waitingJobs.length) {
      setSelectedJobs(new Set());
    } else {
      setSelectedJobs(new Set(waitingJobs.map(j => j.id)));
    }
  };

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'settings', label: 'Settings' },
    { id: 'queue', label: 'Print Queue' },
  ] as const;

  const stats = {
    queued: queuedJobs.filter(j => j.status === 'waiting').length,
    releasedToday: 47,
    expired: queuedJobs.filter(j => j.status === 'expired').length,
    avgReleaseTime: 8, // minutes
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            Secure Print
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage print release and PIN-based secure printing
          </p>
        </div>
        {config.enabled && (
          <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 text-sm font-medium">
            <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            Enabled
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-8 overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap
                ${activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }
              `}
              aria-current={activeTab === tab.id ? 'page' : undefined}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && (
        <div className="space-y-6">
          {/* Stats Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.queued}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Queued Jobs</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.releasedToday}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Released Today</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-amber-100 dark:bg-amber-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.avgReleaseTime}m</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Avg Release Time</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-red-100 dark:bg-red-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.expired}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Expired Jobs</p>
                </div>
              </div>
            </div>
          </div>

          {/* How It Works */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">How Secure Print Works</h2>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
              <div className="flex flex-col items-center text-center">
                <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-blue-600 dark:text-blue-400 font-bold mb-3">
                  1
                </div>
                <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-1">Print with PIN</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">User prints and enters a release PIN</p>
              </div>
              <div className="flex flex-col items-center text-center">
                <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-blue-600 dark:text-blue-400 font-bold mb-3">
                  2
                </div>
                <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-1">Job Queued</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">Document is held securely on the server</p>
              </div>
              <div className="flex flex-col items-center text-center">
                <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-blue-600 dark:text-blue-400 font-bold mb-3">
                  3
                </div>
                <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-1">Authenticate</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">User authenticates at the printer</p>
              </div>
              <div className="flex flex-col items-center text-center">
                <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-blue-600 dark:text-blue-400 font-bold mb-3">
                  4
                </div>
                <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-1">Release</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">Document is released for printing</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Settings Tab */}
      {activeTab === 'settings' && (
        <div className="space-y-6">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Secure Print Configuration</h2>
              <button
                onClick={() => setConfig({ ...config, enabled: !config.enabled })}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  config.enabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                }`}
                role="switch"
                aria-checked={config.enabled}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    config.enabled ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
            </div>

            {config.enabled && (
              <div className="space-y-6">
                {/* PIN Requirement */}
                <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <div>
                    <p className="font-medium text-gray-900 dark:text-gray-100">Require PIN for Release</p>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Users must enter a PIN to release print jobs</p>
                  </div>
                  <button
                    onClick={() => setConfig({ ...config, requirePin: !config.requirePin })}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      config.requirePin ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                    }`}
                    role="switch"
                    aria-checked={config.requirePin}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        config.requirePin ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>

                {config.requirePin && (
                  <div className="ml-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        PIN Type
                      </label>
                      <div className="flex gap-2">
                        {(['4-digit', '6-digit', 'custom'] as PinType[]).map((type) => (
                          <button
                            key={type}
                            onClick={() => setConfig({ ...config, pinType: type })}
                            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                              config.pinType === type
                                ? 'bg-blue-600 text-white'
                                : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300'
                            }`}
                          >
                            {type === 'custom' ? 'Custom Length' : type}
                          </button>
                        ))}
                      </div>
                    </div>

                    {config.pinType === 'custom' && (
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                          Custom PIN Length
                        </label>
                        <input
                          type="number"
                          min="4"
                          max="12"
                          value={config.customPinLength}
                          onChange={(e) => setConfig({ ...config, customPinLength: parseInt(e.target.value) || 6 })}
                          className="w-24 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Card Release */}
                <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <div>
                    <p className="font-medium text-gray-900 dark:text-gray-100">Require Card for Release</p>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Users must badge in to release print jobs</p>
                  </div>
                  <button
                    onClick={() => setConfig({ ...config, requireCardRelease: !config.requireCardRelease })}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      config.requireCardRelease ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                    }`}
                    role="switch"
                    aria-checked={config.requireCardRelease}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        config.requireCardRelease ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>

                {/* Auth at Release */}
                <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <div>
                    <p className="font-medium text-gray-900 dark:text-gray-100">Re-authenticate at Release</p>
                    <p className="text-sm text-gray-600 dark:text-gray-400">Users must verify identity before printing</p>
                  </div>
                  <button
                    onClick={() => setConfig({ ...config, requireAuthAtRelease: !config.requireAuthAtRelease })}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      config.requireAuthAtRelease ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                    }`}
                    role="switch"
                    aria-checked={config.requireAuthAtRelease}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        config.requireAuthAtRelease ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>

                {/* PIN Expiration */}
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    PIN Expiration (minutes)
                  </label>
                  <input
                    type="number"
                    min="30"
                    max="1440"
                    value={config.pinExpiration}
                    onChange={(e) => setConfig({ ...config, pinExpiration: parseInt(e.target.value) || 240 })}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    Queued jobs will expire after this period
                  </p>
                </div>

                {/* Max Queue Size */}
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Maximum Queue Size per User
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="100"
                    value={config.maxQueueSize}
                    onChange={(e) => setConfig({ ...config, maxQueueSize: parseInt(e.target.value) || 50 })}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    Maximum number of jobs a user can have queued
                  </p>
                </div>

                <button className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors">
                  Save Configuration
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Queue Tab */}
      {activeTab === 'queue' && (
        <div className="space-y-4">
          {/* Actions */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <input
                  type="checkbox"
                  checked={selectedJobs.size === queuedJobs.filter(j => j.status === 'waiting').length && selectedJobs.size > 0}
                  onChange={handleSelectAll}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {selectedJobs.size} selected
                </span>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={handleReleaseSelected}
                  disabled={selectedJobs.size === 0}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  Release Selected
                </button>
                <button
                  onClick={handleClearExpired}
                  className="px-4 py-2 border border-red-300 dark:border-red-800 text-red-700 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg text-sm font-medium transition-colors"
                >
                  Clear Expired
                </button>
              </div>
            </div>
          </div>

          {/* Queue Table */}
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50 dark:bg-gray-700/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-10">
                      <input
                        type="checkbox"
                        checked={selectedJobs.size === queuedJobs.filter(j => j.status === 'waiting').length && selectedJobs.size > 0}
                        onChange={handleSelectAll}
                        className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                      />
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Document
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      User
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Pages
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Submitted
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {queuedJobs.map((job) => (
                    <tr key={job.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                      <td className="px-6 py-4">
                        <input
                          type="checkbox"
                          checked={selectedJobs.has(job.id)}
                          disabled={job.status !== 'waiting'}
                          onChange={() => handleJobSelect(job.id)}
                          className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        />
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-3">
                          <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded">
                            <svg className="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                            </svg>
                          </div>
                          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {job.documentName}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">{job.user}</td>
                      <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">{job.pageCount}</td>
                      <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
                        {new Date(job.submittedAt).toLocaleString()}
                      </td>
                      <td className="px-6 py-4">
                        <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                          job.status === 'waiting' ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400' :
                          job.status === 'released' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' :
                          'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400'
                        }`}>
                          {job.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-right">
                        {job.status === 'waiting' && (
                          <button className="p-2 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors">
                            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                            </svg>
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
