import { useState } from 'react';
import { AlertRule, AlertSeverity } from '@/types';
import { PlusIcon, EditIcon, TrashIcon, CheckIcon, XCircleIcon } from '@/components/icons';

interface AlertRulesManagerProps {
  rules: AlertRule[];
  onCreateRule?: (rule: Omit<AlertRule, 'id'>) => void;
  onUpdateRule?: (id: string, rule: Partial<AlertRule>) => void;
  onDeleteRule?: (id: string) => void;
  onToggleRule?: (id: string) => void;
}

interface RuleFormData {
  name: string;
  query: string;
  duration: string;
  severity: AlertSeverity;
  annotations: Record<string, string>;
  labels: Record<string, string>;
}

const SEVERITY_OPTIONS: { value: AlertSeverity; label: string; color: string }[] = [
  { value: 'critical', label: 'Critical', color: 'red' },
  { value: 'warning', label: 'Warning', color: 'amber' },
  { value: 'info', label: 'Info', color: 'blue' },
  { value: 'none', label: 'None', color: 'gray' },
];

const DURATION_PRESETS = ['1m', '5m', '15m', '30m', '1h', '4h', '24h'];

const COMMON_TEMPLATES = [
  {
    name: 'High Error Rate',
    query: 'sum(rate(http_requests_total{status=~"5.."}[5m])) by (service) > 0.05',
    severity: 'critical' as AlertSeverity,
    duration: '5m',
    annotations: {
      summary: 'High error rate on {{ $labels.service }}',
      description: 'Error rate is {{ $value }} errors/sec',
    },
  },
  {
    name: 'High Latency',
    query: 'histogram_quantile(0.95, sum(rate(http_request_duration_ms_bucket[5m])) by (le, service)) > 100',
    severity: 'warning' as AlertSeverity,
    duration: '10m',
    annotations: {
      summary: 'High P95 latency on {{ $labels.service }}',
      description: 'P95 latency is {{ $value }}ms',
    },
  },
  {
    name: 'Service Down',
    query: 'up{job=~".*service"} == 0',
    severity: 'critical' as AlertSeverity,
    duration: '1m',
    annotations: {
      summary: '{{ $labels.job }} is down',
      description: 'Instance {{ $labels.instance }} of job {{ $labels.job }} is down',
    },
  },
  {
    name: 'High CPU Usage',
    query: 'sum(rate(process_cpu_seconds_total[5m])) by (service) * 100 > 80',
    severity: 'warning' as AlertSeverity,
    duration: '15m',
    annotations: {
      summary: 'High CPU usage on {{ $labels.service }}',
      description: 'CPU usage is {{ $value }}%',
    },
  },
  {
    name: 'High Memory Usage',
    query: 'sum(process_resident_memory_bytes{service=~".*-service"}) by (service) / 1024 / 1024 / 1024 > 1',
    severity: 'warning' as AlertSeverity,
    duration: '15m',
    annotations: {
      summary: 'High memory usage on {{ $labels.service }}',
      description: 'Memory usage is {{ $value }}GB',
    },
  },
];

export const AlertRulesManager = ({
  rules = [],
  onCreateRule,
  onUpdateRule,
  onDeleteRule,
  onToggleRule,
}: AlertRulesManagerProps) => {
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<typeof COMMON_TEMPLATES[0] | null>(null);

  const handleSubmit = (data: RuleFormData) => {
    if (editingRule) {
      onUpdateRule?.(editingRule.id, data);
    } else {
      onCreateRule?.({
        name: data.name,
        query: data.query,
        duration: data.duration,
        labels: data.labels,
        annotations: data.annotations,
        isEnabled: true,
      });
    }
    setShowCreateForm(false);
    setEditingRule(null);
    setSelectedTemplate(null);
  };

  const handleEdit = (rule: AlertRule) => {
    setEditingRule(rule);
    setShowCreateForm(true);
  };

  const handleCancel = () => {
    setShowCreateForm(false);
    setEditingRule(null);
    setSelectedTemplate(null);
  };

  const handleUseTemplate = (template: typeof COMMON_TEMPLATES[0]) => {
    setSelectedTemplate(template);
    setShowCreateForm(true);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Alert Rules</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Manage Prometheus alert rules for monitoring
          </p>
        </div>
        <button
          onClick={() => setShowCreateForm(true)}
          className="px-4 py-2 bg-blue-600 text-white hover:bg-blue-700 rounded-lg font-medium flex items-center gap-2 transition-colors"
        >
          <PlusIcon className="w-4 h-4" />
          New Rule
        </button>
      </div>

      {/* Create/Edit Form */}
      {showCreateForm && (
        <AlertRuleForm
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          initialData={editingRule || undefined}
          template={selectedTemplate || undefined}
        />
      )}

      {/* Templates */}
      {!showCreateForm && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Quick Templates</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {COMMON_TEMPLATES.map((template) => {
              const severity = SEVERITY_OPTIONS.find((s) => s.value === template.severity);
              return (
                <div
                  key={template.name}
                  className="p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-700 cursor-pointer transition-colors"
                  onClick={() => handleUseTemplate(template)}
                >
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{template.name}</p>
                    <span
                      className={`px-2 py-0.5 rounded text-xs font-medium bg-${severity?.color}-100 dark:bg-${severity?.color}-900/30 text-${severity?.color}-700 dark:text-${severity?.color}-400`}
                    >
                      {severity?.label}
                    </span>
                  </div>
                  <p className="text-xs font-mono text-gray-500 dark:text-gray-400 truncate">{template.query}</p>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Rules List */}
      <div className="space-y-3">
        {rules.length === 0 ? (
          <div className="bg-white dark:bg-gray-800 rounded-xl p-8 text-center border border-gray-200 dark:border-gray-700">
            <CheckIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600 dark:text-gray-400">No alert rules configured</p>
            <p className="text-sm text-gray-500 dark:text-gray-500 mt-1">
              Create a rule to start monitoring your services
            </p>
          </div>
        ) : (
          rules.map((rule) => (
            <AlertRuleCard
              key={rule.id}
              rule={rule}
              onEdit={() => handleEdit(rule)}
              onDelete={() => onDeleteRule?.(rule.id)}
              onToggle={() => onToggleRule?.(rule.id)}
            />
          ))
        )}
      </div>
    </div>
  );
};

interface AlertRuleCardProps {
  rule: AlertRule;
  onEdit: () => void;
  onDelete: () => void;
  onToggle: () => void;
}

const AlertRuleCard = ({ rule, onEdit, onDelete, onToggle }: AlertRuleCardProps) => {
  const severity = SEVERITY_OPTIONS.find((s) => s.value === rule.labels.severity as AlertSeverity);

  return (
    <div
      className={`bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border transition-all ${
        rule.isEnabled
          ? 'border-gray-200 dark:border-gray-700'
          : 'border-gray-200 dark:border-gray-700 opacity-60'
      }`}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-2">
            <h4 className="font-medium text-gray-900 dark:text-gray-100">{rule.name}</h4>
            {severity && (
              <span
                className={`px-2 py-0.5 rounded text-xs font-medium bg-${severity.color}-100 dark:bg-${severity.color}-900/30 text-${severity.color}-700 dark:text-${severity.color}-400`}
              >
                {severity.label}
              </span>
            )}
            <span
              className={`px-2 py-0.5 rounded text-xs font-medium ${
                rule.isEnabled
                  ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
              }`}
            >
              {rule.isEnabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>
          <p className="text-xs font-mono text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-900 rounded px-2 py-1 mb-2">
            {rule.query}
          </p>
          <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
            <span>Duration: {rule.duration}</span>
            {Object.entries(rule.annotations).slice(0, 2).map(([key, value]) => (
              <span key={key}>
                {key}: {value}
              </span>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2 ml-4">
          <button
            onClick={onToggle}
            className={`p-2 rounded-lg transition-colors ${
              rule.isEnabled
                ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 hover:bg-green-200 dark:hover:bg-green-900/50'
                : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'
            }`}
            title={rule.isEnabled ? 'Disable' : 'Enable'}
          >
            {rule.isEnabled ? <CheckIcon className="w-4 h-4" /> : <XCircleIcon className="w-4 h-4" />}
          </button>
          <button
            onClick={onEdit}
            className="p-2 rounded-lg bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            title="Edit"
          >
            <EditIcon className="w-4 h-4" />
          </button>
          <button
            onClick={onDelete}
            className="p-2 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/40 transition-colors"
            title="Delete"
          >
            <TrashIcon className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
};

interface AlertRuleFormProps {
  onSubmit: (data: RuleFormData) => void;
  onCancel: () => void;
  initialData?: AlertRule;
  template?: typeof COMMON_TEMPLATES[0];
}

const AlertRuleForm = ({ onSubmit, onCancel, initialData, template }: AlertRuleFormProps) => {
  const [name, setName] = useState(initialData?.name || template?.name || '');
  const [query, setQuery] = useState(initialData?.query || template?.query || '');
  const [duration, setDuration] = useState(initialData?.duration || template?.duration || '5m');
  const [severity, setSeverity] = useState<AlertSeverity>(
    (initialData?.labels.severity as AlertSeverity) || template?.severity || 'warning'
  );
  const [summary, setSummary] = useState(initialData?.annotations.summary || template?.annotations.summary || '');
  const [description, setDescription] = useState(
    initialData?.annotations.description || template?.annotations.description || ''
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      name,
      query,
      duration,
      severity,
      annotations: {
        summary,
        description,
      },
      labels: {
        severity,
      },
    });
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
      <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
        {initialData ? 'Edit Alert Rule' : 'Create Alert Rule'}
      </h3>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Rule Name
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            placeholder="e.g., HighErrorRate"
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Query */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            PromQL Query
          </label>
          <textarea
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            required
            rows={3}
            placeholder="sum(rate(http_requests_total{status=~'5..'}[5m])) by (service) > 0.05"
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
          />
        </div>

        {/* Duration & Severity */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Duration
            </label>
            <select
              value={duration}
              onChange={(e) => setDuration(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {DURATION_PRESETS.map((d) => (
                <option key={d} value={d}>
                  {d}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Severity
            </label>
            <select
              value={severity}
              onChange={(e) => setSeverity(e.target.value as AlertSeverity)}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              {SEVERITY_OPTIONS.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Annotations */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Summary Annotation
          </label>
          <input
            type="text"
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            placeholder="{{ $labels.service }} has high error rate"
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Description Annotation
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            placeholder="Error rate is {{ $value }} errors/sec"
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            className="px-4 py-2 bg-blue-600 text-white hover:bg-blue-700 rounded-lg font-medium transition-colors"
          >
            {initialData ? 'Update Rule' : 'Create Rule'}
          </button>
        </div>
      </form>
    </div>
  );
};
