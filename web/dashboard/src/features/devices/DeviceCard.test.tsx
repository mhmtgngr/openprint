/**
 * DeviceCard Component Tests
 * Tests for device/agent/printer card component
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { DeviceCard } from './DeviceCard';
import type { DeviceAgent, DevicePrinter } from './types';

const mockAgent: DeviceAgent = {
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
  printerCount: 3,
};

const mockOfflineAgent: DeviceAgent = {
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
};

const mockPrinter: DevicePrinter = {
  id: 'printer-1',
  name: 'Office HP',
  agentName: 'Office Agent',
  agentId: 'agent-1',
  orgId: 'org-1',
  type: 'network',
  isOnline: true,
  isActive: true,
  capabilities: {
    supportsColor: true,
    supportsDuplex: true,
    supportedPaperSizes: ['A4', 'Letter'],
    resolution: '600x600',
  },
  createdAt: '2025-02-01T10:00:00Z',
  lastSeen: '2025-02-28T12:00:00Z',
};

const mockOfflinePrinter: DevicePrinter = {
  id: 'printer-2',
  name: 'Warehouse Brother',
  agentName: 'Warehouse Agent',
  agentId: 'agent-2',
  orgId: 'org-1',
  type: 'usb',
  isOnline: false,
  isActive: false,
  capabilities: {
    supportsColor: false,
    supportsDuplex: true,
    supportedPaperSizes: ['A4'],
    resolution: '600x600',
  },
  createdAt: '2025-02-10T14:00:00Z',
  lastSeen: '2025-02-28T08:00:00Z',
};

describe('DeviceCard', () => {
  describe('Agent Display', () => {
    it('should render agent name', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText('Office Agent')).toBeInTheDocument();
    });

    it('should render agent platform', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText('linux')).toBeInTheDocument();
    });

    it('should render agent version', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText('1.2.0')).toBeInTheDocument();
    });

    it('should render printer count', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText('3')).toBeInTheDocument();
    });

    it('should render online status for online agent', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText('Online')).toBeInTheDocument();
    });

    it('should render offline status for offline agent', () => {
      render(<DeviceCard device={mockOfflineAgent} />);

      expect(screen.getByText('Offline')).toBeInTheDocument();
    });

    it('should render agent icon', () => {
      const { container } = render(<DeviceCard device={mockAgent} />);

      const icon = container.querySelector('svg');
      expect(icon).toBeInTheDocument();
    });

    it('should render status dot', () => {
      const { container } = render(<DeviceCard device={mockAgent} />);

      const dot = container.querySelector('.w-2.h-2.rounded-full');
      expect(dot).toBeInTheDocument();
    });
  });

  describe('Printer Display', () => {
    it('should render printer name', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText('Office HP')).toBeInTheDocument();
    });

    it('should render printer type', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText('Laser')).toBeInTheDocument();
    });

    it('should render agent name for printer', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText('Office Agent')).toBeInTheDocument();
    });

    it('should render online status for online printer', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText('Online')).toBeInTheDocument();
    });

    it('should render offline status for offline printer', () => {
      render(<DeviceCard device={mockOfflinePrinter} />);

      expect(screen.getByText('Offline')).toBeInTheDocument();
    });

    it('should render printer icon', () => {
      const { container } = render(<DeviceCard device={mockPrinter} />);

      const icon = container.querySelector('svg');
      expect(icon).toBeInTheDocument();
    });

    it('should render Color capability badge', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText('Color')).toBeInTheDocument();
    });

    it('should render Duplex capability badge', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText('Duplex')).toBeInTheDocument();
    });

    it('should not render Color capability when not supported', () => {
      render(<DeviceCard device={mockOfflinePrinter} />);

      expect(screen.queryByText('Color')).not.toBeInTheDocument();
    });
  });

  describe('Status Indicators', () => {
    it('should apply green color for online status', () => {
      render(<DeviceCard device={mockAgent} />);

      const statusContainer = screen.getByText('Online').closest('span');
      expect(statusContainer?.className).toContain('text-green');
    });

    it('should apply gray color for offline status', () => {
      render(<DeviceCard device={mockOfflineAgent} />);

      const statusContainer = screen.getByText('Offline').closest('span');
      expect(statusContainer?.className).toContain('text-gray');
    });

    it('should render status icon background', () => {
      const { container } = render(<DeviceCard device={mockAgent} />);

      const iconBg = container.querySelector('.bg-green-100');
      expect(iconBg).toBeInTheDocument();
    });
  });

  describe('Actions', () => {
    it('should render delete button when onDelete is provided', () => {
      const handleDelete = vi.fn();
      render(<DeviceCard device={mockAgent} onDelete={handleDelete} />);

      const deleteButton = document.querySelector('button[title="Delete device"]');
      expect(deleteButton).toBeInTheDocument();
    });

    it('should not render delete button when onDelete is not provided', () => {
      render(<DeviceCard device={mockAgent} />);

      const deleteButton = document.querySelector('button[title="Delete device"]');
      expect(deleteButton).not.toBeInTheDocument();
    });

    it('should call onDelete when delete button is clicked', async () => {
      const user = userEvent.setup();
      const handleDelete = vi.fn();

      render(<DeviceCard device={mockAgent} onDelete={handleDelete} />);

      const deleteButton = document.querySelector('button[title="Delete device"]') as HTMLElement;
      await user.click(deleteButton);

      expect(handleDelete).toHaveBeenCalledTimes(1);
    });

    it('should render toggle button for printers when onToggleStatus is provided', () => {
      const handleToggle = vi.fn();
      render(<DeviceCard device={mockPrinter} onToggleStatus={handleToggle} />);

      const toggleButton = document.querySelector('button[role="switch"]');
      expect(toggleButton).toBeInTheDocument();
    });

    it('should not render toggle button for agents', () => {
      const handleToggle = vi.fn();
      render(<DeviceCard device={mockAgent} onToggleStatus={handleToggle} />);

      const toggleButton = document.querySelector('button[role="switch"]');
      expect(toggleButton).not.toBeInTheDocument();
    });

    it('should call onToggleStatus when toggle is clicked', async () => {
      const user = userEvent.setup();
      const handleToggle = vi.fn();

      render(<DeviceCard device={mockPrinter} onToggleStatus={handleToggle} />);

      const toggleButton = document.querySelector('.inline-flex.h-6.w-11') as HTMLElement;
      await user.click(toggleButton);

      expect(handleToggle).toHaveBeenCalledTimes(1);
    });

    it('should show toggle in active state when printer is active', () => {
      render(<DeviceCard device={mockPrinter} onToggleStatus={vi.fn()} />);

      const toggle = document.querySelector('.bg-blue-600');
      expect(toggle).toBeInTheDocument();
    });

    it('should show toggle in inactive state when printer is inactive', () => {
      render(<DeviceCard device={mockOfflinePrinter} onToggleStatus={vi.fn()} />);

      const toggle = document.querySelector('.bg-gray-200');
      expect(toggle).toBeInTheDocument();
    });

    it('should disable toggle when isToggling is true', () => {
      render(<DeviceCard device={mockPrinter} onToggleStatus={vi.fn()} isToggling={true} />);

      const toggle = document.querySelector('.inline-flex.h-6.w-11.disabled\\:opacity-50');
      expect(toggle).toBeInTheDocument();
    });
  });

  describe('Click Handling', () => {
    it('should call onClick when card is clicked', async () => {
      const user = userEvent.setup();
      const handleClick = vi.fn();

      render(<DeviceCard device={mockAgent} onClick={handleClick} />);

      const card = screen.getByText('Office Agent').closest('div');
      await user.click(card!);

      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should not call onClick when action button is clicked', async () => {
      const user = userEvent.setup();
      const handleClick = vi.fn();
      const handleDelete = vi.fn();

      render(<DeviceCard device={mockAgent} onClick={handleClick} onDelete={handleDelete} />);

      const deleteButton = document.querySelector('button[title="Delete device"]') as HTMLElement;
      await user.click(deleteButton);

      expect(handleClick).not.toHaveBeenCalled();
      expect(handleDelete).toHaveBeenCalledTimes(1);
    });

    it('should apply cursor-pointer style when onClick is provided', () => {
      const { container } = render(<DeviceCard device={mockAgent} onClick={vi.fn()} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('cursor-pointer');
    });
  });

  describe('Details Section', () => {
    it('should render version for agents', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText(/Version/i)).toBeInTheDocument();
      expect(screen.getByText('1.2.0')).toBeInTheDocument();
    });

    it('should render printers count for agents', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText(/Printers/i)).toBeInTheDocument();
    });

    it('should render type for printers', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText(/Type/i)).toBeInTheDocument();
    });

    it('should render agent for printers', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText(/Agent/i)).toBeInTheDocument();
    });

    it('should render capabilities section for printers', () => {
      render(<DeviceCard device={mockPrinter} />);

      expect(screen.getByText(/Capabilities/i)).toBeInTheDocument();
    });
  });

  describe('Footer', () => {
    it('should render last seen section', () => {
      render(<DeviceCard device={mockAgent} />);

      expect(screen.getByText(/Last seen/i)).toBeInTheDocument();
    });

    it('should render uptime when available', () => {
      const agentWithUptime = { ...mockAgent, uptime: '2 days' };
      render(<DeviceCard device={agentWithUptime} />);

      expect(screen.getByText('2 days')).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply card base styling', () => {
      const { container } = render(<DeviceCard device={mockAgent} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('bg-white');
      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('border');
    });

    it('should apply hover effect', () => {
      const { container } = render(<DeviceCard device={mockAgent} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('hover:shadow-md');
    });

    it('should apply status-specific icon background color', () => {
      const { container } = render(<DeviceCard device={mockAgent} />);

      expect(container.querySelector('.bg-green-100')).toBeInTheDocument();
    });

    it('should apply offline icon background for offline devices', () => {
      const { container } = render(<DeviceCard device={mockOfflineAgent} />);

      expect(container.querySelector('.bg-gray-100')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper button labels', () => {
      const handleDelete = vi.fn();
      render(<DeviceCard device={mockAgent} onDelete={handleDelete} />);

      expect(document.querySelector('button[title="Delete device"]')).toBeInTheDocument();
    });

    it('should have proper heading for device name', () => {
      render(<DeviceCard device={mockAgent} />);

      const name = screen.getByText('Office Agent');
      expect(name.tagName).toBe('H3');
    });
  });

  describe('Error Status', () => {
    it('should render error status for error agent', () => {
      const errorAgent: DeviceAgent = {
        ...mockAgent,
        status: 'error',
      };

      render(<DeviceCard device={errorAgent} />);

      expect(screen.getByText('Error')).toBeInTheDocument();
    });

    it('should apply error color for error status', () => {
      const errorAgent: DeviceAgent = {
        ...mockAgent,
        status: 'error',
      };

      const { container } = render(<DeviceCard device={errorAgent} />);

      expect(container.querySelector('.bg-red-100')).toBeInTheDocument();
    });
  });
});
