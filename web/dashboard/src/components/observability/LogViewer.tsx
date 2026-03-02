import { useState, useEffect, useRef } from 'react';
import { SearchIcon, ChevronDownIcon, ChevronRightIcon } from '@/components/icons';

export interface LogEntry {
  id: string;
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  service: string;
  message: string;
  fields?: Record<string, string>;
  stackTrace?: string;
}

interface LogViewerProps {
  logs: LogEntry[];
  onRefresh?: () => void;
  autoRefresh?: boolean;
  onAutoRefreshChange?: (enabled: boolean) => void;
  maxHeight?: string;
}

const LOG_LEVEL_COLORS = {
  debug: 'text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900/20 border-gray-200 dark:border-gray-700',
  info: 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800',
  warn: 'text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800',
  error: 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800',
};

const LOG_LEVEL_ICONS = {
  debug: (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <circle cx="12" cy="12" r="10" strokeWidth="2" strokeDasharray="4 2" />
    </svg>
  ),
  info: (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <circle cx="12" cy="12" r="10" strokeWidth="2" />
      <path d="M12 16v-4M12 8h.01" strokeWidth="2" strokeLinecap="round" />
    </svg>
  ),
  warn: (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
    </svg>
  ),
  error: (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  ),
};

export const LogViewer = ({
  logs,
  onRefresh,
  autoRefresh = false,
  onAutoRefreshChange,
  maxHeight = '600px',
}: LogViewerProps) => {
  const [filter, setFilter] = useState('');
  const [levelFilter, setLevelFilter] = useState<Set<keyof typeof LOG_LEVEL_COLORS>>(new Set(['info', 'warn', 'error']));
  const [serviceFilter, setServiceFilter] = useState<string>('');
  const [expandedEntries, setExpandedEntries] = useState<Set<string>>(new Set());
  const [followMode, setFollowMode] = useState(true);
  const containerRef = useRef<HTMLDivElement>(null);

  const filteredLogs = logs.filter((log) => {
    if (filter && !log.message.toLowerCase().includes(filter.toLowerCase())) return false;
    if (!levelFilter.has(log.level)) return false;
    if (serviceFilter && log.service !== serviceFilter) return false;
    return true;
  });

  // Auto-scroll to bottom in follow mode
  useEffect(() => {
    if (followMode && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs, followMode]);

  const toggleExpand = (id: string) => {
    setExpandedEntries((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleLevelFilter = (level: keyof typeof LOG_LEVEL_COLORS) => {
    setLevelFilter((prev) => {
      const next = new Set(prev);
      if (next.has(level)) {
        if (next.size > 1) next.delete(level);
      } else {
        next.add(level);
      }
      return next;
    });
  };

  const uniqueServices = Array.from(new Set(logs.map((l) => l.service))).sort();

  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      });
    } catch {
      return timestamp;
    }
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">System Logs</h3>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {filteredLogs.length} of {logs.length} entries
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setFollowMode(!followMode)}
            className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
              followMode
                ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300'
            }`}
          >
            Follow
          </button>
          {onAutoRefreshChange && (
            <button
              onClick={() => onAutoRefreshChange(!autoRefresh)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                autoRefresh
                  ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300'
              }`}
            >
              Auto-refresh
            </button>
          )}
          <button
            onClick={onRefresh}
            className="px-3 py-1.5 bg-blue-600 text-white hover:bg-blue-700 rounded-lg text-sm font-medium transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-4 flex-wrap">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Search logs..."
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
            />
          </div>

          {/* Service Filter */}
          <div>
            <select
              value={serviceFilter}
              onChange={(e) => setServiceFilter(e.target.value)}
              className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
            >
              <option value="">All Services</option>
              {uniqueServices.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </div>

          {/* Level Filters */}
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500 dark:text-gray-400">Level:</span>
            {(['debug', 'info', 'warn', 'error'] as const).map((level) => (
              <button
                key={level}
                onClick={() => toggleLevelFilter(level)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                  levelFilter.has(level)
                    ? LOG_LEVEL_COLORS[level]
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 border border-gray-200 dark:border-gray-600'
                }`}
              >
                {level.toUpperCase()}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Log Entries */}
      <div
        ref={containerRef}
        className="bg-gray-900 dark:bg-black rounded-xl p-4 overflow-auto font-mono text-sm"
        style={{ maxHeight }}
      >
        {filteredLogs.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500">No logs match your filters</p>
          </div>
        ) : (
          <div className="space-y-1">
            {filteredLogs.map((log) => {
              const isExpanded = expandedEntries.has(log.id);
              const hasDetails = log.fields || log.stackTrace;

              return (
                <div key={log.id}>
                  <div
                    className={`flex items-start gap-3 py-2 px-3 rounded hover:bg-gray-800 dark:hover:bg-gray-800/50 cursor-pointer transition-colors ${
                      LOG_LEVEL_COLORS[log.level].split(' ')[0]
                    }`}
                    onClick={() => hasDetails && toggleExpand(log.id)}
                  >
                    {/* Expand icon */}
                    {hasDetails && (
                      <span className="text-gray-500 mt-0.5">
                        {isExpanded ? (
                          <ChevronDownIcon className="w-4 h-4" />
                        ) : (
                          <ChevronRightIcon className="w-4 h-4" />
                        )}
                      </span>
                    )}

                    {/* Timestamp */}
                    <span className="text-gray-400 shrink-0">{formatTimestamp(log.timestamp)}</span>

                    {/* Level */}
                    <span className={`shrink-0 ${LOG_LEVEL_COLORS[log.level]}`}>
                      {LOG_LEVEL_ICONS[log.level]}
                    </span>

                    {/* Service */}
                    <span className="text-cyan-400 shrink-0">[{log.service}]</span>

                    {/* Message */}
                    <span className="text-gray-200 break-all">{log.message}</span>
                  </div>

                  {/* Expanded Details */}
                  {isExpanded && (log.fields || log.stackTrace) && (
                    <div className="ml-8 mt-1 p-3 bg-gray-800/50 rounded-lg border border-gray-700">
                      {log.fields && (
                        <div className="mb-2">
                          <p className="text-xs text-gray-400 mb-1">Fields:</p>
                          <div className="grid grid-cols-2 gap-2">
                            {Object.entries(log.fields).map(([key, value]) => (
                              <div key={key} className="text-xs">
                                <span className="text-gray-400">{key}:</span>{' '}
                                <span className="text-gray-200">{value}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                      {log.stackTrace && (
                        <div>
                          <p className="text-xs text-gray-400 mb-1">Stack Trace:</p>
                          <pre className="text-xs text-red-400 overflow-x-auto whitespace-pre-wrap">
                            {log.stackTrace}
                          </pre>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

// Generate mock logs for development
export const generateMockLogs = (count: number = 50): LogEntry[] => {
  const services = ['auth-service', 'job-service', 'registry-service', 'storage-service', 'notification-service'];
  const levels: Array<'debug' | 'info' | 'warn' | 'error'> = ['debug', 'info', 'warn', 'error'];
  const messages = [
    'Request received',
    'Processing query',
    'Cache hit',
    'Cache miss',
    'Database connection established',
    'Query executed successfully',
    'Response sent',
    'High latency detected',
    'Connection pool exhausted',
    'Invalid authentication token',
    'Rate limit exceeded',
    'Service unavailable',
    'Configuration reloaded',
    'Health check passed',
    'Shutting down gracefully',
  ];

  const logs: LogEntry[] = [];
  const now = Date.now();

  for (let i = 0; i < count; i++) {
    const level = levels[Math.floor(Math.random() * levels.length)];
    const isError = level === 'error';
    const isWarn = level === 'warn';

    logs.push({
      id: `log-${i}`,
      timestamp: new Date(now - i * 1000 * Math.random() * 60).toISOString(),
      level,
      service: services[Math.floor(Math.random() * services.length)],
      message: messages[Math.floor(Math.random() * messages.length)],
      fields: Math.random() > 0.7
        ? {
            method: ['GET', 'POST', 'PUT', 'DELETE'][Math.floor(Math.random() * 4)],
            path: `/api/v1/${['auth', 'jobs', 'printers', 'documents'][Math.floor(Math.random() * 4)]}`,
            duration: `${Math.floor(Math.random() * 1000)}ms`,
            status: isError ? '500' : isWarn ? '429' : '200',
          }
        : undefined,
      stackTrace: isError
        ? `Error: Request failed
    at processTicksAndRejections (internal/process/task_queues.js:95:5)
    at async /app/src/handler.ts:42:15
    at async Layer.handle [as handle_request] (/app/node_modules/@nestjs/core/router/router-execution-context.js:46:28)`
        : undefined,
    });
  }

  return logs;
};
