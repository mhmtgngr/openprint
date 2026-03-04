import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  useAuditLogs,
  useComplianceOverview,
  useComplianceReports,
  useRetentionPolicy,
  useSecuritySettings,
  useRunComplianceChecklist,
  useRunRiskAssessment,
  useUpdateRetentionPolicy,
  useUpdateSecuritySettings,
  useAddIPToWhitelist,
  useRemoveIPFromWhitelist,
  useExportAuditLogs,
  useGenerateReport,
  type AuditEvent,
  type ChecklistItem,
  type ComplianceFramework,
  type ComplianceOverviewType,
  type DataRetentionPolicyType,
  type SecuritySettingsType,
  type IPWhitelistEntry,
  // Components
  ComplianceOverview,
  AuditLogsTable,
  ComplianceReports,
  DataRetentionPolicy,
  SecuritySettings,
  ComplianceChecklist,
  RiskAssessment,
} from '@/features/compliance';

type TabId = 'overview' | 'audit-logs' | 'reports' | 'retention' | 'security';

const tabs: { id: TabId; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'audit-logs', label: 'Audit Logs' },
  { id: 'reports', label: 'Reports' },
  { id: 'retention', label: 'Data Retention' },
  { id: 'security', label: 'Security Settings' },
];

export const Compliance = () => {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<TabId>('overview');
  const [checklistItems, setChecklistItems] = useState<ChecklistItem[]>([]);
  const [riskScore, setRiskScore] = useState<number | null>(null);
  const [riskLevel, setRiskLevel] = useState<'low' | 'medium' | 'high' | null>(null);
  const [riskMitigations, setRiskMitigations] = useState<string[]>([]);

  // Fetch data using hooks
  const {
    data: overview,
    isLoading: overviewLoading,
    error: overviewError,
  } = useComplianceOverview();

  const {
    data: auditLogsData,
    isLoading: auditLogsLoading,
    error: auditLogsError,
    refetch: refetchAuditLogs,
  } = useAuditLogs({ limit: 100 });

  const {
    data: reportsData,
    isLoading: reportsLoading,
    error: reportsError,
  } = useComplianceReports();

  const {
    data: retentionPolicy,
    isLoading: retentionLoading,
  } = useRetentionPolicy();

  const {
    data: securitySettings,
    isLoading: securityLoading,
  } = useSecuritySettings();

  // Mutations
  const updateRetentionPolicy = useUpdateRetentionPolicy();
  const updateSecuritySettings = useUpdateSecuritySettings();
  const addIPToWhitelist = useAddIPToWhitelist();
  const removeIPFromWhitelist = useRemoveIPFromWhitelist();
  const exportAuditLogsMutation = useExportAuditLogs();
  const generateReportMutation = useGenerateReport();

  const runChecklistMutation = useRunComplianceChecklist();
  const runRiskAssessmentMutation = useRunRiskAssessment();

  // Convert overview data format
  const formattedOverview: ComplianceOverviewType | undefined = overview
    ? {
        fedramp: {
          status: overview.fedramp.status,
          last_audit: overview.fedramp.last_audit,
        },
        hipaa: {
          status: overview.hipaa.status,
          last_audit: overview.hipaa.last_audit,
        },
        gdpr: {
          status: overview.gdpr.status,
          last_audit: overview.gdpr.last_audit,
        },
        soc2: {
          status: overview.soc2.status,
          last_audit: overview.soc2.last_audit,
        },
        total_logs: overview.total_logs,
        compliant_standards: overview.compliant_standards,
        pending_actions: overview.pending_actions,
      }
    : undefined;

  // Convert audit logs data format
  const formattedAuditLogs: AuditEvent[] = auditLogsData?.events || [];

  // Convert reports data format
  const formattedReports = reportsData?.reports || [];

  // Format retention policy
  const formattedRetentionPolicy: DataRetentionPolicyType | undefined =
    retentionPolicy;

  // Format security settings
  const formattedSecuritySettings: SecuritySettingsType | undefined =
    securitySettings;

  // Handlers
  const handleRunChecklist = async () => {
    try {
      const result = await runChecklistMutation.mutateAsync();
      setChecklistItems(result.checklist);
    } catch (error) {
      console.error('Failed to run checklist:', error);
    }
  };

  const handleRunRiskAssessment = async () => {
    try {
      const result = await runRiskAssessmentMutation.mutateAsync();
      setRiskScore(result.risk_score);
      setRiskLevel(result.level);
      setRiskMitigations(result.mitigations);
    } catch (error) {
      console.error('Failed to run risk assessment:', error);
    }
  };

  const handleExportAuditLogs = async (format: 'csv' | 'json' | 'xlsx') => {
    try {
      // Convert xlsx to json for API (we can add xlsx support later)
      const apiFormat: 'csv' | 'json' = format === 'xlsx' ? 'json' : format;
      const blob = await exportAuditLogsMutation.mutateAsync({ format: apiFormat });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to export audit logs:', error);
    }
  };

  const handleGenerateReport = async (params: {
    framework: string;
    period_start: string;
    period_end: string;
    format?: 'pdf' | 'json';
  }) => {
    try {
      await generateReportMutation.mutateAsync({
        framework: params.framework as ComplianceFramework,
        period_start: params.period_start,
        period_end: params.period_end,
      });
      queryClient.invalidateQueries({ queryKey: ['compliance', 'reports'] });
    } catch (error) {
      console.error('Failed to generate report:', error);
    }
  };

  const handleUpdateRetentionPolicy = async (
    policy: Partial<DataRetentionPolicyType>
  ) => {
    await updateRetentionPolicy.mutateAsync(policy);
  };

  const handleUpdateSecuritySettings = async (
    settings: Partial<SecuritySettingsType>
  ) => {
    await updateSecuritySettings.mutateAsync(settings);
  };

  const handleAddIPWhitelist = async (entry: Omit<IPWhitelistEntry, 'id'>) => {
    await addIPToWhitelist.mutateAsync(entry);
  };

  const handleRemoveIPWhitelist = async (ip: string) => {
    await removeIPFromWhitelist.mutateAsync(ip);
  };

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
              onClick={() => setActiveTab(tab.id)}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap
                ${
                  activeTab === tab.id
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
          <ComplianceOverview
            data={formattedOverview}
            isLoading={overviewLoading}
            error={overviewError?.message}
          />
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <ComplianceChecklist
              checklist={checklistItems}
              onRun={handleRunChecklist}
              isRunning={runChecklistMutation.isPending}
            />
            <RiskAssessment
              riskScore={riskScore}
              level={riskLevel}
              mitigations={riskMitigations}
              onRun={handleRunRiskAssessment}
              isRunning={runRiskAssessmentMutation.isPending}
            />
          </div>
        </div>
      )}

      {/* Audit Logs Tab */}
      {activeTab === 'audit-logs' && (
        <AuditLogsTable
          logs={formattedAuditLogs}
          isLoading={auditLogsLoading}
          error={auditLogsError?.message}
          onExport={handleExportAuditLogs}
          onRefresh={() => refetchAuditLogs()}
          totalCount={auditLogsData?.total || 0}
        />
      )}

      {/* Reports Tab */}
      {activeTab === 'reports' && (
        <ComplianceReports
          reports={formattedReports}
          isLoading={reportsLoading}
          error={reportsError?.message}
          onGenerate={handleGenerateReport}
        />
      )}

      {/* Data Retention Tab */}
      {activeTab === 'retention' && (
        <DataRetentionPolicy
          policy={formattedRetentionPolicy}
          isLoading={retentionLoading}
          onUpdate={handleUpdateRetentionPolicy}
        />
      )}

      {/* Security Settings Tab */}
      {activeTab === 'security' && (
        <SecuritySettings
          settings={formattedSecuritySettings}
          isLoading={securityLoading}
          onUpdate={handleUpdateSecuritySettings}
          onAddIPWhitelist={handleAddIPWhitelist}
          onRemoveIPWhitelist={handleRemoveIPWhitelist}
        />
      )}
    </div>
  );
};

export default Compliance;
