import { useEffect, useState, useRef } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { TimeRange } from '@/types';

interface RealTimeMetricsChartProps {
  title: string;
  query: string;
  color?: string;
  timeRange?: TimeRange;
  refreshInterval?: number;
  unit?: string;
  formatValue?: (value: number) => string;
  maxHeight?: number;
}

interface DataPoint {
  time: string;
  value: number;
}

const DEFAULT_COLORS = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // amber
  '#ef4444', // red
  '#8b5cf6', // purple
  '#06b6d4', // cyan
  '#ec4899', // pink
];

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
  return multipliers[range] || 60 * 60 * 1000;
}

function generateMockData(points: number, min: number, max: number): DataPoint[] {
  const now = Date.now();
  const interval = 15000; // 15 seconds
  const data: DataPoint[] = [];

  for (let i = points - 1; i >= 0; i--) {
    const timestamp = now - (i * interval);
    const time = new Date(timestamp).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });

    // Generate realistic-looking data with some randomness
    const baseValue = (max + min) / 2;
    const variance = (max - min) / 4;
    const value = baseValue + (Math.random() - 0.5) * variance * 2;

    data.push({ time, value: Math.max(min, Math.min(max, value)) });
  }

  return data;
}

export const RealTimeMetricsChart = ({
  title,
  query,
  color = DEFAULT_COLORS[0],
  timeRange = '15m',
  refreshInterval = 15000,
  unit,
  formatValue,
  maxHeight = 200,
}: RealTimeMetricsChartProps) => {
  const [data, setData] = useState<DataPoint[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<number | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  const fetchData = async (signal?: AbortSignal) => {
    try {
      setIsLoading(true);
      setError(null);

      const prometheusUrl = import.meta.env.VITE_PROMETHEUS_URL || 'http://localhost:9090';

      // Try to fetch from Prometheus first
      const endTime = Math.floor(Date.now() / 1000);
      const startTime = Math.floor((Date.now() - parseTimeRange(timeRange)) / 1000);
      const step = '15s';

      const response = await fetch(
        `${prometheusUrl}/api/v1/query_range?query=${encodeURIComponent(query)}&start=${startTime}&end=${endTime}&step=${step}`,
        { signal }
      );

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result = await response.json();

      if (result.status === 'success' && result.data?.result?.[0]?.values) {
        const chartData = result.data.result[0].values.map(([timestamp, value]: [number, string]) => ({
          time: new Date(timestamp * 1000).toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
          }),
          value: parseFloat(value),
        }));
        setData(chartData);
      } else {
        // Fallback to mock data for development
        setData(generateMockData(60, 0, 100));
      }
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        console.warn(`Failed to fetch metrics for "${query}":`, err);
        setError((err as Error).message);

        // Use mock data as fallback
        setData(generateMockData(60, 0, 100));
      }
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();

    // Set up auto-refresh
    intervalRef.current = window.setInterval(() => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      abortControllerRef.current = new AbortController();
      fetchData(abortControllerRef.current.signal);
    }, refreshInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [query, timeRange, refreshInterval]);

  const currentValue = data.length > 0 ? data[data.length - 1].value : 0;
  const previousValue = data.length > 1 ? data[data.length - 2].value : currentValue;
  const change = previousValue !== 0 ? ((currentValue - previousValue) / previousValue) * 100 : 0;
  const isPositive = change >= 0;

  const formatTooltipValue = (value: number) => {
    if (formatValue) return formatValue(value);
    return unit ? `${value.toFixed(2)}${unit}` : value.toFixed(2);
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{title}</h3>
          {data.length > 0 && (
            <p className="text-2xl font-bold text-gray-900 dark:text-gray-100 mt-1">
              {formatTooltipValue(currentValue)}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {isLoading && (
            <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
          )}
          {!isLoading && data.length > 0 && (
            <span
              className={`text-xs font-medium flex items-center gap-1 ${
                isPositive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
              }`}
            >
              {isPositive ? '↑' : '↓'}
              {Math.abs(change).toFixed(1)}%
            </span>
          )}
        </div>
      </div>

      {/* Error indicator */}
      {error && (
        <div className="mb-2 px-2 py-1 bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-400 text-xs rounded">
          Using demo data
        </div>
      )}

      {/* Chart */}
      <div style={{ height: `${maxHeight}px` }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 5, right: 5, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id={`gradient-${color.replace('#', '')}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                <stop offset="95%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
            <XAxis
              dataKey="time"
              className="text-gray-500 dark:text-gray-400"
              tick={{ fontSize: 10 }}
              tickFormatter={(value) => {
                // Show fewer ticks for readability
                const index = data.findIndex((d) => d.time === value);
                return index % Math.ceil(data.length / 6) === 0 ? value : '';
              }}
            />
            <YAxis
              className="text-gray-500 dark:text-gray-400"
              tick={{ fontSize: 10 }}
              tickFormatter={(value) => formatValue ? formatValue(Number(value)) : value.toFixed(0)}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'rgb(31 41 55)',
                border: 'none',
                borderRadius: '0.5rem',
                fontSize: '12px',
              }}
              formatter={(value: number) => [formatTooltipValue(value), 'Value']}
              labelFormatter={(label) => `Time: ${label}`}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke={color}
              fill={`url(#gradient-${color.replace('#', '')})`}
              strokeWidth={2}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      {/* Query info (for debugging) */}
      <p className="text-xs text-gray-400 dark:text-gray-500 mt-2 font-mono truncate" title={query}>
        {query}
      </p>
    </div>
  );
};

// Multi-metric variant
interface MultiMetricsChartProps {
  title: string;
  queries: Array<{ query: string; label: string; color?: string }>;
  timeRange?: TimeRange;
  refreshInterval?: number;
  maxHeight?: number;
}

export const MultiMetricsChart = ({
  title,
  queries,
  timeRange = '15m',
  refreshInterval = 15000,
  maxHeight = 200,
}: MultiMetricsChartProps) => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
      <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">{title}</h3>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {queries.map((q, i) => (
          <RealTimeMetricsChart
            key={i}
            title={q.label}
            query={q.query}
            color={q.color || DEFAULT_COLORS[i % DEFAULT_COLORS.length]}
            timeRange={timeRange}
            refreshInterval={refreshInterval}
            maxHeight={maxHeight}
          />
        ))}
      </div>
    </div>
  );
};
