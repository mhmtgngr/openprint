import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { TraceViewer, TraceSearch, MetricCard } from '@/components/observability';
import { tracingApi } from '@/services/monitoring';
import { TimeRange } from '@/types';
import { RefreshIcon, ActivityIcon, ExternalLinkIcon } from '@/components/icons';

const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: '1h', label: '1 hour' },
  { value: '3h', label: '3 hours' },
  { value: '6h', label: '6 hours' },
  { value: '12h', label: '12 hours' },
  { value: '24h', label: '24 hours' },
  { value: '7d', label: '7 days' },
];

type Tab = 'search' | 'trace' | 'summary';

export const ObservabilityHub = () => {
  const [activeTab, setActiveTab] = useState<Tab>('search');
  const [timeRange, setTimeRange] = useState<TimeRange>('1h');
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);
  const [selectedService, setSelectedService] = useState<string>('');
  const [autoRefresh, setAutoRefresh] = useState(false);

  // Fetch trace summary
  const { data: traceSummary } = useQuery({
    queryKey: ['traces', 'summary', selectedService, timeRange],
    queryFn: () => tracingApi.getTraceSummary(selectedService || undefined, timeRange),
    refetchInterval: autoRefresh ? 30000 : false,
  });

  // Fetch trace search results
  const { data: searchResults } = useQuery({
    queryKey: ['traces', 'search', selectedService, timeRange],
    queryFn: () =>
      tracingApi.searchTraces({
        service: selectedService || undefined,
        startTimeMin: new Date(Date.now() - parseTimeRange(timeRange)).toISOString(),
        limit: 50,
      }),
    refetchInterval: autoRefresh ? 30000 : false,
    enabled: activeTab === 'search',
  });

  // Fetch selected trace
  const { data: trace, isLoading: traceLoading } = useQuery({
    queryKey: ['trace', selectedTraceId],
    queryFn: () => tracingApi.getTrace(selectedTraceId!),
    enabled: !!selectedTraceId && activeTab === 'trace',
  });

  // Fetch available services
  const { data: services } = useQuery({
    queryKey: ['traces', 'services'],
    queryFn: () => tracingApi.getServices(),
  });

  const handleTraceSelect = (traceId: string) => {
    setSelectedTraceId(traceId);
    setActiveTab('trace');
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Observability Hub</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Distributed tracing and performance analysis
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
          <a
            href={import.meta.env.VITE_JAEGER_URL || 'http://localhost:16686'}
            target="_blank"
            rel="noopener noreferrer"
            className="px-4 py-2 bg-gray-800 dark:bg-gray-700 text-white rounded-lg font-medium hover:bg-gray-900 dark:hover:bg-gray-600 transition-colors flex items-center gap-2"
          >
            <ActivityIcon className="w-4 h-4" />
            Open Jaeger
            <ExternalLinkIcon className="w-4 h-4" />
          </a>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricCard
          label="Total Traces"
          value={traceSummary?.totalTraces || 0}
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
          }
          color="blue"
        />
        <MetricCard
          label="Error Traces"
          value={traceSummary?.errorTraces || 0}
          change={traceSummary?.totalTraces ? `${((traceSummary.errorTraces / traceSummary.totalTraces) * 100).toFixed(1)}%` : '0%'}
          changeType={traceSummary && traceSummary.errorTraces > 0 ? 'negative' : 'neutral'}
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          }
          color={traceSummary && traceSummary.errorTraces > 0 ? 'red' : 'green'}
        />
        <MetricCard
          label="Avg Duration"
          value={traceSummary?.avgDuration ? (traceSummary.avgDuration / 1000000).toFixed(2) : '0'}
          unit="ms"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          color="purple"
        />
        <MetricCard
          label="P99 Duration"
          value={traceSummary?.p99Duration ? (traceSummary.p99Duration / 1000000).toFixed(2) : '0'}
          unit="ms"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          }
          color="amber"
        />
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 flex-wrap bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Service:</span>
          <select
            value={selectedService}
            onChange={(e) => setSelectedService(e.target.value)}
            className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500"
          >
            <option value="">All Services</option>
            {services?.map((service) => (
              <option key={service} value={service}>
                {service}
              </option>
            ))}
          </select>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Time Range:</span>
          {TIME_RANGES.map(({ value, label }) => (
            <button
              key={value}
              onClick={() => setTimeRange(value)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                timeRange === value
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="border-b border-gray-200 dark:border-gray-700">
          <nav className="flex gap-1 px-4" aria-label="Tabs">
            {[
              { key: 'search' as Tab, label: 'Search Traces' },
              { key: 'trace' as Tab, label: 'View Trace', disabled: !selectedTraceId },
              { key: 'summary' as Tab, label: 'Summary' },
            ].map((tab) => (
              <button
                key={tab.key}
                onClick={() => !tab.disabled && setActiveTab(tab.key)}
                disabled={tab.disabled}
                className={`flex items-center gap-2 px-4 py-4 font-medium transition-colors relative ${
                  tab.disabled
                    ? 'text-gray-300 dark:text-gray-600 cursor-not-allowed'
                    : activeTab === tab.key
                      ? 'text-blue-600 dark:text-blue-400'
                      : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }`}
              >
                <span>{tab.label}</span>
                {activeTab === tab.key && !tab.disabled && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600 dark:bg-blue-400" />
                )}
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {activeTab === 'search' && (
            <TraceSearch
              onTraceSelect={handleTraceSelect}
              recentTraces={searchResults || []}
            />
          )}

          {activeTab === 'trace' && trace && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Trace ID</p>
                  <p className="font-mono text-sm text-gray-900 dark:text-gray-100">{trace.traceID}</p>
                </div>
                <a
                  href={`${import.meta.env.VITE_JAEGER_URL || 'http://localhost:16686'}/trace/${trace.traceID}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="px-4 py-2 bg-gray-800 dark:bg-gray-700 text-white rounded-lg font-medium hover:bg-gray-900 dark:hover:bg-gray-600 transition-colors flex items-center gap-2"
                >
                  View in Jaeger
                  <ExternalLinkIcon className="w-4 h-4" />
                </a>
              </div>
              <TraceViewer trace={trace} />
            </div>
          )}

          {activeTab === 'trace' && !trace && !traceLoading && (
            <div className="text-center py-12">
              <ActivityIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600 dark:text-gray-400">No trace selected</p>
              <p className="text-sm text-gray-500 dark:text-gray-500 mt-1">
                Select a trace from the search results to view details
              </p>
            </div>
          )}

          {activeTab === 'trace' && traceLoading && (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
            </div>
          )}

          {activeTab === 'summary' && traceSummary && (
            <div className="space-y-6">
              {/* Overall Stats */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4">
                  <p className="text-sm text-gray-500 dark:text-gray-400">Total Traces</p>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {traceSummary.totalTraces}
                  </p>
                </div>
                <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4">
                  <p className="text-sm text-red-600 dark:text-red-400">Error Traces</p>
                  <p className="text-2xl font-bold text-red-700 dark:text-red-300">
                    {traceSummary.errorTraces}
                  </p>
                </div>
                <div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-4">
                  <p className="text-sm text-amber-600 dark:text-amber-400">Slow Traces</p>
                  <p className="text-2xl font-bold text-amber-700 dark:text-amber-300">
                    {traceSummary.slowTraces}
                  </p>
                </div>
                <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
                  <p className="text-sm text-blue-600 dark:text-blue-400">Avg Duration</p>
                  <p className="text-2xl font-bold text-blue-700 dark:text-blue-300">
                    {(traceSummary.avgDuration / 1000000).toFixed(2)}ms
                  </p>
                </div>
              </div>

              {/* Service Breakdown */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
                  Service Breakdown
                </h3>
                <div className="space-y-3">
                  {Object.entries(traceSummary.byService).map(([service, stats]) => (
                    <div
                      key={service}
                      className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200 dark:border-gray-700"
                    >
                      <div className="flex items-center justify-between mb-3">
                        <p className="font-medium text-gray-900 dark:text-gray-100">{service}</p>
                        <div className="flex items-center gap-4 text-sm">
                          <span className="text-gray-500 dark:text-gray-400">
                            {stats.count} traces
                          </span>
                          {stats.errorCount > 0 && (
                            <span className="text-red-600 dark:text-red-400">
                              {stats.errorCount} errors
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="grid grid-cols-3 gap-4">
                        <div>
                          <p className="text-xs text-gray-500 dark:text-gray-400">Avg Duration</p>
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {(stats.avgDuration / 1000000).toFixed(2)}ms
                          </p>
                        </div>
                        <div>
                          <p className="text-xs text-gray-500 dark:text-gray-400">Max Duration</p>
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {(stats.maxDuration / 1000000).toFixed(2)}ms
                          </p>
                        </div>
                        <div>
                          <p className="text-xs text-gray-500 dark:text-gray-400">Error Rate</p>
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {stats.count > 0 ? ((stats.errorCount / stats.count) * 100).toFixed(1) : 0}%
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Grafana Dashboards Link */}
      <div className="bg-gradient-to-r from-purple-500 to-pink-500 rounded-xl p-6 text-white">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">Grafana Dashboards</h3>
            <p className="text-white/80 mt-1">
              View comprehensive dashboards with advanced visualization
            </p>
          </div>
          <a
            href={import.meta.env.VITE_GRAFANA_URL || 'http://localhost:3000'}
            target="_blank"
            rel="noopener noreferrer"
            className="px-6 py-3 bg-white text-purple-600 rounded-lg font-medium hover:bg-white/90 transition-colors flex items-center gap-2"
          >
            Open Grafana
            <ExternalLinkIcon className="w-4 h-4" />
          </a>
        </div>
      </div>
    </div>
  );
};

function parseTimeRange(range: TimeRange): number {
  const multipliers: Partial<Record<TimeRange, number>> = {
    '5m': 5 * 60 * 1000,
    '15m': 15 * 60 * 1000,
    '30m': 30 * 60 * 1000,
    '1h': 60 * 60 * 1000,
    '3h': 3 * 60 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '12h': 12 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
    '30d': 30 * 24 * 60 * 60 * 1000,
  };
  return multipliers[range] || 60 * 60 * 1000; // Default to 1 hour
}
