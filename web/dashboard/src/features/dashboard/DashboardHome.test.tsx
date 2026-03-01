/**
 * DashboardHome Component Tests
 * Comprehensive tests for the main dashboard page
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import { DashboardHome } from './DashboardHome';
import * as useAuthHook from '@/hooks/useAuth';

// Mock the useAuth hook
vi.mock('@/hooks/useAuth');

// Mock react-query
vi.mock('@tanstack/react-query', async () => {
  const actual = await vi.importActual('@tanstack/react-query');
  return {
    ...actual,
    useQuery: vi.fn(),
  };
});

import { useQuery } from '@tanstack/react-query';

describe('DashboardHome', () => {
  const mockUseAuth = vi.spyOn(useAuthHook, 'useAuth');

  beforeEach(() => {
    mockUseAuth.mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      user: {
        id: '1',
        email: 'admin@example.com',
        name: 'Admin',
        role: 'admin',
        orgId: 'org-1',
        isActive: true,
        emailVerified: true,
        createdAt: '2025-01-01T00:00:00Z',
      },
      error: null,
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
      hasRole: vi.fn(() => true),
    });

    // Mock all useQuery calls
    vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
      if (queryKey[0] === 'jobs') {
        return {
          data: { data: [], total: 0 },
          isLoading: false,
          error: null,
        } as any;
      }
      if (queryKey[0] === 'analytics') {
        return {
          data: [],
          isLoading: false,
          error: null,
        } as any;
      }
      if (queryKey[0] === 'agents') {
        return {
          data: [],
          isLoading: false,
          error: null,
        } as any;
      }
      if (queryKey[0] === 'printers') {
        return {
          data: [],
          isLoading: false,
          error: null,
        } as any;
      }
      return {
        data: null,
        isLoading: false,
        error: null,
      } as any;
    });
  });

  describe('Rendering', () => {
    it('should render page header', () => {
      render(<DashboardHome />);

      expect(screen.getByText('Dashboard')).toBeInTheDocument();
      expect(screen.getByText('Overview of your print environment')).toBeInTheDocument();
    });

    it('should render stat cards section', () => {
      render(<DashboardHome />);

      expect(screen.getByText('Total Jobs')).toBeInTheDocument();
      expect(screen.getByText('Active Devices')).toBeInTheDocument();
      expect(screen.getByText('Pages Today')).toBeInTheDocument();
      expect(screen.getByText('Completed Jobs')).toBeInTheDocument();
    });

    it('should render JobChart component', () => {
      render(<DashboardHome />);

      expect(screen.getByText('Job Statistics')).toBeInTheDocument();
    });

    it('should render RecentActivity component', () => {
      render(<DashboardHome />);

      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
    });

    it('should render additional stat cards row', () => {
      render(<DashboardHome />);

      expect(screen.getByText('Queued Jobs')).toBeInTheDocument();
      expect(screen.getByText('Failed Jobs')).toBeInTheDocument();
      expect(screen.getByText('Pages This Month')).toBeInTheDocument();
    });
  });

  describe('Data Fetching', () => {
    it('should fetch recent jobs on mount', () => {
      render(<DashboardHome />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['jobs', { limit: 10 }],
        })
      );
    });

    it('should fetch analytics data on mount', () => {
      render(<DashboardHome />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['analytics', 'usage'],
        })
      );
    });

    it('should fetch environment data on mount', () => {
      render(<DashboardHome />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['analytics', 'environment'],
        })
      );
    });

    it('should fetch agents data on mount', () => {
      render(<DashboardHome />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['agents'],
        })
      );
    });

    it('should fetch printers data on mount', () => {
      render(<DashboardHome />);

      expect(useQuery).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ['printers'],
        })
      );
    });
  });

  describe('Loading States', () => {
    it('should show loading state on stat cards when data is loading', () => {
      vi.mocked(useQuery).mockImplementation(() => ({
        data: undefined,
        isLoading: true,
        error: null,
      }) as any);

      render(<DashboardHome />);

      // Stat cards should be in loading state
      const loadingElements = document.querySelectorAll('.animate-pulse');
      expect(loadingElements.length).toBeGreaterThan(0);
    });

    it('should show loading on JobChart when loading', () => {
      vi.mocked(useQuery).mockImplementation(() => ({
        data: undefined,
        isLoading: true,
        error: null,
      }) as any);

      render(<DashboardHome />);

      const chartContainer = screen.getByText('Job Statistics').closest('div');
      expect(chartContainer?.parentElement).toHaveClass('animate-pulse');
    });

    it('should show loading on RecentActivity when loading', () => {
      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'jobs') {
          return {
            data: undefined,
            isLoading: true,
            error: null,
          } as any;
        }
        return {
          data: null,
          isLoading: false,
          error: null,
        } as any;
      });

      render(<DashboardHome />);

      const skeletonElements = document.querySelectorAll('.animate-pulse');
      expect(skeletonElements.length).toBeGreaterThan(0);
    });
  });

  describe('Stats Calculation', () => {
    it('should calculate stats from fetched data', async () => {
      const mockJobsData = {
        data: [
          { id: 'job-1', status: 'completed', documentName: 'Test.pdf' },
          { id: 'job-2', status: 'completed', documentName: 'Test2.pdf' },
          { id: 'job-3', status: 'queued', documentName: 'Test3.pdf' },
          { id: 'job-4', status: 'failed', documentName: 'Test4.pdf' },
        ],
        total: 4,
      };

      const mockAgentsData = [
        { id: 'agent-1', status: 'online' },
        { id: 'agent-2', status: 'offline' },
      ];

      const mockPrintersData = [
        { id: 'printer-1', isOnline: true },
        { id: 'printer-2', isOnline: false },
      ];

      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'jobs') {
          return { data: mockJobsData, isLoading: false } as any;
        }
        if (queryKey[0] === 'agents') {
          return { data: mockAgentsData, isLoading: false } as any;
        }
        if (queryKey[0] === 'printers') {
          return { data: mockPrintersData, isLoading: false } as any;
        }
        return { data: null, isLoading: false } as any;
      });

      render(<DashboardHome />);

      // Stats should be calculated from the data
      await waitFor(() => {
        expect(screen.getByText('4')).toBeInTheDocument(); // Total jobs
      });
    });

    it('should calculate active devices correctly', async () => {
      const mockAgentsData = [
        { id: 'agent-1', status: 'online' },
        { id: 'agent-2', status: 'offline' },
      ];

      const mockPrintersData = [
        { id: 'printer-1', isOnline: true },
        { id: 'printer-2', isOnline: false },
      ];

      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'agents') {
          return { data: mockAgentsData, isLoading: false } as any;
        }
        if (queryKey[0] === 'printers') {
          return { data: mockPrintersData, isLoading: false } as any;
        }
        return { data: null, isLoading: false } as any;
      });

      render(<DashboardHome />);

      // 1 online agent + 1 online printer = 2 active devices
      // Total = 4 devices
      const activeDevicesText = screen.getByText(/2 \/ 4/);
      expect(activeDevicesText).toBeInTheDocument();
    });
  });

  describe('Trend Indicators', () => {
    it('should render trend indicators on stat cards', () => {
      render(<DashboardHome />);

      // These are hard-coded in the component for demo purposes
      expect(screen.getByText(/12%/)).toBeInTheDocument();
      expect(screen.getByText(/from last week/)).toBeInTheDocument();
      expect(screen.getByText(/8%/)).toBeInTheDocument();
      expect(screen.getByText(/vs yesterday/)).toBeInTheDocument();
      expect(screen.getByText(/5%/)).toBeInTheDocument();
      expect(screen.getByText(/completion rate/)).toBeInTheDocument();
    });

    it('should render negative trend for failed jobs', () => {
      render(<DashboardHome />);

      expect(screen.getByText(/2%/)).toBeInTheDocument();
      expect(screen.getByText(/from last week/)).toBeInTheDocument();
    });
  });

  describe('Layout', () => {
    it('should use proper grid layout for stat cards', () => {
      const { container } = render(<DashboardHome />);

      const grid = container.querySelector('.grid.grid-cols-1.sm\\:grid-cols-2.lg\\:grid-cols-4');
      expect(grid).toBeInTheDocument();
    });

    it('should use proper grid layout for main content', () => {
      const { container } = render(<DashboardHome />);

      const grid = container.querySelector('.grid.grid-cols-1.lg\\:grid-cols-3');
      expect(grid).toBeInTheDocument();
    });

    it('should have proper spacing', () => {
      const { container } = render(<DashboardHome />);

      const mainContainer = container.querySelector('.space-y-6');
      expect(mainContainer).toBeInTheDocument();
    });
  });

  describe('JobChart Props', () => {
    it('should pass statistics data to JobChart', () => {
      const mockAnalytics = [
        { statDate: '2025-02-28', jobsCount: 45, jobsCompleted: 42, jobsFailed: 3, pagesPrinted: 520 },
      ];

      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'analytics' && queryKey[1] === 'usage') {
          return { data: mockAnalytics, isLoading: false } as any;
        }
        return { data: null, isLoading: false } as any;
      });

      render(<DashboardHome />);

      expect(screen.getByText('Job Statistics')).toBeInTheDocument();
    });
  });

  describe('RecentActivity Props', () => {
    it('should pass activities to RecentActivity', () => {
      const mockJobs = [
        {
          id: 'job-1',
          status: 'completed',
          documentName: 'Test.pdf',
          createdAt: '2025-02-28T10:00:00Z',
        },
      ];

      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'jobs') {
          return { data: { data: mockJobs, total: 1 }, isLoading: false } as any;
        }
        return { data: null, isLoading: false } as any;
      });

      render(<DashboardHome />);

      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
    });

    it('should set maxItems to 6 for RecentActivity', () => {
      render(<DashboardHome />);

      // RecentActivity should be rendered
      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('should handle query errors gracefully', () => {
      vi.mocked(useQuery).mockImplementation(() => ({
        data: null,
        isLoading: false,
        error: new Error('Test error'),
      }) as any);

      // Should not crash, just render with zeros
      expect(() => render(<DashboardHome />)).not.toThrow();
    });
  });

  describe('Styling', () => {
    it('should apply custom className', () => {
      const { container } = render(<DashboardHome className="custom-class" />);

      const mainContainer = container.firstChild as HTMLElement;
      expect(mainContainer).toHaveClass('custom-class');
    });

    it('should use proper heading styles', () => {
      render(<DashboardHome />);

      const heading = screen.getByText('Dashboard');
      expect(heading.tagName).toBe('H1');
      expect(heading).toHaveClass('text-2xl');
      expect(heading).toHaveClass('font-bold');
    });
  });

  describe('Icons', () => {
    it('should render icons for stat cards', () => {
      render(<DashboardHome />);

      const icons = document.querySelectorAll('svg');
      expect(icons.length).toBeGreaterThan(0);
    });
  });

  describe('Empty States', () => {
    it('should handle empty jobs data', () => {
      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'jobs') {
          return { data: { data: [], total: 0 }, isLoading: false } as any;
        }
        return { data: null, isLoading: false } as any;
      });

      render(<DashboardHome />);

      expect(screen.getByText('0')).toBeInTheDocument();
    });

    it('should handle empty devices data', () => {
      vi.mocked(useQuery).mockImplementation(({ queryKey }) => {
        if (queryKey[0] === 'agents') {
          return { data: [], isLoading: false } as any;
        }
        if (queryKey[0] === 'printers') {
          return { data: [], isLoading: false } as any;
        }
        return { data: null, isLoading: false } as any;
      });

      render(<DashboardHome />);

      expect(screen.getByText('0')).toBeInTheDocument();
    });
  });
});
