import { useState } from 'react';
import { Alert, AlertSeverity } from '@/types';
import { AlertIcon, CheckIcon, XCircleIcon, ClockIcon } from '@/components/icons';

interface AlertPanelProps {
  alerts: Alert[];
  onSilence?: (alertId: string) => void;
  onView?: (alert: Alert) => void;
}

const severityConfig: Record<
  AlertSeverity,
  { bg: string; text: string; border: string; icon: React.ReactNode }
> = {
  critical: {
    bg: 'bg-red-50 dark:bg-red-900/20',
    text: 'text-red-700 dark:text-red-400',
    border: 'border-red-200 dark:border-red-800',
    icon: <XCircleIcon className="w-5 h-5" />,
  },
  warning: {
    bg: 'bg-amber-50 dark:bg-amber-900/20',
    text: 'text-amber-700 dark:text-amber-400',
    border: 'border-amber-200 dark:border-amber-800',
    icon: <AlertIcon className="w-5 h-5" />,
  },
  info: {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    text: 'text-blue-700 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-800',
    icon: <AlertIcon className="w-5 h-5" />,
  },
  none: {
    bg: 'bg-gray-50 dark:bg-gray-900/20',
    text: 'text-gray-700 dark:text-gray-400',
    border: 'border-gray-200 dark:border-gray-800',
    icon: <ClockIcon className="w-5 h-5" />,
  },
};

export const AlertPanel = ({ alerts, onSilence, onView }: AlertPanelProps) => {
  const [filter, setFilter] = useState<AlertSeverity | 'all'>('all');
  const [stateFilter, setStateFilter] = useState<'firing' | 'pending' | 'resolved' | 'all'>('firing');

  const filteredAlerts = alerts.filter((alert) => {
    if (filter !== 'all' && alert.labels.severity !== filter) return false;
    if (stateFilter !== 'all' && alert.state !== stateFilter) return false;
    return true;
  });

  const groupedAlerts = filteredAlerts.reduce((acc, alert) => {
    const key = alert.service || alert.labels.alertname || 'other';
    if (!acc[key]) acc[key] = [];
    acc[key].push(alert);
    return acc;
  }, {} as Record<string, Alert[]>);

  const severityCounts = {
    critical: alerts.filter((a) => a.labels.severity === 'critical' && a.state === 'firing').length,
    warning: alerts.filter((a) => a.labels.severity === 'warning' && a.state === 'firing').length,
    info: alerts.filter((a) => a.labels.severity === 'info' && a.state === 'firing').length,
  };

  return (
    <div className="space-y-4">
      {/* Summary */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {alerts.filter((a) => a.state === 'firing').length}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400">Firing</p>
        </div>
        <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 border border-red-200 dark:border-red-800">
          <p className="text-2xl font-bold text-red-700 dark:text-red-400">{severityCounts.critical}</p>
          <p className="text-sm text-red-600 dark:text-red-500">Critical</p>
        </div>
        <div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-4 border border-amber-200 dark:border-amber-800">
          <p className="text-2xl font-bold text-amber-700 dark:text-amber-400">{severityCounts.warning}</p>
          <p className="text-sm text-amber-600 dark:text-amber-500">Warning</p>
        </div>
        <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4 border border-blue-200 dark:border-blue-800">
          <p className="text-2xl font-bold text-blue-700 dark:text-blue-400">{severityCounts.info}</p>
          <p className="text-sm text-blue-600 dark:text-blue-500">Info</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm text-gray-500 dark:text-gray-400">Severity:</span>
        {(['all', 'critical', 'warning', 'info', 'none'] as const).map((s) => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
              filter === s
                ? 'bg-gray-800 dark:bg-gray-700 text-white'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
        <span className="ml-4 text-sm text-gray-500 dark:text-gray-400">State:</span>
        {(['all', 'firing', 'pending', 'resolved'] as const).map((s) => (
          <button
            key={s}
            onClick={() => setStateFilter(s)}
            className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
              stateFilter === s
                ? 'bg-gray-800 dark:bg-gray-700 text-white'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {/* Alerts List */}
      {Object.entries(groupedAlerts).length === 0 ? (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-12 text-center border border-gray-200 dark:border-gray-700">
          <CheckIcon className="w-12 h-12 text-green-500 mx-auto mb-4" />
          <p className="text-gray-600 dark:text-gray-400">No alerts match your filters</p>
        </div>
      ) : (
        <div className="space-y-4">
          {Object.entries(groupedAlerts).map(([group, groupAlerts]) => (
            <div key={group} className="space-y-2">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 px-2">
                {group}
              </h3>
              {groupAlerts.map((alert) => {
                const severity = alert.labels.severity as AlertSeverity;
                const config = severityConfig[severity];

                return (
                  <div
                    key={alert.id}
                    className={`${config.bg} ${config.border} border rounded-lg p-4 cursor-pointer transition-all hover:shadow-md`}
                    onClick={() => onView?.(alert)}
                  >
                    <div className="flex items-start gap-3">
                      <div className={`${config.text} mt-0.5`}>{config.icon}</div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between gap-2">
                          <p className="font-medium text-gray-900 dark:text-gray-100 truncate">
                            {alert.labels.alertname || alert.name}
                          </p>
                          <span
                            className={`px-2 py-0.5 rounded text-xs font-medium ${
                              alert.state === 'firing'
                                ? 'bg-red-200 text-red-800 dark:bg-red-900 dark:text-red-200'
                                : alert.state === 'pending'
                                  ? 'bg-amber-200 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
                                  : 'bg-green-200 text-green-800 dark:bg-green-900 dark:text-green-200'
                            }`}
                          >
                            {alert.state}
                          </span>
                        </div>
                        <p className={`text-sm ${config.text} mt-1`}>{alert.message}</p>
                        <div className="flex items-center gap-4 mt-2 text-xs text-gray-500 dark:text-gray-400">
                          <span>Fired at {new Date(alert.startsAt).toLocaleString()}</span>
                          {alert.endsAt && <span>Ends at {new Date(alert.endsAt).toLocaleString()}</span>}
                        </div>
                        {Object.entries(alert.labels).length > 0 && (
                          <div className="flex flex-wrap gap-2 mt-2">
                            {Object.entries(alert.labels)
                              .filter(([key]) => key !== 'severity' && key !== 'alertname')
                              .slice(0, 5)
                              .map(([key, value]) => (
                                <span
                                  key={key}
                                  className="px-2 py-0.5 bg-white/50 dark:bg-black/20 rounded text-xs"
                                >
                                  {key}={value}
                                </span>
                              ))}
                          </div>
                        )}
                      </div>
                      {onSilence && alert.state === 'firing' && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onSilence(alert.id);
                          }}
                          className="p-2 hover:bg-white/50 dark:hover:bg-black/20 rounded-lg transition-colors"
                          title="Silence alert"
                        >
                          <svg className="w-4 h-4 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
                          </svg>
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

interface AlertSummaryCardProps {
  summary: {
    total: number;
    firing: number;
    pending: number;
    resolved: number;
  };
  onClick?: () => void;
}

export const AlertSummaryCard = ({ summary, onClick }: AlertSummaryCardProps) => {
  return (
    <div
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 cursor-pointer hover:shadow-md transition-all"
      onClick={onClick}
    >
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-gray-900 dark:text-gray-100">Alerts</h3>
        <AlertIcon className="w-5 h-5 text-amber-500" />
      </div>
      <div className="grid grid-cols-3 gap-4">
        <div>
          <p className="text-2xl font-bold text-red-600 dark:text-red-400">{summary.firing}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">Firing</p>
        </div>
        <div>
          <p className="text-2xl font-bold text-amber-600 dark:text-amber-400">{summary.pending}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">Pending</p>
        </div>
        <div>
          <p className="text-2xl font-bold text-green-600 dark:text-green-400">{summary.resolved}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">Resolved</p>
        </div>
      </div>
    </div>
  );
};
