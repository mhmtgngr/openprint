import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { MetricCard, AlertSummaryCard } from '@/components/observability';
import { prometheusApi, serviceHealthApi, alertsApi } from '@/services/monitoring';
import { TimeRange } from '@/types';
import { RefreshIcon, ChartIcon } from '@/components/icons';

const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: '5m', label: '5 min' },
  { value: '15m', label: '15 min' },
  { value: '30m', label: '30 min' },
  { value: '1h', label: '1 hour' },
  { value: '3h', label: '3 hours' },
  { value: '6h', label: '6 hours' },
  { value: '12h', label: '12 hours' },
  { value: '24h', label: '24 hours' },
];

const SERVICES = ['auth-service', 'registry-service', 'job-service', 'storage-service', 'notification-service'];

export const MetricsDashboard = () => {
  const [timeRange, setTimeRange] = useState<TimeRange>('1h');
  const [selectedService, setSelectedService] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);

  // Fetch service health
  const { data: services, isLoading: servicesLoading } = useQuery({
    queryKey: ['services', 'health'],
    queryFn: () => serviceHealthApi.getAllServices(),
    refetchInterval: autoRefresh ? 15000 : false,
  });

  // Fetch alert summary
  const { data: alertSummary } = useQuery({
    queryKey: ['alerts', 'summary'],
    queryFn: () => alertsApi.getAlertSummary(),
    refetchInterval: autoRefresh ? 30000 : false,
  });

  // Fetch request rate metrics
  const { data: requestRateData } = useQuery({
    queryKey: ['metrics', 'request-rate', timeRange, selectedService],
    queryFn: () => prometheusApi.getRequestRate(selectedService || undefined, timeRange),
    refetchInterval: autoRefresh ? 15000 : false,
    enabled: !!services,
  });

  // Fetch error rate metrics
  const { data: errorRateData } = useQuery({
    queryKey: ['metrics', 'error-rate', timeRange, selectedService],
    queryFn: () => prometheusApi.getErrorRate(selectedService || undefined, timeRange),
    refetchInterval: autoRefresh ? 15000 : false,
    enabled: !!services,
  });

  // Fetch latency metrics
  const { data: latencyData } = useQuery({
    queryKey: ['metrics', 'latency', timeRange, selectedService],
    queryFn: () => prometheusApi.getLatency(selectedService || undefined, timeRange),
    refetchInterval: autoRefresh ? 15000 : false,
    enabled: !!services,
  });

  // Transform Prometheus data to chart format
  const transformChartData = (result: any) => {
    if (!result?.data?.result) return [];

    const dataPoints: Record<string, any>[] = [];
    const series = result.data.result;

    // Find the time range from the first series
    if (series.length === 0) return [];

    const timestamps = series[0].values?.map((v: [number, string]) => v[0]) || [];

    timestamps.forEach((timestamp: number, i: number) => {
      const point: any = {
        time: new Date(timestamp * 1000).toLocaleTimeString(),
      };

      series.forEach((s: any) => {
        const label = s.metric.service || s.metric.instance || 'unknown';
        const value = s.values?.[i]?.[1];
        point[label] = value ? parseFloat(value) : 0;
      });

      dataPoints.push(point);
    });

    return dataPoints;
  };

  const requestRateChartData = transformChartData(requestRateData);
  const errorRateChartData = transformChartData(errorRateData);
  const latencyChartData = transformChartData(latencyData);

  // Calculate summary metrics
  const totalRequestRate = requestRateChartData.reduce((sum, point) => {
    const total = Object.values(point).filter((v) => typeof v === 'number').reduce((a: number, b) => a + (b as number), 0);
    return sum + total;
  }, 0) / Math.max(requestRateChartData.length, 1);

  const totalErrorRate = errorRateChartData.reduce((sum, point) => {
    const total = Object.values(point).filter((v) => typeof v === 'number').reduce((a: number, b) => a + (b as number), 0);
    return sum + total;
  }, 0) / Math.max(errorRateChartData.length, 1) * 100;

  const avgLatency = latencyChartData.length > 0
    ? latencyChartData[latencyChartData.length - 1]?.[Object.keys(latencyChartData[latencyChartData.length - 1])[1]] || 0
    : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Metrics Dashboard</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Real-time performance metrics for all OpenPrint services
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
          {TIME_RANGES.map(({ value, label }) => (
            <button
              key={value}
              onClick={() => setTimeRange(value)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                timeRange === value
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricCard
          label="Request Rate"
          value={totalRequestRate.toFixed(2)}
          unit="req/s"
          change={autoRefresh ? 'Live' : undefined}
          changeType="neutral"
          icon={<ChartIcon className="w-6 h-6" />}
          color="blue"
        />
        <MetricCard
          label="Error Rate"
          value={totalErrorRate.toFixed(2)}
          unit="%"
          change={totalErrorRate > 1 ? 'High' : totalErrorRate > 0.1 ? 'Normal' : 'Low'}
          changeType={totalErrorRate > 1 ? 'negative' : totalErrorRate > 0.1 ? 'neutral' : 'positive'}
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          }
          color={totalErrorRate > 1 ? 'red' : totalErrorRate > 0.1 ? 'amber' : 'green'}
        />
        <MetricCard
          label="P95 Latency"
          value={avgLatency.toFixed(0)}
          unit="ms"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          }
          color="purple"
        />
        {alertSummary && (
          <AlertSummaryCard
            summary={{
              total: alertSummary.total,
              firing: alertSummary.firing,
              pending: alertSummary.pending,
              resolved: alertSummary.resolved,
            }}
            onClick={() => window.location.assign('/monitoring')}
          />
        )}
      </div>

      {/* Service Selector */}
      <div className="flex items-center gap-2 flex-wrap bg-white dark:bg-gray-800 rounded-lg p-3 border border-gray-200 dark:border-gray-700">
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Services:</span>
        <button
          onClick={() => setSelectedService(null)}
          className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            selectedService === null
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
          }`}
        >
          All Services
        </button>
        {SERVICES.map((service) => (
          <button
            key={service}
            onClick={() => setSelectedService(service)}
            className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
              selectedService === service
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
            }`}
          >
            {service}
          </button>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Request Rate Chart */}
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
            Request Rate
          </h3>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={requestRateChartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
              <XAxis dataKey="time" className="text-gray-500 dark:text-gray-400" />
              <YAxis className="text-gray-500 dark:text-gray-400" />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'rgb(31 41 55)',
                  border: 'none',
                  borderRadius: '0.5rem',
                }}
              />
              <Legend />
              {requestRateChartData.length > 0 &&
                Object.keys(requestRateChartData[0])
                  .filter((k) => k !== 'time')
                  .map((key, i) => (
                    <Area
                      key={key}
                      type="monotone"
                      dataKey={key}
                      stackId="1"
                      stroke={['#3b82f6', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899'][i % 5]}
                      fill={['#3b82f6', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899'][i % 5]}
                      fillOpacity={0.6}
                    />
                  ))}
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Error Rate Chart */}
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
            Error Rate
          </h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={errorRateChartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
              <XAxis dataKey="time" className="text-gray-500 dark:text-gray-400" />
              <YAxis className="text-gray-500 dark:text-gray-400" />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'rgb(31 41 55)',
                  border: 'none',
                  borderRadius: '0.5rem',
                }}
              />
              <Legend />
              {errorRateChartData.length > 0 &&
                Object.keys(errorRateChartData[0])
                  .filter((k) => k !== 'time')
                  .map((key, i) => (
                    <Line
                      key={key}
                      type="monotone"
                      dataKey={key}
                      stroke={['#ef4444', '#f59e0b', '#8b5cf6'][i % 3]}
                      strokeWidth={2}
                      dot={{ fill: ['#ef4444', '#f59e0b', '#8b5cf6'][i % 3] }}
                    />
                  ))}
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Latency Chart */}
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 lg:col-span-2">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
            P95 Latency
          </h3>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={latencyChartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
              <XAxis dataKey="time" className="text-gray-500 dark:text-gray-400" />
              <YAxis className="text-gray-500 dark:text-gray-400" />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'rgb(31 41 55)',
                  border: 'none',
                  borderRadius: '0.5rem',
                }}
              />
              <Legend />
              {latencyChartData.length > 0 &&
                Object.keys(latencyChartData[0])
                  .filter((k) => k !== 'time')
                  .map((key, i) => (
                    <Area
                      key={key}
                      type="monotone"
                      dataKey={key}
                      stroke={['#8b5cf6', '#06b6d4', '#f59e0b'][i % 3]}
                      fill={['#8b5cf6', '#06b6d4', '#f59e0b'][i % 3]}
                      fillOpacity={0.3}
                    />
                  ))}
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Services Health Grid */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">Service Health</h2>
        {servicesLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="bg-white dark:bg-gray-800 rounded-xl p-6 animate-pulse" />
            ))}
          </div>
        ) : services && services.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {services.map((service) => (
              <div
                key={service.serviceName + service.instance}
                className={`bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border ${
                  service.status === 'healthy'
                    ? 'border-green-200 dark:border-green-800'
                    : service.status === 'degraded'
                      ? 'border-amber-200 dark:border-amber-800'
                      : 'border-red-200 dark:border-red-800'
                }`}
              >
                <div className="flex items-center justify-between mb-3">
                  <h4 className="font-medium text-gray-900 dark:text-gray-100">{service.serviceName}</h4>
                  <span
                    className={`px-2 py-1 rounded text-xs font-medium ${
                      service.status === 'healthy'
                        ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                        : service.status === 'degraded'
                          ? 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400'
                          : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'
                    }`}
                  >
                    {service.status}
                  </span>
                </div>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">CPU:</span>
                    <span className="text-gray-900 dark:text-gray-100">{service.metrics.cpuPercent.toFixed(1)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">Memory:</span>
                    <span className="text-gray-900 dark:text-gray-100">{service.metrics.memoryPercent.toFixed(1)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">Request Rate:</span>
                    <span className="text-gray-900 dark:text-gray-100">{service.metrics.requestRate.toFixed(1)}/s</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-500 dark:text-gray-400">P95 Latency:</span>
                    <span className="text-gray-900 dark:text-gray-100">{service.metrics.latency.p95}ms</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="bg-white dark:bg-gray-800 rounded-xl p-12 text-center border border-gray-200 dark:border-gray-700">
            <p className="text-gray-500 dark:text-gray-400">No service health data available</p>
          </div>
        )}
      </div>
    </div>
  );
};
