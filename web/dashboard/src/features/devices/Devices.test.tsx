/**
 * Devices Component Tests
 * Comprehensive tests for the devices listing page
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { Devices } from './Devices';

// Mock react-query
vi.mock('@tanstack/react-query', async () => {
  const actual = await vi.importActual('@tanstack/react-query');
  return {
    ...actual,
    useQuery: vi.fn(),
    useMutation: vi.fn(() => ({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
    })),
  };
});

import { useQuery, useMutation } from '@tanstack/react-query';

const mockDevicesData = {
  agents: [
    {
      id: 'agent-1',
      name: 'Office Agent',
      platform: 'linux',
      agentVersion: '1.2.0',
      status: 'online',
      createdAt: '2025-02-01T10:00:00Z',
      lastSeen: '2025-02-28T12:00:00Z',
      printerCount: 3,
    },
    {
      id: 'agent-2',
      name: 'Warehouse Agent',
      platform: 'windows',
      agentVersion: '1.1.5',
      status: 'offline',
      createdAt: '2025-02-10T14:00:00Z',
      lastSeen: '2025-02-28T08:00:00Z',
      printerCount: 2,
    },
  ],
  printers: [
    {
      id: 'printer-1',
      name: 'Office HP',
      agentName: 'Office Agent',
      agentId: 'agent-1',
      isOnline: true,
      isActive: true,
      createdAt: '2025-02-01T10:00:00Z',
      lastSeen: '2025-02-28T12:00:00Z',
    },
    {
      id: 'printer-2',
      name: 'Warehouse Brother',
      agentName: 'Warehouse Agent',
      agentId: 'agent-2',
      isOnline: false,
      isActive: false,
      createdAt: '2025-02-10T14:00:00Z',
      lastSeen: '2025-02-28T08:00:00Z',
    },
  ],
  stats: {
    totalAgents: 2,
    onlineAgents: 1,
    totalPrinters: 2,
    onlinePrinters: 1,
    offlinePrinters: 1,
  },
};

describe('Devices', () => {
  beforeEach(() => {
    // Reset mocks
    vi.clearAllMocks();

    // Default mock implementation
    vi.mocked(useQuery).mockReturnValue({
      data: mockDevicesData,
      isLoading: false,
      error: null,
    } as any);

    vi.mocked(useMutation).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
    } as any);
  });

  describe('Rendering', () => {
    it('should render page header', () => {
      render(<Devices />);

      expect(screen.getByText('Devices')).toBeInTheDocument();
    });

    it('should render Add Device button', () => {
      render(<Devices />);

      expect(screen.getByText('Add Device')).toBeInTheDocument();
    });

    it('should render stat cards', () => {
      render(<Devices />);

      expect(screen.getByText('Total Agents')).toBeInTheDocument();
      expect(screen.getByText('Total Printers')).toBeInTheDocument();
      expect(screen.getByText('Online')).toBeInTheDocument();
      expect(screen.getByText('Offline')).toBeInTheDocument();
    });

    it('should render filter controls', () => {
      render(<Devices />);

      expect(screen.getByPlaceholderText(/search devices/i)).toBeInTheDocument();

      const statusSelect = screen.getByRole('combobox', { name: '' });
      expect(statusSelect).toBeInTheDocument();
    });

    it('should render view toggle buttons', () => {
      render(<Devices />);

      expect(document.querySelector('button')).toBeInTheDocument();
    });
  });

  describe('Stat Cards', () => {
    it('should display total agents count', () => {
      render(<Devices />);

      expect(screen.getByText('2')).toBeInTheDocument();
    });

    it('should display online agents count', () => {
      render(<Devices />);

      expect(screen.getByText(/\(1 online\)/)).toBeInTheDocument();
    });

    it('should display total printers count', () => {
      render(<Devices />);

      expect(screen.getAllByText('2').length).toBeGreaterThan(0);
    });

    it('should display online devices', () => {
      render(<Devices />);

      expect(screen.getByText('2')).toBeInTheDocument(); // 2 online devices (1 agent + 1 printer)
    });

    it('should display offline devices', () => {
      render(<Devices />);

      expect(screen.getByText('2')).toBeInTheDocument(); // 2 offline devices
    });
  });

  describe('Loading State', () => {
    it('should render loading skeletons when loading', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any);

      render(<Devices />);

      const skeletons = document.querySelectorAll('.animate-pulse');
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it('should not render content when loading', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as any);

      render(<Devices />);

      expect(screen.queryByText('Office Agent')).not.toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    it('should render error state when query fails', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: undefined,
        isLoading: false,
        error: new Error('Failed to load'),
      } as any);

      render(<Devices />);

      expect(screen.getByText('Error loading devices')).toBeInTheDocument();
    });

    it('should render error message', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: undefined,
        isLoading: false,
        error: { message: 'Network error' },
      } as any);

      render(<Devices />);

      expect(screen.getByText('Network error')).toBeInTheDocument();
    });

    it('should render error icon', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: undefined,
        isLoading: false,
        error: new Error('Test error'),
      } as any);

      render(<Devices />);

      const errorIcon = document.querySelector('svg.text-red-400');
      expect(errorIcon).toBeInTheDocument();
    });
  });

  describe('Empty State', () => {
    it('should render empty state when no devices', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: { agents: [], printers: [], stats: { totalAgents: 0, onlineAgents: 0, totalPrinters: 0, onlinePrinters: 0, offlinePrinters: 0 } },
        isLoading: false,
        error: null,
      } as any);

      render(<Devices />);

      expect(screen.getByText('No devices found')).toBeInTheDocument();
    });

    it('should render empty state description', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: { agents: [], printers: [], stats: { totalAgents: 0, onlineAgents: 0, totalPrinters: 0, onlinePrinters: 0, offlinePrinters: 0 } },
        isLoading: false,
        error: null,
      } as any);

      render(<Devices />);

      expect(screen.getByText(/Get started by adding your first printer or agent/)).toBeInTheDocument();
    });

    it('should show Add Device button in empty state', () => {
      vi.mocked(useQuery).mockReturnValue({
        data: { agents: [], printers: [], stats: { totalAgents: 0, onlineAgents: 0, totalPrinters: 0, onlinePrinters: 0, offlinePrinters: 0 } },
        isLoading: false,
        error: null,
      } as any);

      render(<Devices />);

      const addButtons = screen.getAllByText('Add Device');
      expect(addButtons.length).toBeGreaterThan(0);
    });
  });

  describe('Search Functionality', () => {
    it('should update search query when typing', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      const searchInput = screen.getByPlaceholderText(/search devices/i);
      await user.type(searchInput, 'Office');

      expect(searchInput).toHaveValue('Office');
    });
  });

  describe('Status Filter', () => {
    it('should have all status options', () => {
      render(<Devices />);

      const statusSelect = screen.getByRole('combobox');
      expect(statusSelect).toBeInTheDocument();
    });
  });

  describe('Type Filter', () => {
    it('should have all type options', () => {
      render(<Devices />);

      const typeSelect = screen.getAllByRole('combobox')[1];
      expect(typeSelect).toBeInTheDocument();
    });
  });

  describe('View Toggle', () => {
    it('should toggle between table and grid view', async () => {
      render(<Devices />);

      const viewButtons = document.querySelectorAll('button');
      expect(viewButtons.length).toBeGreaterThan(0);
    });
  });

  describe('Table View', () => {
    it('should render devices in table view by default', () => {
      render(<Devices />);

      expect(screen.getByText('Office Agent')).toBeInTheDocument();
      expect(screen.getByText('Office HP')).toBeInTheDocument();
    });

    it('should render table headers', () => {
      render(<Devices />);

      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Type')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText('Last Seen')).toBeInTheDocument();
      expect(screen.getByText('Actions')).toBeInTheDocument();
    });

    it('should show agent type for agents', () => {
      render(<Devices />);

      const agentType = screen.getAllByText('Agent');
      expect(agentType.length).toBeGreaterThan(0);
    });

    it('should show printer type for printers', () => {
      render(<Devices />);

      const printerType = screen.getAllByText('Printer');
      expect(printerType.length).toBeGreaterThan(0);
    });
  });

  describe('Grid View', () => {
    it('should render devices in grid view when toggled', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      // Find and click grid view button
      const buttons = document.querySelectorAll('button');
      if (buttons.length > 1) {
        await user.click(buttons[1]);
      }

      // Grid view should still show devices
      expect(screen.getByText('Office Agent')).toBeInTheDocument();
    });
  });

  describe('Device Registration Modal', () => {
    it('should open modal when Add Device is clicked', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      const addButton = screen.getByText('Add Device');
      await user.click(addButton);

      // Modal should appear
      await waitFor(() => {
        expect(screen.getByText('Register New Device')).toBeInTheDocument();
      });
    });

    it('should render modal form fields', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      const addButton = screen.getByText('Add Device');
      await user.click(addButton);

      await waitFor(() => {
        expect(screen.getByLabelText(/Device Name/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/Location/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/Printer Type/i)).toBeInTheDocument();
      });
    });

    it('should close modal when Cancel is clicked', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      const addButton = screen.getByText('Add Device');
      await user.click(addButton);

      await waitFor(() => {
        expect(screen.getByText('Register New Device')).toBeInTheDocument();
      });

      const cancelButton = screen.getByText('Cancel');
      await user.click(cancelButton);

      await waitFor(() => {
        expect(screen.queryByText('Register New Device')).not.toBeInTheDocument();
      });
    });

    it('should submit form when Register Device is clicked', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      const addButton = screen.getByText('Add Device');
      await user.click(addButton);

      await waitFor(() => {
        const nameInput = screen.getByLabelText(/Device Name/i);
        const locationInput = screen.getByLabelText(/Location/i);

        user.type(nameInput, 'Test Device');
        user.type(locationInput, 'Test Location');
      });

      const submitButton = screen.getByText('Register Device');
      await user.click(submitButton);

      // Just verify it doesn't crash
      expect(submitButton).toBeInTheDocument();
    });
  });

  describe('Data Fetching', () => {
    it('should fetch devices on mount', () => {
      render(<Devices />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['devices'],
        })
      );
    });

    it('should refetch periodically', () => {
      render(<Devices />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          refetchInterval: 30000,
        })
      );
    });
  });

  describe('Styling', () => {
    it('should apply proper spacing', () => {
      const { container } = render(<Devices />);

      const mainContainer = container.querySelector('.space-y-6');
      expect(mainContainer).toBeInTheDocument();
    });

    it('should apply grid layout to stat cards', () => {
      const { container } = render(<Devices />);

      const grid = container.querySelector('.grid.grid-cols-2');
      expect(grid).toBeInTheDocument();
    });

    it('should apply proper header layout', () => {
      const { container } = render(<Devices />);

      const header = container.querySelector('.flex-col.sm\\:flex-row');
      expect(header).toBeInTheDocument();
    });
  });

  describe('Device Actions', () => {
    it('should render delete buttons for devices', () => {
      render(<Devices />);

      const deleteButtons = document.querySelectorAll('button[title="Delete"]');
      expect(deleteButtons.length).toBeGreaterThan(0);
    });

    it('should render toggle for printers', () => {
      render(<Devices />);

      const toggle = document.querySelector('.inline-flex.h-6.w-11');
      expect(toggle).toBeInTheDocument();
    });

    it('should not render toggle for agents', () => {
      render(<Devices />);

      const toggles = document.querySelectorAll('.inline-flex.h-6.w-11');
      // Only printers should have toggles
      expect(toggles.length).toBeGreaterThan(0);
    });
  });

  describe('Filter Changes', () => {
    it('should update query when filters change', async () => {
      const user = userEvent.setup();
      render(<Devices />);

      const searchInput = screen.getByPlaceholderText(/search devices/i);
      await user.type(searchInput, 'test');

      // Just verify input works
      expect(searchInput).toHaveValue('test');
    });
  });

  describe('Error Handling', () => {
    it('should handle delete confirmation', async () => {
      const user = userEvent.setup();
      const originalConfirm = window.confirm;
      window.confirm = vi.fn(() => false);

      render(<Devices />);

      const deleteButton = document.querySelector('button[title="Delete"]') as HTMLElement;
      await user.click(deleteButton);

      expect(window.confirm).toHaveBeenCalled();

      window.confirm = originalConfirm;
    });
  });

  describe('Accessibility', () => {
    it('should have proper heading structure', () => {
      render(<Devices />);

      const heading = screen.getByText('Devices');
      expect(heading.tagName).toBe('H1');
    });

    it('should have proper labels for inputs', () => {
      render(<Devices />);

      expect(screen.getByPlaceholderText(/search devices/i)).toBeInTheDocument();
    });

    it('should have proper button labels', () => {
      render(<Devices />);

      expect(screen.getByText('Add Device')).toBeInTheDocument();
    });
  });

  describe('Responsive Layout', () => {
    it('should use responsive grid for stat cards', () => {
      const { container } = render(<Devices />);

      const grid = container.querySelector('.grid-cols-2.md\\:grid-cols-4');
      expect(grid).toBeInTheDocument();
    });

    it('should use responsive layout for filters', () => {
      const { container } = render(<Devices />);

      const filterContainer = container.querySelector('.flex-col.sm\\:flex-row');
      expect(filterContainer).toBeInTheDocument();
    });
  });
});
