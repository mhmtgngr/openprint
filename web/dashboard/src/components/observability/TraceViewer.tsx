import { useState } from 'react';
import { Trace, Span, TraceSearchResult } from '@/types';
import { SearchIcon, FilterIcon } from '@/components/icons';

interface TraceViewerProps {
  trace: Trace;
}

interface SpanRowProps {
  span: Span;
  level: number;
  allSpans: Span[];
  isExpanded: boolean;
  onToggle: (spanId: string) => void;
}

const SpanRow = ({ span, level, allSpans, isExpanded, onToggle }: SpanRowProps) => {
  const hasChildren = allSpans.some((s) => s.parentSpanID === span.spanID);
  const childSpans = allSpans.filter((s) => s.parentSpanID === span.spanID);

  // Calculate width based on duration relative to trace
  const getDurationPercent = () => {
    const traceDuration = Math.max(...allSpans.map((s) => s.startTime + s.duration));
    return ((span.duration / traceDuration) * 100);
  };

  // Calculate offset based on start time
  const getOffsetPercent = () => {
    const traceStart = Math.min(...allSpans.map((s) => s.startTime));
    const traceDuration = Math.max(...allSpans.map((s) => s.startTime + s.duration)) - traceStart;
    return (((span.startTime - traceStart) / traceDuration) * 100);
  };

  const getSpanColor = (): string => {
    const hasError = span.tags.some((t) => t.key === 'error' && t.value === 'true');
    if (hasError) return 'bg-red-500';

    const serviceName = span.tags.find((t) => t.key === 'span.kind')?.value || 'unknown';
    const colors = [
      'bg-blue-500',
      'bg-green-500',
      'bg-purple-500',
      'bg-amber-500',
      'bg-cyan-500',
      'bg-pink-500',
      'bg-indigo-500',
    ];
    const index = Math.abs(serviceName.charCodeAt(0)) % colors.length;
    return colors[index];
  };

  const formatDuration = (nanos: number): string => {
    if (nanos < 1000) return `${nanos}ns`;
    if (nanos < 1000000) return `${(nanos / 1000).toFixed(2)}µs`;
    if (nanos < 1000000000) return `${(nanos / 1000000).toFixed(2)}ms`;
    return `${(nanos / 1000000000).toFixed(2)}s`;
  };

  return (
    <>
      <div
        className="flex items-center gap-2 py-2 px-3 hover:bg-gray-50 dark:hover:bg-gray-700/50 rounded cursor-pointer"
        style={{ paddingLeft: `${level * 24 + 12}px` }}
        onClick={() => hasChildren && onToggle(span.spanID)}
      >
        {hasChildren ? (
          <button className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
            <svg
              className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        ) : (
          <span className="w-4" />
        )}

        {/* Operation Name */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
            {span.operationName}
          </p>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {span.tags.find((t) => t.key === 'peer.service')?.value || 'Unknown Service'}
          </p>
        </div>

        {/* Duration */}
        <div className="text-right">
          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
            {formatDuration(span.duration)}
          </p>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {new Date(span.startTime / 1000).toLocaleTimeString()}
          </p>
        </div>

        {/* Timeline */}
        <div className="w-32 h-6 bg-gray-200 dark:bg-gray-700 rounded relative overflow-hidden">
          <div
            className={`absolute top-0 bottom-0 ${getSpanColor()} opacity-70 rounded-sm`}
            style={{
              left: `${getOffsetPercent()}%`,
              width: `${Math.max(getDurationPercent(), 2)}%`,
            }}
          />
        </div>

        {/* Tags indicator */}
        {span.tags.length > 0 && (
          <span className="text-xs text-gray-400">+{span.tags.length}</span>
        )}
      </div>

      {/* Child spans */}
      {isExpanded &&
        childSpans.map((child) => (
          <SpanRow
            key={child.spanID}
            span={child}
            level={level + 1}
            allSpans={allSpans}
            isExpanded={false}
            onToggle={onToggle}
          />
        ))}
    </>
  );
};

export const TraceViewer = ({ trace }: TraceViewerProps) => {
  const [expandedSpans, setExpandedSpans] = useState<Set<string>>(new Set());

  const toggleSpan = (spanId: string) => {
    setExpandedSpans((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) {
        next.delete(spanId);
      } else {
        next.add(spanId);
      }
      return next;
    });
  };

  const rootSpans = trace.spans.filter((s) => !s.parentSpanID);
  const errorSpans = trace.spans.filter((s) =>
    s.tags.some((t) => t.key === 'error' && t.value === 'true')
  );

  const formatDuration = (nanos: number): string => {
    if (nanos < 1000000) return `${(nanos / 1000).toFixed(2)}µs`;
    if (nanos < 1000000000) return `${(nanos / 1000000).toFixed(2)}ms`;
    return `${(nanos / 1000000000).toFixed(2)}s`;
  };

  return (
    <div className="space-y-4">
      {/* Trace Header */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {trace.rootSpanName}
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Service: {trace.rootServiceName}
            </p>
          </div>
          <div className="text-right">
            <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {formatDuration(trace.duration)}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Started: {new Date(trace.startTime / 1000).toLocaleString()}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <div>
            <p className="text-sm text-gray-500 dark:text-gray-400">Total Spans</p>
            <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {trace.spans.length}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-500 dark:text-gray-400">Services</p>
            <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {Object.keys(trace.processes).length}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-500 dark:text-gray-400">Errors</p>
            <p
              className={`text-lg font-semibold ${
                errorSpans.length > 0
                  ? 'text-red-600 dark:text-red-400'
                  : 'text-green-600 dark:text-green-400'
              }`}
            >
              {errorSpans.length}
            </p>
          </div>
        </div>
      </div>

      {/* Span Tree */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
          <h4 className="font-medium text-gray-900 dark:text-gray-100">Trace Spans</h4>
        </div>
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {rootSpans.map((span) => (
            <SpanRow
              key={span.spanID}
              span={span}
              level={0}
              allSpans={trace.spans}
              isExpanded={expandedSpans.has(span.spanID)}
              onToggle={toggleSpan}
            />
          ))}
        </div>
      </div>

      {/* Span Details */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
          <h4 className="font-medium text-gray-900 dark:text-gray-100">Services</h4>
        </div>
        <div className="p-4">
          <div className="flex flex-wrap gap-2">
            {Object.values(trace.processes).map((process, i) => (
              <div
                key={i}
                className="px-3 py-2 bg-gray-100 dark:bg-gray-700 rounded-lg"
              >
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {process.serviceName}
                </p>
                <div className="flex flex-wrap gap-1 mt-1">
                  {process.tags.slice(0, 3).map((tag, j) => (
                    <span
                      key={j}
                      className="text-xs text-gray-500 dark:text-gray-400"
                    >
                      {tag.key}={tag.value}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

interface TraceSearchProps {
  onTraceSelect: (traceId: string) => void;
  recentTraces?: TraceSearchResult[];
}

export const TraceSearch = ({ onTraceSelect, recentTraces = [] }: TraceSearchProps) => {
  const [query, setQuery] = useState('');
  const [service, setService] = useState('');
  const [maxDuration, setMaxDuration] = useState('');

  const formatDuration = (nanos: number): string => {
    if (nanos < 1000000) return `${(nanos / 1000).toFixed(2)}µs`;
    if (nanos < 1000000000) return `${(nanos / 1000000).toFixed(2)}ms`;
    return `${(nanos / 1000000000).toFixed(2)}s`;
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
      <div className="p-6 border-b border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          Search Traces
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="md:col-span-2">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Operation
            </label>
            <div className="relative">
              <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search by operation name..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Service
            </label>
            <input
              type="text"
              value={service}
              onChange={(e) => setService(e.target.value)}
              placeholder="Filter by service..."
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Max Duration
            </label>
            <input
              type="text"
              value={maxDuration}
              onChange={(e) => setMaxDuration(e.target.value)}
              placeholder="e.g. 500ms, 1s"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>
        <div className="flex items-center justify-between mt-4">
          <div className="flex items-center gap-2">
            <FilterIcon className="w-4 h-4 text-gray-400" />
            <span className="text-sm text-gray-500 dark:text-gray-400">
              {recentTraces.length} traces found
            </span>
          </div>
          <div className="flex items-center gap-2">
            <button className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors">
              Reset
            </button>
            <button className="px-4 py-2 text-sm font-medium bg-blue-600 text-white hover:bg-blue-700 rounded-lg transition-colors flex items-center gap-2">
              <SearchIcon className="w-4 h-4" />
              Search
            </button>
          </div>
        </div>
      </div>

      {/* Results */}
      <div className="divide-y divide-gray-200 dark:divide-gray-700">
        {recentTraces.map((result) => (
          <div
            key={result.traceID}
            className="px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer transition-colors"
            onClick={() => onTraceSelect(result.traceID)}
          >
            <div className="flex items-center justify-between">
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                  {result.rootSpanName}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {result.rootServiceName} • {result.spanCount} spans
                </p>
              </div>
              <div className="text-right ml-4">
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {formatDuration(result.duration)}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {new Date(result.startTime / 1000).toLocaleTimeString()}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
