import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';

type ComplianceStandard = 'fedramp' | 'hipaa' | 'gdpr' | 'soc2';
type ComplianceStatus = 'compliant' | 'non_compliant' | 'in_progress' | 'pending';

interface ComplianceOverview {
  fedramp: { status: ComplianceStatus; lastAudit: string };
  hipaa: { status: ComplianceStatus; lastAudit: string };
  gdpr: { status: ComplianceStatus; lastAudit: string };
  soc2: { status: ComplianceStatus; lastAudit: string };
  totalLogs: number;
  compliantStandards: number;
  pendingActions: number;
}

interface AuditLog {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  resource: string;
  details: string;
  ipAddress: string;
}

interface ComplianceReport {
  id: string;
  name: string;
  type: ComplianceStandard;
  createdAt: string;
  status: 'complete' | 'generating' | 'failed';
}

interface ChecklistItem {
  name: string;
  status: 'pass' | 'fail' | 'warning' | 'pending';
}

const StatusBadge = ({ status }: { status: ComplianceStatus }) => {
  const variants = {
    compliant: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 border-green-200 dark:border-green-800',
    non_compliant: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800',
    in_progress: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800',
    pending: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-600',
  };

  const labels = {
    compliant: 'Compliant',
    non_compliant: 'Non-Compliant',
    in_progress: 'In Progress',
    pending: 'Pending',
  };

  return (
    <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border ${variants[status]}`}>
      {labels[status]}
    </span>
  );
};

const ChecklistStatusBadge = ({ status }: { status: ChecklistItem['status'] }) => {
  const variants = {
    pass: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
    fail: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
    warning: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400',
    pending: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400',
  };

  const icons = {
    pass: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
      </svg>
    ),
    fail: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
      </svg>
    ),
    warning: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
      </svg>
    ),
    pending: (
      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clipRule="evenodd" />
      </svg>
    ),
  };

  return (
    <div className={`inline-flex items-center gap-2 px-3 py-2 rounded-lg ${variants[status]}`}>
      {icons[status]}
    </div>
  );
};

export const Compliance = () => {
  const [activeTab, setActiveTab] = useState<'overview' | 'audit-logs' | 'reports' | 'retention' | 'security'>('overview');
  const [auditLogSearch, setAuditLogSearch] = useState('');
  const [auditLogActionFilter, setAuditLogActionFilter] = useState('all');
  const [auditLogUserFilter, setAuditLogUserFilter] = useState('all');
  const [retentionEnabled, setRetentionEnabled] = useState(false);
  const [retentionPeriod, setRetentionPeriod] = useState('90');
  const [encryptionEnabled, setEncryptionEnabled] = useState(false);
  const [encryptionAlgorithm, setEncryptionAlgorithm] = useState('AES-256');
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(false);
  const [sessionTimeout, setSessionTimeout] = useState('30');
  const [newIpWhitelist, setNewIpWhitelist] = useState('');
  const [ipWhitelistDescription, setIpWhitelistDescription] = useState('');
  const [ipWhitelist, setIpWhitelist] = useState<Array<{ ip: string; description: string }>>([
    { ip: '192.168.1.100', description: 'Office Network' },
    { ip: '192.168.1.101', description: 'Admin VPN' },
  ]);
  const [riskScore, setRiskScore] = useState<number | null>(null);
  const [checklistItems, setChecklistItems] = useState<ChecklistItem[]>([]);

  // Mock data - in real app, fetch from API
  const { data: overview = {
    fedramp: { status: 'compliant', lastAudit: '2024-01-15' },
    hipaa: { status: 'compliant', lastAudit: '2024-01-15' },
    gdpr: { status: 'compliant', lastAudit: '2024-01-15' },
    soc2: { status: 'in_progress', lastAudit: '2024-01-15' },
    totalLogs: 1523,
    compliantStandards: 3,
    pendingActions: 5,
  } } = useQuery({
    queryKey: ['compliance-overview'],
    queryFn: async () => {
      // Mock API call
      return {
        fedramp: { status: 'compliant', lastAudit: '2024-01-15' },
        hipaa: { status: 'compliant', lastAudit: '2024-01-15' },
        gdpr: { status: 'compliant', lastAudit: '2024-01-15' },
        soc2: { status: 'in_progress', lastAudit: '2024-01-15' },
        totalLogs: 1523,
        compliantStandards: 3,
        pendingActions: 5,
      } as ComplianceOverview;
    },
  });

  const auditLogs: AuditLog[] = [
    {
      id: '1',
      timestamp: '2024-01-15T10:30:00Z',
      user: 'admin@openprint.test',
      action: 'login',
      resource: '/login',
      details: 'Successful login',
      ipAddress: '192.168.1.100',
    },
    {
      id: '2',
      timestamp: '2024-01-15T10:25:00Z',
      user: 'user@openprint.test',
      action: 'print_job_created',
      resource: '/jobs',
      details: 'Created job "Document.pdf"',
      ipAddress: '192.168.1.101',
    },
    {
      id: '3',
      timestamp: '2024-01-15T10:20:00Z',
      user: 'admin@openprint.test',
      action: 'policy_updated',
      resource: '/policies',
      details: 'Updated policy "Color Restriction"',
      ipAddress: '192.168.1.100',
    },
  ];

  const reports: ComplianceReport[] = [
    {
      id: 'report-1',
      name: 'FedRAMP Assessment',
      type: 'fedramp',
      createdAt: '2024-01-01T00:00:00Z',
      status: 'complete',
    },
    {
      id: 'report-2',
      name: 'HIPAA Audit',
      type: 'hipaa',
      createdAt: '2024-01-02T00:00:00Z',
      status: 'complete',
    },
    {
      id: 'report-3',
      name: 'SOC 2 Audit',
      type: 'soc2',
      createdAt: '2024-01-10T00:00:00Z',
      status: 'generating',
    },
  ];

  const filteredAuditLogs = auditLogs.filter((log) => {
    const matchesSearch = !auditLogSearch ||
      log.user.toLowerCase().includes(auditLogSearch.toLowerCase()) ||
      log.action.toLowerCase().includes(auditLogSearch.toLowerCase()) ||
      log.details.toLowerCase().includes(auditLogSearch.toLowerCase());
    const matchesAction = auditLogActionFilter === 'all' || log.action === auditLogActionFilter;
    const matchesUser = auditLogUserFilter === 'all' || log.user === auditLogUserFilter;
    return matchesSearch && matchesAction && matchesUser;
  });

  const handleExportAuditLogs = (format: 'csv' | 'json' | 'xlsx') => {
    const data = filteredAuditLogs;
    let content = '';
    let filename = '';
    let type = '';

    if (format === 'csv') {
      content = [
        ['Timestamp', 'User', 'Action', 'Resource', 'Details', 'IP Address'],
        ...data.map(log => [log.timestamp, log.user, log.action, log.resource, log.details, log.ipAddress]),
      ].map(row => row.join(',')).join('\n');
      filename = `audit-logs-${new Date().toISOString().split('T')[0]}.csv`;
      type = 'text/csv';
    } else if (format === 'json') {
      content = JSON.stringify(data, null, 2);
      filename = `audit-logs-${new Date().toISOString().split('T')[0]}.json`;
      type = 'application/json';
    } else {
      content = JSON.stringify(data, null, 2);
      filename = `audit-logs-${new Date().toISOString().split('T')[0]}.xlsx`;
      type = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
    }

    const blob = new Blob([content], { type });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleRunRiskAssessment = () => {
    setRiskScore(25);
  };

  const handleRunComplianceChecklist = () => {
    setChecklistItems([
      { name: 'Access Control', status: 'pass' },
      { name: 'Audit Logging', status: 'pass' },
      { name: 'Data Encryption', status: 'pass' },
      { name: 'Incident Response', status: 'warning' },
      { name: 'Security Training', status: 'pending' },
    ]);
  };

  const handleAddIpWhitelist = () => {
    if (newIpWhitelist) {
      setIpWhitelist([...ipWhitelist, { ip: newIpWhitelist, description: ipWhitelistDescription }]);
      setNewIpWhitelist('');
      setIpWhitelistDescription('');
    }
  };

  const handleRemoveIpWhitelist = (ip: string) => {
    setIpWhitelist(ipWhitelist.filter(item => item.ip !== ip));
  };

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'audit-logs', label: 'Audit Logs' },
    { id: 'reports', label: 'Reports' },
    { id: 'retention', label: 'Data Retention' },
    { id: 'security', label: 'Security Settings' },
  ] as const;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            Compliance Center
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage FedRAMP, HIPAA, GDPR, and SOC 2 compliance
          </p>
        </div>
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
          {/* Compliance Standards */}
          <div data-testid="compliance-overview" className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Compliance Status</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">FedRAMP</span>
                  <div data-testid="fedramp-status">
                    <StatusBadge status={overview.fedramp.status} />
                  </div>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Last audit: {overview.fedramp.lastAudit}
                </p>
              </div>
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">HIPAA</span>
                  <div data-testid="hipaa-status">
                    <StatusBadge status={overview.hipaa.status} />
                  </div>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Last audit: {overview.hipaa.lastAudit}
                </p>
              </div>
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">GDPR</span>
                  <div data-testid="gdpr-status">
                    <StatusBadge status={overview.gdpr.status} />
                  </div>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Last audit: {overview.gdpr.lastAudit}
                </p>
              </div>
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">SOC 2</span>
                  <div data-testid="soc2-status">
                    <StatusBadge status={overview.soc2.status} />
                  </div>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Last audit: {overview.soc2.lastAudit}
                </p>
              </div>
            </div>
          </div>

          {/* Quick Stats */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{overview.totalLogs}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Total Audit Logs</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{overview.compliantStandards}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Compliant Standards</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-4">
                <div className="p-3 bg-amber-100 dark:bg-amber-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{overview.pendingActions}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Pending Actions</p>
                </div>
              </div>
            </div>
          </div>

          {/* Compliance Checklist */}
          <div data-testid="compliance-checklist" className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Compliance Checklist</h2>
              <button
                onClick={handleRunComplianceChecklist}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
              >
                Run Checklist
              </button>
            </div>
            {checklistItems.length === 0 ? (
              <p className="text-gray-500 dark:text-gray-400 text-sm py-4">Click "Run Checklist" to verify compliance status</p>
            ) : (
              <div className="space-y-2">
                {checklistItems.map((item, index) => (
                  <div
                    key={index}
                    data-testid="checklist-item"
                    className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
                  >
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{item.name}</span>
                    <ChecklistStatusBadge status={item.status} />
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Risk Assessment */}
          <div data-testid="risk-assessment-section" className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Risk Assessment</h2>
              <button
                onClick={handleRunRiskAssessment}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
              >
                Run Assessment
              </button>
            </div>
            {riskScore !== null ? (
              <div className="flex items-center gap-6">
                <div data-testid="risk-score" className="text-center">
                  <div className={`text-5xl font-bold ${
                    riskScore < 30 ? 'text-green-600 dark:text-green-400' :
                    riskScore < 60 ? 'text-amber-600 dark:text-amber-400' :
                    'text-red-600 dark:text-red-400'
                  }`}>
                    {riskScore}
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    Risk Score
                  </p>
                </div>
                <div className="flex-1">
                  <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className={`h-full ${
                        riskScore < 30 ? 'bg-green-500' :
                        riskScore < 60 ? 'bg-amber-500' :
                        'bg-red-500'
                      }`}
                      style={{ width: `${riskScore}%` }}
                    />
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                    {riskScore < 30 ? 'Low Risk - Good security posture' :
                     riskScore < 60 ? 'Medium Risk - Some improvements needed' :
                     'High Risk - Immediate action required'}
                  </p>
                </div>
              </div>
            ) : (
              <p className="text-gray-500 dark:text-gray-400 text-sm">Click "Run Assessment" to evaluate security risks</p>
            )}
          </div>
        </div>
      )}

      {/* Audit Logs Tab */}
      {activeTab === 'audit-logs' && (
        <section data-testid="audit-logs-section" className="space-y-4">
          {/* Filters */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <input
                  type="text"
                  value={auditLogSearch}
                  onChange={(e) => setAuditLogSearch(e.target.value)}
                  placeholder="Search logs..."
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              <select
                value={auditLogActionFilter}
                onChange={(e) => setAuditLogActionFilter(e.target.value)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
              >
                <option value="all">All Actions</option>
                <option value="login">Login</option>
                <option value="print_job_created">Job Created</option>
                <option value="policy_updated">Policy Updated</option>
              </select>
              <select
                value={auditLogUserFilter}
                onChange={(e) => setAuditLogUserFilter(e.target.value)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
              >
                <option value="all">All Users</option>
                <option value="admin@openprint.test">admin@openprint.test</option>
                <option value="user@openprint.test">user@openprint.test</option>
              </select>
            </div>
          </div>

          {/* Audit Logs Table */}
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="overflow-x-auto">
              <table data-testid="audit-logs-table" className="w-full">
                <thead className="bg-gray-50 dark:bg-gray-700/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Timestamp
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      User
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Action
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Resource
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Details
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      IP Address
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {filteredAuditLogs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600 dark:text-gray-400">
                        {new Date(log.timestamp).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">{log.user}</td>
                      <td className="px-6 py-4 text-sm text-gray-900 dark:text-gray-100">{log.action}</td>
                      <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">{log.resource}</td>
                      <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400 max-w-xs truncate">{log.details}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-500 font-mono">{log.ipAddress}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Export Actions */}
          <div className="flex justify-end gap-2">
            <button
              onClick={() => handleExportAuditLogs('csv')}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Export CSV
            </button>
            <button
              onClick={() => handleExportAuditLogs('json')}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Export JSON
            </button>
            <button
              onClick={() => handleExportAuditLogs('xlsx')}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Export XLSX
            </button>
          </div>
        </section>
      )}

      {/* Reports Tab */}
      {activeTab === 'reports' && (
        <section data-testid="compliance-reports-section" className="space-y-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Compliance Reports</h2>
            <div data-testid="reports-list" className="space-y-3">
              {reports.map((report) => (
                <div
                  key={report.id}
                  data-report-id={report.id}
                  data-testid="report-item"
                  className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
                >
                  <div className="flex items-center gap-4">
                    <div className={`p-2 rounded-lg ${
                      report.type === 'fedramp' ? 'bg-blue-100 dark:bg-blue-900/30' :
                      report.type === 'hipaa' ? 'bg-green-100 dark:bg-green-900/30' :
                      report.type === 'gdpr' ? 'bg-purple-100 dark:bg-purple-900/30' :
                      'bg-amber-100 dark:bg-amber-900/30'
                    }`}>
                      <svg className="w-5 h-5 text-gray-700 dark:text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                    </div>
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100">{report.name}</p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {new Date(report.createdAt).toLocaleDateString()} • {report.type.toUpperCase()}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`text-xs px-2 py-1 rounded-full ${
                      report.status === 'complete' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' :
                      report.status === 'generating' ? 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400' :
                      'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'
                    }`}>
                      {report.status}
                    </span>
                    {report.status === 'complete' && (
                      <button className="p-2 text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors">
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                        </svg>
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      )}

      {/* Data Retention Tab */}
      {activeTab === 'retention' && (
        <section data-testid="data-retention-section" className="space-y-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Data Retention Policy</h2>
            <div className="space-y-4">
              <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div>
                  <p className="font-medium text-gray-900 dark:text-gray-100">Enable Automatic Retention</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Automatically delete old audit logs after specified period</p>
                </div>
                <button
                  onClick={() => setRetentionEnabled(!retentionEnabled)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    retentionEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                  }`}
                  role="switch"
                  aria-checked={retentionEnabled}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      retentionEnabled ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              {retentionEnabled && (
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Retention Period
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="number"
                        value={retentionPeriod}
                        onChange={(e) => setRetentionPeriod(e.target.value)}
                        className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                      />
                      <select className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100">
                        <option>Days</option>
                        <option>Months</option>
                        <option>Years</option>
                      </select>
                    </div>
                  </div>
                  <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors">
                    Save Retention Policy
                  </button>
                </div>
              )}
            </div>
          </div>
        </section>
      )}

      {/* Security Settings Tab */}
      {activeTab === 'security' && (
        <section data-testid="security-settings-section" className="space-y-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Security Settings</h2>

            <div className="space-y-6">
              {/* Encryption */}
              <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div>
                  <p className="font-medium text-gray-900 dark:text-gray-100">Data Encryption</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Encrypt sensitive data at rest</p>
                </div>
                <button
                  onClick={() => setEncryptionEnabled(!encryptionEnabled)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    encryptionEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                  }`}
                  role="switch"
                  aria-checked={encryptionEnabled}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      encryptionEnabled ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              {encryptionEnabled && (
                <div className="ml-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <select
                    value={encryptionAlgorithm}
                    onChange={(e) => setEncryptionAlgorithm(e.target.value)}
                    className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  >
                    <option>AES-256</option>
                    <option>AES-128</option>
                    <option>ChaCha20</option>
                  </select>
                </div>
              )}

              {/* Two-Factor Authentication */}
              <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div>
                  <p className="font-medium text-gray-900 dark:text-gray-100">Two-Factor Authentication (2FA)</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Require 2FA for all admin users</p>
                </div>
                <button
                  onClick={() => setTwoFactorEnabled(!twoFactorEnabled)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    twoFactorEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                  }`}
                  role="switch"
                  aria-checked={twoFactorEnabled}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      twoFactorEnabled ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              {/* Session Timeout */}
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Session Timeout (minutes)
                </label>
                <input
                  type="number"
                  value={sessionTimeout}
                  onChange={(e) => setSessionTimeout(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                />
              </div>

              {/* IP Whitelist */}
              <div data-testid="ip-whitelist" className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <p className="font-medium text-gray-900 dark:text-gray-100 mb-3">IP Whitelist</p>
                <div className="flex gap-2 mb-4">
                  <input
                    type="text"
                    value={newIpWhitelist}
                    onChange={(e) => setNewIpWhitelist(e.target.value)}
                    placeholder="192.168.1.100"
                    className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                  <input
                    type="text"
                    value={ipWhitelistDescription}
                    onChange={(e) => setIpWhitelistDescription(e.target.value)}
                    placeholder="Description (optional)"
                    className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                  />
                  <button
                    onClick={handleAddIpWhitelist}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
                  >
                    Add
                  </button>
                </div>
                <div className="space-y-2">
                  {ipWhitelist.map((item, index) => (
                    <div
                      key={index}
                      data-testid="whitelist-item"
                      className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg"
                    >
                      <div>
                        <p className="font-medium text-gray-900 dark:text-gray-100">{item.ip}</p>
                        <p className="text-sm text-gray-500 dark:text-gray-400">{item.description}</p>
                      </div>
                      <button
                        onClick={() => handleRemoveIpWhitelist(item.ip)}
                        className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              <button className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors">
                Save Security Settings
              </button>
            </div>
          </div>
        </section>
      )}
    </div>
  );
};
