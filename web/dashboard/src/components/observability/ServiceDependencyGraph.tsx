import { useState, useRef, useEffect } from 'react';
import { ServiceHealth, HealthStatus } from '@/types';

interface ServiceDependencyGraphProps {
  services: ServiceHealth[];
  onServiceClick?: (service: ServiceHealth) => void;
}

interface ServiceNode {
  id: string;
  name: string;
  status: HealthStatus;
  x: number;
  y: number;
}

interface DependencyEdge {
  from: string;
  to: string;
  status: HealthStatus;
}

const STATUS_COLORS: Record<HealthStatus, string> = {
  healthy: '#10b981',
  degraded: '#f59e0b',
  unhealthy: '#ef4444',
  unknown: '#9ca3af',
};

const getServiceIcon = (status: HealthStatus) => {
  switch (status) {
    case 'healthy':
      return (
        <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" strokeWidth="2" />
      );
    case 'degraded':
      return (
        <path
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        />
      );
    case 'unhealthy':
      return (
        <>
          <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" strokeWidth="2" />
          <path d="M15 9l-6 6M9 9l6 6" stroke="currentColor" strokeWidth="2" />
        </>
      );
    default:
      return (
        <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" strokeWidth="2" strokeDasharray="4 2" />
      );
  }
};

export const ServiceDependencyGraph = ({ services, onServiceClick }: ServiceDependencyGraphProps) => {
  const [selectedService, setSelectedService] = useState<ServiceHealth | null>(null);
  const [hoveredService, setHoveredService] = useState<string | null>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [nodes, setNodes] = useState<ServiceNode[]>([]);
  const [edges, setEdges] = useState<DependencyEdge[]>([]);

  // Build dependency graph
  useEffect(() => {
    const nodeMap = new Map<string, ServiceNode>();
    const edgeList: DependencyEdge[] = [];
    const centerX = 400;
    const centerY = 300;
    const radius = 200;

    // Create nodes in a circular layout
    services.forEach((service, i) => {
      const angle = (2 * Math.PI * i) / services.length - Math.PI / 2;
      nodeMap.set(service.serviceName, {
        id: service.serviceName,
        name: service.serviceName.replace('-', '\n'),
        status: service.status,
        x: centerX + radius * Math.cos(angle),
        y: centerY + radius * Math.sin(angle),
      });
    });

    // Create edges from dependencies
    services.forEach((service) => {
      service.dependencies?.forEach((dep) => {
        if (nodeMap.has(dep.name)) {
          edgeList.push({
            from: service.serviceName,
            to: dep.name,
            status: dep.status,
          });
        }
      });
    });

    setNodes(Array.from(nodeMap.values()));
    setEdges(edgeList);
  }, [services]);

  // Draw graph on canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || nodes.length === 0) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Clear canvas
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // Draw edges
    edges.forEach((edge) => {
      const fromNode = nodes.find((n) => n.id === edge.from);
      const toNode = nodes.find((n) => n.id === edge.to);
      if (!fromNode || !toNode) return;

      ctx.beginPath();
      ctx.moveTo(fromNode.x, fromNode.y);
      ctx.lineTo(toNode.x, toNode.y);

      const isHighlighted = hoveredService === edge.from || hoveredService === edge.to;
      ctx.strokeStyle = isHighlighted ? STATUS_COLORS[edge.status] : '#e5e7eb';
      ctx.lineWidth = isHighlighted ? 3 : 2;
      ctx.stroke();

      // Draw arrowhead
      const angle = Math.atan2(toNode.y - fromNode.y, toNode.x - fromNode.x);
      const arrowSize = 10;
      const arrowX = toNode.x - 50 * Math.cos(angle);
      const arrowY = toNode.y - 50 * Math.sin(angle);

      ctx.beginPath();
      ctx.moveTo(arrowX, arrowY);
      ctx.lineTo(
        arrowX - arrowSize * Math.cos(angle - Math.PI / 6),
        arrowY - arrowSize * Math.sin(angle - Math.PI / 6)
      );
      ctx.lineTo(
        arrowX - arrowSize * Math.cos(angle + Math.PI / 6),
        arrowY - arrowSize * Math.sin(angle + Math.PI / 6)
      );
      ctx.closePath();
      ctx.fillStyle = ctx.strokeStyle;
      ctx.fill();
    });

    // Draw nodes
    nodes.forEach((node) => {
      const isHovered = hoveredService === node.id;
      const isSelected = selectedService?.serviceName === node.id;

      // Draw node circle
      ctx.beginPath();
      ctx.arc(node.x, node.y, isHovered || isSelected ? 45 : 40, 0, 2 * Math.PI);
      ctx.fillStyle = STATUS_COLORS[node.status] + (isHovered ? '40' : '20');
      ctx.fill();
      ctx.strokeStyle = STATUS_COLORS[node.status];
      ctx.lineWidth = isSelected ? 4 : 2;
      ctx.stroke();

      // Draw service name
      ctx.fillStyle = '#1f2937';
      ctx.font = `${isHovered ? 'bold ' : ''}12px system-ui`;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';

      const lines = node.name.split('\n');
      lines.forEach((line, i) => {
        ctx.fillText(line, node.x, node.y - 6 + i * 14);
      });
    });
  }, [nodes, edges, hoveredService, selectedService]);

  // Handle canvas interactions
  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    const clickedNode = nodes.find(
      (node) => Math.hypot(node.x - x, node.y - y) < 40
    );

    if (clickedNode) {
      const service = services.find((s) => s.serviceName === clickedNode.id);
      if (service) {
        setSelectedService(service);
        onServiceClick?.(service);
      }
    } else {
      setSelectedService(null);
    }
  };

  const handleCanvasMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    const hovered = nodes.find(
      (node) => Math.hypot(node.x - x, node.y - y) < 40
    );

    setHoveredService(hovered?.id || null);
  };

  // Count services by status
  const statusCounts = services.reduce(
    (acc, s) => {
      acc[s.status] = (acc[s.status] || 0) + 1;
      return acc;
    },
    {} as Record<HealthStatus, number>
  );

  return (
    <div className="space-y-4">
      {/* Legend */}
      <div className="flex items-center gap-6 text-sm">
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-green-500" />
          <span className="text-gray-600 dark:text-gray-400">
            Healthy: {statusCounts.healthy || 0}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-amber-500" />
          <span className="text-gray-600 dark:text-gray-400">
            Degraded: {statusCounts.degraded || 0}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="w-3 h-3 rounded-full bg-red-500" />
          <span className="text-gray-600 dark:text-gray-400">
            Unhealthy: {statusCounts.unhealthy || 0}
          </span>
        </div>
      </div>

      {/* Canvas */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <canvas
          ref={canvasRef}
          width={800}
          height={600}
          onClick={handleCanvasClick}
          onMouseMove={handleCanvasMouseMove}
          onMouseLeave={() => setHoveredService(null)}
          className="w-full cursor-pointer"
          style={{ maxWidth: '100%', height: 'auto' }}
        />
      </div>

      {/* Selected Service Info */}
      {selectedService && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between mb-4">
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

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Status</p>
              <p className={`font-medium capitalize ${
                selectedService.status === 'healthy'
                  ? 'text-green-600 dark:text-green-400'
                  : selectedService.status === 'degraded'
                    ? 'text-amber-600 dark:text-amber-400'
                    : 'text-red-600 dark:text-red-400'
              }`}>
                {selectedService.status}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">CPU</p>
              <p className="font-medium text-gray-900 dark:text-gray-100">
                {selectedService.metrics.cpuPercent.toFixed(1)}%
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Memory</p>
              <p className="font-medium text-gray-900 dark:text-gray-100">
                {selectedService.metrics.memoryPercent.toFixed(1)}%
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Request Rate</p>
              <p className="font-medium text-gray-900 dark:text-gray-100">
                {selectedService.metrics.requestRate.toFixed(1)}/s
              </p>
            </div>
          </div>

          {selectedService.dependencies && selectedService.dependencies.length > 0 && (
            <div className="mt-4">
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-2">Dependencies</p>
              <div className="flex flex-wrap gap-2">
                {selectedService.dependencies.map((dep, i) => {
                  const depStatus = dep.status || 'unknown';
                  return (
                    <span
                      key={i}
                      className={`px-2 py-1 rounded-md text-sm flex items-center gap-1.5 border-${
                        depStatus === 'healthy' ? 'green' : depStatus === 'degraded' ? 'amber' : 'red'
                      }-200 dark:border-${
                        depStatus === 'healthy' ? 'green' : depStatus === 'degraded' ? 'amber' : 'red'
                      }-800 bg-${
                        depStatus === 'healthy' ? 'green' : depStatus === 'degraded' ? 'amber' : 'red'
                      }-50 dark:bg-${
                        depStatus === 'healthy' ? 'green' : depStatus === 'degraded' ? 'amber' : 'red'
                      }-900/20`}
                    >
                      <span
                        className={`text-${
                          depStatus === 'healthy' ? 'green' : depStatus === 'degraded' ? 'amber' : 'red'
                        }-600 dark:text-${
                          depStatus === 'healthy' ? 'green' : depStatus === 'degraded' ? 'amber' : 'red'
                        }-400`}
                      >
                        {getServiceIcon(depStatus)}
                      </span>
                      {dep.name}
                    </span>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// Legend component for standalone use
export const DependencyGraphLegend = () => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
      <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Legend</h4>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded-full border-2 border-green-500" />
          <span className="text-sm text-gray-600 dark:text-gray-400">Healthy</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded-full border-2 border-amber-500" />
          <span className="text-sm text-gray-600 dark:text-gray-400">Degraded</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-4 h-4 rounded-full border-2 border-red-500" />
          <span className="text-sm text-gray-600 dark:text-gray-400">Unhealthy</span>
        </div>
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-gray-400" viewBox="0 0 24 24" fill="none">
            <path d="M5 12h14" stroke="currentColor" strokeWidth="2" markerEnd="url(#arrow)" />
          </svg>
          <span className="text-sm text-gray-600 dark:text-gray-400">Dependency</span>
        </div>
      </div>
    </div>
  );
};
