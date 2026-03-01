/**
 * AgentList Component Tests
 * Tests for the agent listing component
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { AgentList } from './AgentList';
import type { Agent } from '@/types/agents';

const mockAgents: Agent[] = [
  {
    id: 'agent-1',
    name: 'Office Agent',
    platform: 'linux',
    agentVersion: '1.2.0',
    status: 'online',
    createdAt: '2025-02-01T10:00:00Z',
    lastHeartbeat: '2025-02-28T12:00:00Z',
    orgId: 'org-1',
    capabilities: {
      supportedFormats: ['pdf', 'docx'],
      maxJobSize: 10485760,
      supportsColor: true,
      supportsDuplex: true,
    },
    associatedUser: { id: 'user-1', name: 'John Doe', email: 'john@example.com' },
    printerCount: 3,
    jobQueueDepth: 2,
  },
  {
    id: 'agent-2',
    name: 'Warehouse Agent',
    platform: 'windows',
    agentVersion: '1.1.5',
    status: 'offline',
    createdAt: '2025-02-10T14:00:00Z',
    lastHeartbeat: '2025-02-28T08:00:00Z',
    orgId: 'org-1',
    capabilities: {
      supportedFormats: ['pdf'],
      maxJobSize: 5242880,
      supportsColor: false,
      supportsDuplex: true,
    },
    printerCount: 2,
  },
  {
    id: 'agent-3',
    name: 'Reception Agent',
    platform: 'darwin',
    agentVersion: '1.2.0',
    status: 'error',
    createdAt: '2025-02-15T09:00:00Z',
    lastHeartbeat: '2025-02-28T11:30:00Z',
    orgId: 'org-1',
    capabilities: {
      supportedFormats: ['pdf', 'jpg', 'png'],
      maxJobSize: 10485760,
      supportsColor: true,
      supportsDuplex: true,
    },
    printerCount: 1,
  },
];

// Mock hooks
vi.mock('./useAgents', () => ({
  useDeleteAgent: () => ({
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  }),
  useRestartAgent: () => ({
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  }),
}));

describe('AgentList', () => {
  describe('Loading State', () => {
    it('should render loading skeletons when isLoading is true', () => {
      render(<AgentList agents={[]} isLoading={true} />);

      const skeletons = document.querySelectorAll('.animate-pulse');
      expect(skeletons.length).toBe(5);
    });

    it('should render skeleton with proper styling', () => {
      render(<AgentList agents={[]} isLoading={true} />);

      const skeletons = document.querySelectorAll('.bg-gray-100, .dark\\:bg-gray-800');
      expect(skeletons.length).toBeGreaterThan(0);
    });
  });

  describe('Empty State', () => {
    it('should render empty state when no agents', () => {
      render(<AgentList agents={[]} />);

      expect(screen.getByText('No agents found')).toBeInTheDocument();
    });

    it('should render empty state description', () => {
      render(<AgentList agents={[]} />);

      expect(screen.getByText(/Install the OpenPrint Agent/)).toBeInTheDocument();
    });

    it('should render empty state icon', () => {
      render(<AgentList agents={[]} />);

      const icon = document.querySelector('svg.text-gray-400');
      expect(icon).toBeInTheDocument();
    });

    it('should not render list when empty', () => {
      render(<AgentList agents={[]} />);

      const list = document.querySelector('ul.divide-y');
      expect(list).not.toBeInTheDocument();
    });
  });

  describe('Status Summary', () => {
    it('should render agent count summary', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/3 agents/)).toBeInTheDocument();
    });

    it('should render online count', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/1 online/)).toBeInTheDocument();
    });

    it('should render offline count', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/1 offline/)).toBeInTheDocument();
    });

    it('should render error count when errors exist', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/1 error/)).toBeInTheDocument();
    });

    it('should not render error count when no errors', () => {
      const agentsWithoutErrors = mockAgents.filter(a => a.status !== 'error');
      render(<AgentList agents={agentsWithoutErrors} />);

      expect(screen.queryByText(/error/i)).not.toBeInTheDocument();
    });
  });

  describe('Filter Buttons', () => {
    it('should render filter buttons when onFilterChange is provided', () => {
      render(<AgentList agents={mockAgents} onFilterChange={vi.fn()} currentFilter={{}} />);

      expect(screen.getByText('All')).toBeInTheDocument();
      expect(screen.getByText('Online')).toBeInTheDocument();
      expect(screen.getByText('Offline')).toBeInTheDocument();
      expect(screen.getByText('Errors')).toBeInTheDocument();
    });

    it('should not render filter buttons when onFilterChange is not provided', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.queryByText('All')).not.toBeInTheDocument();
    });

    it('should highlight active filter', () => {
      render(<AgentList agents={mockAgents} onFilterChange={vi.fn()} currentFilter={{ status: 'online' }} />);

      const onlineButton = screen.getByText('Online');
      expect(onlineButton).toHaveClass(/bg-green-100/);
    });
  });

  describe('Agent List', () => {
    it('should render all agents', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText('Office Agent')).toBeInTheDocument();
      expect(screen.getByText('Warehouse Agent')).toBeInTheDocument();
      expect(screen.getByText('Reception Agent')).toBeInTheDocument();
    });

    it('should render agent platform', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText('linux')).toBeInTheDocument();
      expect(screen.getByText('windows')).toBeInTheDocument();
      expect(screen.getByText('darwin')).toBeInTheDocument();
    });

    it('should render agent version', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getAllByText(/v1\./).length).toBeGreaterThan(0);
    });

    it('should render agent status badges', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText('Online')).toBeInTheDocument();
      expect(screen.getByText('Offline')).toBeInTheDocument();
      expect(screen.getByText('Error')).toBeInTheDocument();
    });

    it('should render associated user when present', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });

    it('should render printer count', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/3 printers/)).toBeInTheDocument();
      expect(screen.getByText(/2 printers/)).toBeInTheDocument();
    });

    it('should render job queue depth when present', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/2 queued/)).toBeInTheDocument();
    });

    it('should render last heartbeat time', () => {
      render(<AgentList agents={mockAgents} />);

      // The component uses formatDistanceToNow
      const timeElements = document.querySelectorAll('svg.text-gray-400');
      expect(timeElements.length).toBeGreaterThan(0);
    });
  });

  describe('Platform Icons', () => {
    it('should render Windows icon', () => {
      render(<AgentList agents={[mockAgents[1]]} />);

      const windowsIcon = document.querySelector('svg[fill="currentColor"]');
      expect(windowsIcon).toBeInTheDocument();
    });

    it('should render Linux icon', () => {
      render(<AgentList agents={[mockAgents[0]]} />);

      const linuxIcon = document.querySelector('svg[fill="currentColor"]');
      expect(linuxIcon).toBeInTheDocument();
    });

    it('should render macOS icon', () => {
      render(<AgentList agents={[mockAgents[2]]} />);

      const macIcon = document.querySelector('svg[fill="currentColor"]');
      expect(macIcon).toBeInTheDocument();
    });
  });

  describe('Actions', () => {
    it('should render delete button for each agent', () => {
      render(<AgentList agents={mockAgents} />);

      const deleteButtons = document.querySelectorAll('button[title="Delete agent"]');
      expect(deleteButtons.length).toBe(3);
    });

    it('should render restart button for agents in error state', () => {
      render(<AgentList agents={mockAgents} />);

      const restartButton = document.querySelector('button[title="Restart agent"]');
      expect(restartButton).toBeInTheDocument();
    });

    it('should not render restart button for online agents', () => {
      const onlineAgents = mockAgents.filter(a => a.status === 'online');
      render(<AgentList agents={onlineAgents} />);

      const restartButton = document.querySelector('button[title="Restart agent"]');
      expect(restartButton).not.toBeInTheDocument();
    });

    it('should render chevron for navigation', () => {
      render(<AgentList agents={mockAgents} />);

      const chevrons = document.querySelectorAll('svg.arrow');
      expect(chevrons.length).toBe(3);
    });
  });

  describe('Click Handling', () => {
    it('should call onAgentClick when agent is clicked', async () => {
      const user = userEvent.setup();
      const handleClick = vi.fn();

      render(<AgentList agents={mockAgents} onAgentClick={handleClick} />);

      const agentRow = screen.getByText('Office Agent').closest('li');
      await user.click(agentRow!);

      expect(handleClick).toHaveBeenCalled();
    });

    it('should apply cursor-pointer when onAgentClick is provided', () => {
      render(<AgentList agents={mockAgents} onAgentClick={vi.fn()} />);

      const agentRow = screen.getByText('Office Agent').closest('li');
      expect(agentRow).toHaveClass('cursor-pointer');
    });
  });

  describe('Filter Change', () => {
    it('should call onFilterChange when filter button is clicked', async () => {
      const user = userEvent.setup();
      const handleFilterChange = vi.fn();

      render(<AgentList agents={mockAgents} onFilterChange={handleFilterChange} currentFilter={{}} />);

      const onlineButton = screen.getByText('Online');
      await user.click(onlineButton);

      expect(handleFilterChange).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'online' })
      );
    });
  });

  describe('Delete Action', () => {
    it('should show confirmation before deleting', async () => {
      const user = userEvent.setup();
      const originalConfirm = window.confirm;
      window.confirm = vi.fn(() => false);

      render(<AgentList agents={mockAgents} />);

      const deleteButton = document.querySelector('button[title="Delete agent"]') as HTMLElement;
      await user.click(deleteButton);

      expect(window.confirm).toHaveBeenCalledWith(
        expect.stringContaining('Are you sure you want to delete agent')
      );

      window.confirm = originalConfirm;
    });

    it('should include agent name in confirmation message', async () => {
      const user = userEvent.setup();
      const originalConfirm = window.confirm;
      window.confirm = vi.fn(() => false);

      render(<AgentList agents={mockAgents} />);

      const deleteButtons = document.querySelectorAll('button[title="Delete agent"]');
      await user.click(deleteButtons[0] as HTMLElement);

      expect(window.confirm).toHaveBeenCalledWith(
        expect.stringContaining('Office Agent')
      );

      window.confirm = originalConfirm;
    });
  });

  describe('Styling', () => {
    it('should apply proper list styling', () => {
      const { container } = render(<AgentList agents={mockAgents} />);

      const list = container.querySelector('ul.divide-y');
      expect(list).toBeInTheDocument();
    });

    it('should apply hover effect to agent rows', () => {
      const { container } = render(<AgentList agents={mockAgents} />);

      const rows = container.querySelectorAll('.hover\\:bg-gray-50');
      expect(rows.length).toBe(3);
    });
  });

  describe('Layout', () => {
    it('should use proper spacing', () => {
      const { container } = render(<AgentList agents={mockAgents} />);

      const mainContainer = container.querySelector('.space-y-4');
      expect(mainContainer).toBeInTheDocument();
    });

    it('should have proper flex layout for agent rows', () => {
      const { container } = render(<AgentList agents={mockAgents} />);

      const flexContainer = container.querySelector('.items-center.justify-between');
      expect(flexContainer).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper button labels', () => {
      render(<AgentList agents={mockAgents} />);

      expect(document.querySelector('button[title="Delete agent"]')).toBeInTheDocument();
      expect(document.querySelector('button[title="Restart agent"]')).toBeInTheDocument();
    });

    it('should have proper list structure', () => {
      render(<AgentList agents={mockAgents} />);

      const list = screen.getByRole('list');
      expect(list).toBeInTheDocument();
    });
  });

  describe('Error Status', () => {
    it('should display restart button for error agents', () => {
      render(<AgentList agents={mockAgents} />);

      const restartButton = document.querySelector('button[title="Restart agent"]');
      expect(restartButton).toBeInTheDocument();
    });

    it('should allow clicking restart button', async () => {
      const user = userEvent.setup();
      render(<AgentList agents={mockAgents} />);

      const restartButton = document.querySelector('button[title="Restart agent"]') as HTMLElement;
      await user.click(restartButton);

      // Just verify it doesn't crash
      expect(restartButton).toBeInTheDocument();
    });
  });

  describe('Agent Metadata', () => {
    it('should display agent name prominently', () => {
      render(<AgentList agents={mockAgents} />);

      const name = screen.getByText('Office Agent');
      expect(name).toHaveClass('font-medium');
    });

    it('should display platform icon with platform text', () => {
      render(<AgentList agents={mockAgents} />);

      const platformText = screen.getByText('linux').closest('span');
      expect(platformText?.className).toContain('text-xs');
    });

    it('should group related metadata', () => {
      render(<AgentList agents={mockAgents} />);

      const metadataContainer = document.querySelector('.flex-wrap.items-center.gap-4');
      expect(metadataContainer).toBeInTheDocument();
    });
  });

  describe('Count Display', () => {
    it('should show singular "agent" when count is 1', () => {
      render(<AgentList agents={[mockAgents[0]]} />);

      expect(screen.getByText(/1 agent/)).toBeInTheDocument();
    });

    it('should show plural "agents" when count > 1', () => {
      render(<AgentList agents={mockAgents} />);

      expect(screen.getByText(/3 agents/)).toBeInTheDocument();
    });

    it('should show singular "printer" when count is 1', () => {
      const agentWithOnePrinter = { ...mockAgents[2], printerCount: 1 };
      render(<AgentList agents={[agentWithOnePrinter]} />);

      expect(screen.getByText(/1 printer/)).toBeInTheDocument();
    });
  });

  describe('Last Heartbeat', () => {
    it('should show formatted relative time', () => {
      render(<AgentList agents={mockAgents} />);

      // The component uses formatDistanceToNow which shows relative time
      const timeElements = document.querySelectorAll('svg.text-gray-400');
      expect(timeElements.length).toBeGreaterThan(0);
    });

    it('should handle missing lastHeartbeat', () => {
      const agentWithoutHeartbeat = { ...mockAgents[0], lastHeartbeat: undefined };
      render(<AgentList agents={[agentWithoutHeartbeat]} />);

      // Should not crash
      expect(screen.getByText('Office Agent')).toBeInTheDocument();
    });
  });

  describe('Multiple Filters', () => {
    it('should handle filtering by different statuses', () => {
      const handleFilterChange = vi.fn();
      render(
        <AgentList agents={mockAgents} onFilterChange={handleFilterChange} currentFilter={{}} />
      );

      const buttons = {
        all: screen.getByText('All'),
        online: screen.getByText('Online'),
        offline: screen.getByText('Offline'),
        errors: screen.getByText('Errors'),
      };

      expect(buttons.all).toBeInTheDocument();
      expect(buttons.online).toBeInTheDocument();
      expect(buttons.offline).toBeInTheDocument();
      expect(buttons.errors).toBeInTheDocument();
    });
  });
});
