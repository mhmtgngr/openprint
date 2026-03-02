import { ServiceHealth, HealthStatus } from '@/types';
import { CheckIcon, XCircleIcon, AlertIcon, WifiOffIcon } from '@/components/icons';

interface ServiceHealthCardProps {
  health: ServiceHealth;
  onClick?: () => void;
}

const statusConfig: Record<
  HealthStatus,
  { bg: string; text: string; border: string; icon: React.ReactNode }
> = {
  healthy: {
    bg: 'bg-green-100 dark:bg-green-900/30',
    text: 'text-green-700 dark:text-green-400',
    border: 'border-green-200 dark:border-green-800',
    icon: <CheckIcon className="w-5 h-5" />,
  },
  degraded: {
    bg: 'bg-amber-100 dark:bg-amber-900/30',
    text: 'text-amber-700 dark:text-amber-400',
    border: 'border-amber-200 dark:border-amber-800',
    icon: <AlertIcon className="w-5 h-5" />,
  },
  unhealthy: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    text: 'text-red-700 dark:text-red-400',
    border: 'border-red-200 dark:border-red-800',
    icon: <XCircleIcon className="w-5 h-5" />,
  },
  unknown: {
    bg: 'bg-gray-100 dark:bg-gray-900/30',
    text: 'text-gray-700 dark:text-gray-400',
    border: 'border-gray-200 dark:border-gray-800',
    icon: <WifiOffIcon className="w-5 h-5" />,
  },
};

export const ServiceHealthCard = ({ health, onClick }: ServiceHealthCardProps) => {
  const status = health.status || 'unknown';
  const config = statusConfig[status];

  const formatUptime = (ms: number): string => {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ${hours % 24}h`;
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m`;
    return `${seconds}s`;
  };

  const getUsageColor = (percent: number): string => {
    if (percent >= 90) return 'text-red-600 dark:text-red-400';
    if (percent >= 70) return 'text-amber-600 dark:text-amber-400';
    return 'text-green-600 dark:text-green-400';
  };

  return (
    <div
      className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border transition-all cursor-pointer hover:shadow-md ${
        onClick ? 'hover:border-blue-300 dark:hover:border-blue-700' : config.border
      } border-gray-200 dark:border-gray-700`}
      onClick={onClick}
    >
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={`${config.bg} ${config.text} p-2 rounded-lg`}>
              {config.icon}
            </div>
            <div>
              <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                {health.serviceName}
              </h3>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {health.instance} • v{health.version}
              </p>
            </div>
          </div>
          <div className="text-right">
            <span
              className={`px-2 py-1 rounded-lg text-xs font-medium ${config.bg} ${config.text}`}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </span>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Up {formatUptime(health.uptime)}
            </p>
          </div>
        </div>
      </div>

      {/* Metrics */}
      <div className="p-4 space-y-3">
        {/* Resource Usage */}
        <div className="grid grid-cols-3 gap-3">
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">CPU</p>
            <p className={`text-sm font-medium ${getUsageColor(health.metrics.cpuPercent)}`}>
              {health.metrics.cpuPercent.toFixed(1)}%
            </p>
            <div className="h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full mt-1">
              <div
                className={`h-full rounded-full ${
                  health.metrics.cpuPercent >= 90
                    ? 'bg-red-500'
                    : health.metrics.cpuPercent >= 70
                      ? 'bg-amber-500'
                      : 'bg-green-500'
                }`}
                style={{ width: `${Math.min(health.metrics.cpuPercent, 100)}%` }}
              />
            </div>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Memory</p>
            <p className={`text-sm font-medium ${getUsageColor(health.metrics.memoryPercent)}`}>
              {health.metrics.memoryPercent.toFixed(1)}%
            </p>
            <div className="h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full mt-1">
              <div
                className={`h-full rounded-full ${
                  health.metrics.memoryPercent >= 90
                    ? 'bg-red-500'
                    : health.metrics.memoryPercent >= 70
                      ? 'bg-amber-500'
                      : 'bg-green-500'
                }`}
                style={{ width: `${Math.min(health.metrics.memoryPercent, 100)}%` }}
              />
            </div>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">Disk</p>
            <p className={`text-sm font-medium ${getUsageColor(health.metrics.diskPercent)}`}>
              {health.metrics.diskPercent.toFixed(1)}%
            </p>
            <div className="h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full mt-1">
              <div
                className={`h-full rounded-full ${
                  health.metrics.diskPercent >= 90
                    ? 'bg-red-500'
                    : health.metrics.diskPercent >= 70
                      ? 'bg-amber-500'
                      : 'bg-green-500'
                }`}
                style={{ width: `${Math.min(health.metrics.diskPercent, 100)}%` }}
              />
            </div>
          </div>
        </div>

        {/* Performance Metrics */}
        <div className="grid grid-cols-3 gap-3 pt-2 border-t border-gray-100 dark:border-gray-700">
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400">Request Rate</p>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
              {health.metrics.requestRate.toFixed(1)}/s
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400">Error Rate</p>
            <p
              className={`text-sm font-medium ${
                health.metrics.errorRate > 1
                  ? 'text-red-600 dark:text-red-400'
                  : health.metrics.errorRate > 0.1
                    ? 'text-amber-600 dark:text-amber-400'
                    : 'text-green-600 dark:text-green-400'
              }`}
            >
              {health.metrics.errorRate.toFixed(2)}%
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500 dark:text-gray-400">P95 Latency</p>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
              {health.metrics.latency.p95}ms
            </p>
          </div>
        </div>

        {/* Dependencies */}
        {health.dependencies && health.dependencies.length > 0 && (
          <div className="pt-2 border-t border-gray-100 dark:border-gray-700">
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">Dependencies</p>
            <div className="flex flex-wrap gap-2">
              {health.dependencies.map((dep, i) => {
                const depStatus = dep.status || 'unknown';
                const depConfig = statusConfig[depStatus];
                return (
                  <div
                    key={i}
                    className={`flex items-center gap-1.5 px-2 py-1 rounded-md ${depConfig.bg} ${depConfig.border} border`}
                  >
                    <span className={`${depConfig.text} text-xs`}>{depConfig.icon}</span>
                    <span className="text-xs text-gray-700 dark:text-gray-300">{dep.name}</span>
                    {dep.latency && (
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {dep.latency}ms
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>

      {/* Last Check */}
      <div className="px-4 py-2 bg-gray-50 dark:bg-gray-900/50 border-t border-gray-200 dark:border-gray-700 rounded-b-xl">
        <p className="text-xs text-gray-500 dark:text-gray-400">
          Last check: {new Date(health.lastCheck).toLocaleString()}
        </p>
      </div>
    </div>
  );
};

interface ServiceHealthListProps {
  services: ServiceHealth[];
  onServiceClick?: (service: ServiceHealth) => void;
}

export const ServiceHealthList = ({ services, onServiceClick }: ServiceHealthListProps) => {
  const groupedByStatus = services.reduce(
    (acc, service) => {
      const status = service.status || 'unknown';
      acc[status] = acc[status] || [];
      acc[status].push(service);
      return acc;
    },
    {} as Record<HealthStatus, ServiceHealth[]>
  );

  const statusOrder: HealthStatus[] = ['unhealthy', 'degraded', 'healthy', 'unknown'];

  return (
    <div className="space-y-6">
      {statusOrder.map((status) => {
        const statusServices = groupedByStatus[status];
        if (!statusServices || statusServices.length === 0) return null;

        const config = statusConfig[status];

        return (
          <div key={status}>
            <div className={`flex items-center gap-2 mb-3 ${config.text}`}>
              {config.icon}
              <h3 className="font-semibold capitalize">{status} Services</h3>
              <span className="px-2 py-0.5 bg-white/50 dark:bg-black/20 rounded-full text-xs">
                {statusServices.length}
              </span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {statusServices.map((service) => (
                <ServiceHealthCard
                  key={service.serviceName + service.instance}
                  health={service}
                  onClick={() => onServiceClick?.(service)}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};
