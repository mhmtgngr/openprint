import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertPanel, ServiceHealthList } from '@/components/observability';
import { alertsApi, serviceHealthApi } from '@/services/monitoring';
import { Alert, ServiceHealth } from '@/types';
import { RefreshIcon, BellIcon, HeartIcon } from '@/components/icons';

type Tab = 'alerts' | 'services' | 'silences';

export const Monitoring = () => {
  const [activeTab, setActiveTab] = useState<Tab>('alerts');
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [selectedAlert, setSelectedAlert] = useState<Alert | null>(null);
  const [selectedService, setSelectedService] = useState<ServiceHealth | null>(null);

  // Fetch alerts
  const { data: alerts, refetch: refetchAlerts } = useQuery({
    queryKey: ['alerts'],
    queryFn: () => alertsApi.getAlerts(),
    refetchInterval: autoRefresh ? 30000 : false,
  });

  // Fetch service health
  const { data: services, refetch: refetchServices } = useQuery({
    queryKey: ['services', 'health'],
    queryFn: () => serviceHealthApi.getAllServices(),
    refetchInterval: autoRefresh ? 15000 : false,
  });

  // Fetch silences
  const { data: silences } = useQuery({
    queryKey: ['silences'],
    queryFn: () => alertsApi.getSilences(),
    refetchInterval: autoRefresh ? 60000 : false,
  });

  const handleSilenceAlert = async (alertId: string) => {
    const alert = alerts?.find((a) => a.id === alertId);
    if (!alert) return;

    try {
      await alertsApi.createSilence(
        Object.entries(alert.labels).map(([name, value]) => ({ name, value, isRegex: false })),
        '1h',
        `Silenced from dashboard`,
        'dashboard-user'
      );
      await refetchAlerts();
    } catch (error) {
      console.error('Failed to silence alert:', error);
    }
  };

  const handleRefreshAll = () => {
    refetchAlerts();
    refetchServices();
  };

  const firingCount = alerts?.filter((a) => a.state === 'firing').length || 0;
  const unhealthyCount = services?.filter((s) => s.status === 'unhealthy').length || 0;
  const degradedCount = services?.filter((s) => s.status === 'degraded').length || 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Monitoring</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Real-time alerts and service health monitoring
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`p-2 rounded-lg transition-colors ${
              autoRefresh
                ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'
            }`}
            title={autoRefresh ? 'Auto-refresh on' : 'Auto-refresh off'}
          >
            <RefreshIcon className={`w-5 h-5 ${autoRefresh ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={handleRefreshAll}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors flex items-center gap-2"
          >
            <RefreshIcon className="w-4 h-4" />
            Refresh
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Total Alerts</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {alerts?.length || 0}
              </p>
            </div>
            <BellIcon className="w-8 h-8 text-amber-500 opacity-80" />
          </div>
        </div>
        <div className="bg-red-50 dark:bg-red-900/20 rounded-xl p-6 shadow-sm border border-red-200 dark:border-red-800">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-red-600 dark:text-red-400">Firing</p>
              <p className="text-2xl font-bold text-red-700 dark:text-red-300">
                {firingCount}
              </p>
            </div>
            <div className="w-8 h-8 bg-red-500 rounded-full flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Services</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {services?.length || 0}
              </p>
            </div>
            <HeartIcon className="w-8 h-8 text-blue-500 opacity-80" />
          </div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Health Issues</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {unhealthyCount + degradedCount}
              </p>
            </div>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
              unhealthyCount > 0
                ? 'bg-red-500'
                : degradedCount > 0
                  ? 'bg-amber-500'
                  : 'bg-green-500'
            }`}>
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="border-b border-gray-200 dark:border-gray-700">
          <nav className="flex gap-1 px-4" aria-label="Tabs">
            {[
              { key: 'alerts' as Tab, label: 'Alerts', icon: BellIcon, count: firingCount },
              { key: 'services' as Tab, label: 'Services', icon: HeartIcon, count: unhealthyCount },
              { key: 'silences' as Tab, label: 'Silences', icon: () => null, count: silences?.length || 0 },
            ].map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex items-center gap-2 px-4 py-4 font-medium transition-colors relative ${
                  activeTab === tab.key
                    ? 'text-blue-600 dark:text-blue-400'
                    : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }`}
              >
                <tab.icon className="w-5 h-5" />
                <span>{tab.label}</span>
                {tab.count > 0 && (
                  <span
                    className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                      activeTab === tab.key
                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                    }`}
                  >
                    {tab.count}
                  </span>
                )}
                {activeTab === tab.key && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 dark:bg-blue-400" />
                )}
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {activeTab === 'alerts' && (
            <AlertPanel
              alerts={alerts || []}
              onSilence={handleSilenceAlert}
              onView={setSelectedAlert}
            />
          )}

          {activeTab === 'services' && (
            <ServiceHealthList
              services={services || []}
              onServiceClick={setSelectedService}
            />
          )}

          {activeTab === 'silences' && (
            <div className="space-y-4">
              {silences && silences.length > 0 ? (
                silences.map((silence) => (
                  <div
                    key={silence.id}
                    className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200 dark:border-gray-700"
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-medium text-gray-900 dark:text-gray-100">
                          {silence.comment}
                        </p>
                        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                          By {silence.createdBy}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                          {new Date(silence.endsAt).toLocaleString()}
                        </p>
                        <button
                          onClick={() => alertsApi.deleteSilence(silence.id).then(() => window.location.reload())}
                          className="text-red-600 dark:text-red-400 text-sm hover:underline"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2 mt-3">
                      {silence.matchers.map((matcher, i) => (
                        <span
                          key={i}
                          className="px-2 py-1 bg-white dark:bg-gray-800 rounded text-sm"
                        >
                          {matcher.isRegex ? '' : ''}{matcher.name}={matcher.value}
                        </span>
                      ))}
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-12">
                  <svg className="w-12 h-12 text-gray-400 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                  </svg>
                  <p className="text-gray-600 dark:text-gray-400">No active silences</p>
                  <p className="text-sm text-gray-500 dark:text-gray-500 mt-1">
                    Silences temporarily suppress alert notifications
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Alert Detail Modal */}
      {selectedAlert && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={() => setSelectedAlert(null)}
        >
          <div
            className="bg-white dark:bg-gray-800 rounded-xl max-w-2xl w-full max-h-[80vh] overflow-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {selectedAlert.labels.alertname || selectedAlert.name}
                </h3>
                <button
                  onClick={() => setSelectedAlert(null)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">Message</p>
                <p className="text-gray-900 dark:text-gray-100 mt-1">{selectedAlert.message}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">State</p>
                <span
                  className={`inline-block px-2 py-1 rounded text-sm font-medium mt-1 ${
                    selectedAlert.state === 'firing'
                      ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                      : selectedAlert.state === 'pending'
                        ? 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
                        : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                  }`}
                >
                  {selectedAlert.state}
                </span>
              </div>
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">Time</p>
                <p className="text-gray-900 dark:text-gray-100 mt-1">
                  Started: {new Date(selectedAlert.startsAt).toLocaleString()}
                </p>
                {selectedAlert.endsAt && (
                  <p className="text-gray-900 dark:text-gray-100">
                    Ends: {new Date(selectedAlert.endsAt).toLocaleString()}
                  </p>
                )}
              </div>
              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400">Labels</p>
                <div className="mt-2 space-y-1">
                  {Object.entries(selectedAlert.labels).map(([key, value]) => (
                    <div
                      key={key}
                      className="flex items-center justify-between px-3 py-2 bg-gray-50 dark:bg-gray-900/50 rounded"
                    >
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{key}</span>
                      <span className="text-sm text-gray-900 dark:text-gray-100 font-mono">{value}</span>
                    </div>
                  ))}
                </div>
              </div>
              {selectedAlert.annotations && Object.keys(selectedAlert.annotations).length > 0 && (
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Annotations</p>
                  <div className="mt-2 space-y-1">
                    {Object.entries(selectedAlert.annotations).map(([key, value]) => (
                      <div
                        key={key}
                        className="flex items-center justify-between px-3 py-2 bg-gray-50 dark:bg-gray-900/50 rounded"
                      >
                        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{key}</span>
                        <span className="text-sm text-gray-900 dark:text-gray-100">{value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
            <div className="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
              <button
                onClick={() => setSelectedAlert(null)}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                Close
              </button>
              <a
                href={selectedAlert.generatorURL}
                target="_blank"
                rel="noopener noreferrer"
                className="px-4 py-2 bg-blue-600 text-white hover:bg-blue-700 rounded-lg transition-colors"
              >
                View in Prometheus
              </a>
            </div>
          </div>
        </div>
      )}

      {/* Service Detail Modal */}
      {selectedService && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={() => setSelectedService(null)}
        >
          <div
            className="bg-white dark:bg-gray-800 rounded-xl max-w-2xl w-full max-h-[80vh] overflow-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {selectedService.serviceName}
                </h3>
                <button
                  onClick={() => setSelectedService(null)}
                  className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
            <div className="p-6 space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Status</p>
                  <span
                    className={`inline-block px-2 py-1 rounded text-sm font-medium mt-1 capitalize ${
                      selectedService.status === 'healthy'
                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                        : selectedService.status === 'degraded'
                          ? 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
                          : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                    }`}
                  >
                    {selectedService.status}
                  </span>
                </div>
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Version</p>
                  <p className="text-gray-900 dark:text-gray-100 mt-1">{selectedService.version}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Instance</p>
                  <p className="text-gray-900 dark:text-gray-100 mt-1 font-mono text-sm">{selectedService.instance}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Uptime</p>
                  <p className="text-gray-900 dark:text-gray-100 mt-1">
                    {Math.floor(selectedService.uptime / 3600000)}h {Math.floor((selectedService.uptime % 3600000) / 60000)}m
                  </p>
                </div>
              </div>

              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Resource Usage</p>
                <div className="space-y-3">
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-gray-700 dark:text-gray-300">CPU</span>
                      <span className="text-gray-900 dark:text-gray-100">{selectedService.metrics.cpuPercent.toFixed(1)}%</span>
                    </div>
                    <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full">
                      <div
                        className={`h-full rounded-full ${
                          selectedService.metrics.cpuPercent >= 90
                            ? 'bg-red-500'
                            : selectedService.metrics.cpuPercent >= 70
                              ? 'bg-amber-500'
                              : 'bg-green-500'
                        }`}
                        style={{ width: `${Math.min(selectedService.metrics.cpuPercent, 100)}%` }}
                      />
                    </div>
                  </div>
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-gray-700 dark:text-gray-300">Memory</span>
                      <span className="text-gray-900 dark:text-gray-100">{selectedService.metrics.memoryPercent.toFixed(1)}%</span>
                    </div>
                    <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full">
                      <div
                        className={`h-full rounded-full ${
                          selectedService.metrics.memoryPercent >= 90
                            ? 'bg-red-500'
                            : selectedService.metrics.memoryPercent >= 70
                              ? 'bg-amber-500'
                              : 'bg-green-500'
                        }`}
                        style={{ width: `${Math.min(selectedService.metrics.memoryPercent, 100)}%` }}
                      />
                    </div>
                  </div>
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-gray-700 dark:text-gray-300">Disk</span>
                      <span className="text-gray-900 dark:text-gray-100">{selectedService.metrics.diskPercent.toFixed(1)}%</span>
                    </div>
                    <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full">
                      <div
                        className={`h-full rounded-full ${
                          selectedService.metrics.diskPercent >= 90
                            ? 'bg-red-500'
                            : selectedService.metrics.diskPercent >= 70
                              ? 'bg-amber-500'
                              : 'bg-green-500'
                        }`}
                        style={{ width: `${Math.min(selectedService.metrics.diskPercent, 100)}%` }}
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div>
                <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Performance</p>
                <div className="grid grid-cols-3 gap-4">
                  <div className="text-center p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                    <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                      {selectedService.metrics.requestRate.toFixed(1)}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">req/s</p>
                  </div>
                  <div className="text-center p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                    <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                      {selectedService.metrics.errorRate.toFixed(2)}%
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">error rate</p>
                  </div>
                  <div className="text-center p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                    <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                      {selectedService.metrics.latency.p95}ms
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">P95 latency</p>
                  </div>
                </div>
              </div>

              {selectedService.dependencies && selectedService.dependencies.length > 0 && (
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Dependencies</p>
                  <div className="space-y-2">
                    {selectedService.dependencies.map((dep, i) => (
                      <div
                        key={i}
                        className={`flex items-center justify-between px-4 py-3 rounded-lg border ${
                          dep.status === 'healthy'
                            ? 'border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-900/20'
                            : dep.status === 'degraded'
                              ? 'border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-900/20'
                              : 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20'
                        }`}
                      >
                        <div>
                          <p className="font-medium text-gray-900 dark:text-gray-100">{dep.name}</p>
                          <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">{dep.type}</p>
                        </div>
                        <div className="text-right">
                          <p className="text-sm font-medium capitalize text-gray-900 dark:text-gray-100">{dep.status}</p>
                          {dep.latency && (
                            <p className="text-xs text-gray-500 dark:text-gray-400">{dep.latency}ms</p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
