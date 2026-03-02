import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { SearchIcon, PlayIcon, BookOpenIcon } from '@/components/icons';
import { prometheusApi } from '@/services/monitoring';
import { TimeRange, QueryResult } from '@/types';

interface PrometheusQueryBuilderProps {
  onQueryExecute?: (query: string, result: QueryResult) => void;
  savedQueries?: Array<{ name: string; query: string }>;
}

const COMMON_QUERIES = [
  {
    category: 'HTTP Metrics',
    queries: [
      { name: 'Request Rate', query: 'sum(rate(http_requests_total[5m])) by (service)' },
      { name: 'Error Rate', query: 'sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)' },
      { name: 'P95 Latency', query: 'histogram_quantile(0.95, sum(rate(http_request_duration_ms_bucket[5m])) by (le, service))' },
      { name: 'Active Connections', query: 'sum(http_requests_in_progress) by (service)' },
    ],
  },
  {
    category: 'Database Metrics',
    queries: [
      { name: 'DB Connections', query: 'sum(pg_stat_activity_count{datname="openprint"}) by (service)' },
      { name: 'DB Query Duration', query: 'sum(rate(db_query_duration_seconds_sum[5m])) by (service, operation)' },
      { name: 'DB Errors', query: 'sum(rate(db_errors_total[5m])) by (service, error_type)' },
    ],
  },
  {
    category: 'Redis Metrics',
    queries: [
      { name: 'Redis Commands', query: 'sum(rate(redis_commands_total[5m])) by (service, command)' },
      { name: 'Redis Hit Rate', query: 'rate(redis_key_hits[5m]) / (rate(redis_key_hits[5m]) + rate(redis_key_misses[5m]))' },
      { name: 'Redis Connections', query: 'sum(redis_connected_clients) by (service)' },
    ],
  },
  {
    category: 'System Resources',
    queries: [
      { name: 'CPU Usage', query: 'sum(rate(process_cpu_seconds_total[5m])) by (service) * 100' },
      { name: 'Memory Usage', query: 'sum(process_resident_memory_bytes) by (service) / 1024 / 1024' },
      { name: 'Disk I/O', query: 'sum(rate(disk_io_bytes[5m])) by (service, device)' },
    ],
  },
];

export const PrometheusQueryBuilder = ({
  onQueryExecute,
  savedQueries = [],
}: PrometheusQueryBuilderProps) => {
  const [query, setQuery] = useState('');
  const [result, setResult] = useState<QueryResult | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [showQueryLibrary, setShowQueryLibrary] = useState(true);

  const executeQuery = async (queryToExecute?: string) => {
    const q = queryToExecute || query;
    if (!q.trim()) return;

    setIsLoading(true);
    setError(null);

    try {
      const response = await prometheusApi.query(q);
      setResult(response);
      onQueryExecute?.(q, response);
    } catch (err) {
      setError((err as Error).message);
      setResult(null);
    } finally {
      setIsLoading(false);
    }
  };

  const handleQuickQuery = (q: string) => {
    setQuery(q);
    executeQuery(q);
  };

  const formatResultValue = (value: string | number): string => {
    if (typeof value === 'number') {
      return value >= 1000000
        ? `${(value / 1000000).toFixed(2)}M`
        : value >= 1000
          ? `${(value / 1000).toFixed(2)}K`
          : value.toFixed(2);
    }
    return String(value);
  };

  return (
    <div className="space-y-4">
      {/* Query Input */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">Prometheus Query Builder</h3>
          <button
            onClick={() => setShowQueryLibrary(!showQueryLibrary)}
            className="text-sm text-blue-600 dark:text-blue-400 hover:underline flex items-center gap-1"
          >
            <BookOpenIcon className="w-4 h-4" />
            {showQueryLibrary ? 'Hide' : 'Show'} Library
          </button>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex-1 relative">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && executeQuery()}
              placeholder="Enter PromQL query..."
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
            />
          </div>
          <button
            onClick={() => executeQuery()}
            disabled={isLoading || !query.trim()}
            className="px-4 py-2 bg-blue-600 text-white hover:bg-blue-700 disabled:bg-gray-300 dark:disabled:bg-gray-700 rounded-lg font-medium flex items-center gap-2 transition-colors"
          >
            {isLoading ? (
              <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : (
              <PlayIcon className="w-4 h-4" />
            )}
            Run
          </button>
        </div>

        {/* Saved Queries */}
        {savedQueries.length > 0 && (
          <div className="mt-3">
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">Saved Queries:</p>
            <div className="flex flex-wrap gap-2">
              {savedQueries.map((sq, i) => (
                <button
                  key={i}
                  onClick={() => handleQuickQuery(sq.query)}
                  className="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg text-sm font-medium transition-colors"
                >
                  {sq.name}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Query Library */}
      {showQueryLibrary && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
          <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Query Library</h4>

          {/* Category Tabs */}
          <div className="flex items-center gap-2 mb-4 overflow-x-auto pb-2">
            {COMMON_QUERIES.map((category) => (
              <button
                key={category.category}
                onClick={() => setSelectedCategory(category.category === selectedCategory ? null : category.category)}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium whitespace-nowrap transition-colors ${
                  selectedCategory === category.category
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                }`}
              >
                {category.category}
              </button>
            ))}
          </div>

          {/* Queries for selected category */}
          {selectedCategory ? (
            <div className="space-y-2">
              {COMMON_QUERIES.find((c) => c.category === selectedCategory)?.queries.map((q) => (
                <div
                  key={q.name}
                  className="p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-700 cursor-pointer transition-colors"
                  onClick={() => handleQuickQuery(q.query)}
                >
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{q.name}</p>
                    <PlayIcon className="w-4 h-4 text-gray-400" />
                  </div>
                  <p className="text-xs font-mono text-gray-500 dark:text-gray-400 truncate">{q.query}</p>
                </div>
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {COMMON_QUERIES.map((category) => (
                <div key={category.category} className="space-y-2">
                  <p className="text-sm font-medium text-gray-700 dark:text-gray-300">{category.category}</p>
                  {category.queries.slice(0, 2).map((q) => (
                    <div
                      key={q.name}
                      className="p-2 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-700 cursor-pointer transition-colors"
                      onClick={() => handleQuickQuery(q.query)}
                    >
                      <p className="text-xs font-medium text-gray-900 dark:text-gray-100">{q.name}</p>
                      <p className="text-xs font-mono text-gray-500 dark:text-gray-400 truncate">{q.query}</p>
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-sm font-medium text-red-700 dark:text-red-400">Query Error</p>
          <p className="text-sm text-red-600 dark:text-red-500 mt-1">{error}</p>
        </div>
      )}

      {/* Results Display */}
      {result && result.status === 'success' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">Query Results</h4>
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {result.data.result.length} result{result.data.result.length !== 1 ? 's' : ''}
            </span>
          </div>

          {result.data.resultType === 'vector' && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-200 dark:border-gray-700">
                    <th className="text-left py-2 px-3 text-gray-500 dark:text-gray-400 font-medium">Metric</th>
                    <th className="text-right py-2 px-3 text-gray-500 dark:text-gray-400 font-medium">Value</th>
                  </tr>
                </thead>
                <tbody>
                  {result.data.result.map((item, i) => (
                    <tr key={i} className="border-b border-gray-100 dark:border-gray-800">
                      <td className="py-2 px-3">
                        <div className="flex flex-wrap gap-1">
                          {Object.entries(item.metric).map(([key, value]) => (
                            <span
                              key={key}
                              className="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono"
                            >
                              {key}={value}
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="py-2 px-3 text-right font-mono text-gray-900 dark:text-gray-100">
                        {formatResultValue(parseFloat(item.value?.[1] || '0'))}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {result.data.resultType === 'matrix' && (
            <div className="text-sm text-gray-500 dark:text-gray-400">
              <p>Range query returned time-series data. Use the Metrics Dashboard for visualization.</p>
              <p className="mt-2">
                {result.data.result.length} series returned with{' '}
                {result.data.result[0]?.values?.length || 0} data points each.
              </p>
            </div>
          )}

          {result.data.resultType === 'scalar' && (
            <div className="text-center py-4">
              <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">
                {formatResultValue(parseFloat(result.data.result[0]?.value?.[1] || '0'))}
              </p>
            </div>
          )}
        </div>
      )}

      {/* Empty state */}
      {!result && !isLoading && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-8 shadow-sm border border-gray-200 dark:border-gray-700 text-center">
          <SearchIcon className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-600 dark:text-gray-400">Enter a PromQL query to see results</p>
          <p className="text-sm text-gray-500 dark:text-gray-500 mt-1">
            Select a query from the library or write your own
          </p>
        </div>
      )}
    </div>
  );
};

// Compact variant for embedding in other components
interface CompactQueryRunnerProps {
  query: string;
  timeRange?: TimeRange;
}

export const CompactQueryRunner = ({ query, timeRange }: CompactQueryRunnerProps) => {
  const { data, isLoading, error } = useQuery({
    queryKey: ['prometheus', 'query', query, timeRange],
    queryFn: () => prometheusApi.queryRange(query, timeRange || '1h'),
    refetchInterval: 15000,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-4">
        <div className="w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (error || !data || data.status !== 'success') {
    return (
      <div className="text-center p-4 text-gray-500 dark:text-gray-400">
        <span className="text-red-500">Error loading data</span>
      </div>
    );
  }

  const latestValue = data.data.result[0]?.values?.slice(-1)?.[0]?.[1];

  return (
    <div className="text-center">
      <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
        {latestValue ? parseFloat(latestValue).toFixed(2) : 'N/A'}
      </p>
    </div>
  );
};
