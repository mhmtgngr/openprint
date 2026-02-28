/**
 * API functions for device/printer management
 * Communicates with registry-service endpoints
 */

import type {
  Device,
  DeviceAgent,
  DevicePrinter,
  DeviceListParams,
  DeviceStats,
  RegisterPrinterFormData,
  RegisterAgentFormData,
  Agent,
  Printer,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

const getAccessToken = (): string | null => {
  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored) as { accessToken: string };
    return tokens.accessToken;
  }
  return null;
};

const fetchWithAuth = async (
  url: string,
  options: RequestInit = {}
): Promise<Response> => {
  const token = getAccessToken();

  if (token) {
    options.headers = {
      ...options.headers,
      Authorization: `Bearer ${token}`,
    };
  }

  return fetch(url, options);
};

const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = (await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }))) as { code: string; message: string; details?: Record<string, unknown> };
    throw new Error(error.message || 'API request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

// Convert agents and printers to unified Device format
const toDevices = (agents: DeviceAgent[], printers: DevicePrinter[]): Device[] => {
  return [
    ...agents.map(
      (agent): Device => ({
        id: agent.id,
        type: 'agent' as const,
        name: agent.name,
        status: agent.status as 'online' | 'offline' | 'error',
        lastSeen: agent.lastHeartbeat || agent.createdAt,
        createdAt: agent.createdAt,
      })
    ),
    ...printers.map(
      (printer): Device => ({
        id: printer.id,
        type: 'printer' as const,
        name: printer.name,
        status: printer.isOnline ? 'online' : 'offline',
        lastSeen: printer.lastSeen || printer.createdAt,
        createdAt: printer.createdAt,
      })
    ),
  ];
};

// Devices API
export const devicesApi = {
  /**
   * List all devices (agents and printers) with optional filtering
   * GET /api/agents, GET /api/printers
   */
  async list(params?: DeviceListParams): Promise<{
    devices: Device[];
    agents: DeviceAgent[];
    printers: DevicePrinter[];
    stats: DeviceStats;
  }> {
    // Fetch both agents and printers in parallel
    const [agentsResponse, printersResponse] = await Promise.all([
      fetchWithAuth(`${API_BASE_URL}/agents`),
      fetchWithAuth(`${API_BASE_URL}/printers`),
    ]);

    const agents = (await handleResponse<Agent[]>(agentsResponse)) as Agent[];
    const printers = (await handleResponse<Printer[]>(printersResponse)) as Printer[];

    // Calculate stats
    const onlineAgents = agents.filter((a) => a.status === 'online').length;
    const onlinePrinters = printers.filter((p) => p.isOnline).length;

    const stats: DeviceStats = {
      totalAgents: agents.length,
      onlineAgents,
      totalPrinters: printers.length,
      onlinePrinters,
      offlinePrinters: printers.length - onlinePrinters,
    };

    // Enhance with printer count per agent
    const enhancedAgents: DeviceAgent[] = agents.map((agent) => ({
      ...agent,
      printerCount: printers.filter((p) => p.agentId === agent.id).length,
      uptime: agent.lastHeartbeat
        ? formatUptime(new Date(agent.lastHeartbeat))
        : undefined,
    }));

    // Enhance printers with agent info
    const enhancedPrinters: DevicePrinter[] = printers.map((printer) => {
      const agent = agents.find((a) => a.id === printer.agentId);
      return {
        ...printer,
        agentName: agent?.name,
        agentStatus: agent?.status,
        uptime: printer.lastSeen
          ? formatUptime(new Date(printer.lastSeen))
          : undefined,
      };
    });

    // Filter if needed
    let filteredAgents = enhancedAgents;
    let filteredPrinters = enhancedPrinters;

    if (params) {
      if (params.status && params.status !== 'all') {
        filteredAgents = enhancedAgents.filter((a) => a.status === params.status);
        filteredPrinters = enhancedPrinters.filter((p) =>
          params.status === 'online' ? p.isOnline : !p.isOnline
        );
      }

      if (params.type && params.type !== 'all') {
        if (params.type === 'agent') {
          filteredPrinters = [];
        } else {
          filteredAgents = [];
        }
      }

      if (params.search) {
        const searchLower = params.search.toLowerCase();
        filteredAgents = filteredAgents.filter((a) =>
          a.name.toLowerCase().includes(searchLower)
        );
        filteredPrinters = filteredPrinters.filter((p) =>
          p.name.toLowerCase().includes(searchLower)
        );
      }

      if (params.limit) {
        const offset = params.offset || 0;
        filteredAgents = filteredAgents.slice(offset, offset + params.limit);
        filteredPrinters = filteredPrinters.slice(offset, offset + params.limit);
      }
    }

    const devices = toDevices(filteredAgents, filteredPrinters);

    return { devices, agents: filteredAgents, printers: filteredPrinters, stats };
  },

  /**
   * Get a single device by ID
   * GET /api/agents/:id or GET /api/printers/:id
   */
  async get(id: string, type: 'agent' | 'printer'): Promise<DeviceAgent | DevicePrinter> {
    const endpoint = type === 'agent' ? 'agents' : 'printers';
    const response = await fetchWithAuth(`${API_BASE_URL}/${endpoint}/${id}`);
    const data = await handleResponse<Agent | Printer>(response);

    if (type === 'agent') {
      const agent = data as Agent;
      return {
        ...agent,
        printerCount: 0, // Will be populated separately
        uptime: agent.lastHeartbeat
          ? formatUptime(new Date(agent.lastHeartbeat))
          : undefined,
      } as DeviceAgent;
    } else {
      const printer = data as Printer;
      return {
        ...printer,
        uptime: printer.lastSeen
          ? formatUptime(new Date(printer.lastSeen))
          : undefined,
      } as DevicePrinter;
    }
  },

  /**
   * Register a new printer
   * POST /api/printers
   */
  async registerPrinter(data: RegisterPrinterFormData): Promise<Printer> {
    const response = await fetchWithAuth(`${API_BASE_URL}/printers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Printer>(response);
  },

  /**
   * Register a new agent
   * POST /api/agents
   */
  async registerAgent(data: RegisterAgentFormData): Promise<Agent> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Agent>(response);
  },

  /**
   * Update a device
   * PATCH /api/agents/:id or PATCH /api/printers/:id
   */
  async update(
    id: string,
    type: 'agent' | 'printer',
    data: Partial<Agent | Printer>
  ): Promise<Agent | Printer> {
    const endpoint = type === 'agent' ? 'agents' : 'printers';
    const response = await fetchWithAuth(`${API_BASE_URL}/${endpoint}/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Agent | Printer>(response);
  },

  /**
   * Delete a device
   * DELETE /api/agents/:id or DELETE /api/printers/:id
   */
  async delete(id: string, type: 'agent' | 'printer'): Promise<void> {
    const endpoint = type === 'agent' ? 'agents' : 'printers';
    const response = await fetchWithAuth(`${API_BASE_URL}/${endpoint}/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Enable/disable a printer
   * PATCH /api/printers/:id/active
   */
  async setPrinterActive(id: string, isActive: boolean): Promise<Printer> {
    const response = await fetchWithAuth(`${API_BASE_URL}/printers/${id}/active`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_active: isActive }),
    });
    return handleResponse<Printer>(response);
  },

  /**
   * Get printers for a specific agent
   * GET /api/agents/:id/printers
   */
  async getAgentPrinters(agentId: string): Promise<Printer[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${agentId}/printers`);
    return handleResponse<Printer[]>(response);
  },

  /**
   * Get device stats
   */
  async getStats(): Promise<DeviceStats> {
    const result = await this.list();
    return result.stats;
  },
};

// Utility function to format uptime
function formatUptime(lastSeen: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - lastSeen.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) {
    return 'Just now';
  } else if (diffMins < 60) {
    return `${diffMins}m ago`;
  } else if (diffHours < 24) {
    return `${diffHours}h ago`;
  } else if (diffDays < 7) {
    return `${diffDays}d ago`;
  } else {
    return lastSeen.toLocaleDateString();
  }
}

// Type guards
export const isAgent = (device: DeviceAgent | DevicePrinter): device is DeviceAgent => {
  return 'platform' in device;
};

export const isPrinter = (device: DeviceAgent | DevicePrinter): device is DevicePrinter => {
  return 'capabilities' in device;
};
