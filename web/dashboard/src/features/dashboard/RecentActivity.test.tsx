/**
 * RecentActivity Component Tests
 * Unit tests for the recent activity feed component
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { RecentActivity } from './RecentActivity';
import type { Activity } from './types';

const mockActivities: Activity[] = [
  {
    id: 'act-1',
    type: 'job_completed',
    jobId: 'job-1',
    jobName: 'Quarterly Report.pdf',
    printerName: 'Office HP',
    timestamp: '2025-02-28T10:30:00Z',
  },
  {
    id: 'act-2',
    type: 'job_failed',
    jobId: 'job-2',
    jobName: 'Invoice.pdf',
    printerName: 'Design Canon',
    errorMessage: 'Out of paper',
    timestamp: '2025-02-28T09:15:00Z',
  },
  {
    id: 'act-3',
    type: 'agent_connected',
    agentId: 'agent-1',
    timestamp: '2025-02-28T08:00:00Z',
  },
  {
    id: 'act-4',
    type: 'printer_offline',
    printerId: 'printer-1',
    printerName: 'Warehouse Printer',
    timestamp: '2025-02-28T07:30:00Z',
  },
  {
    id: 'act-5',
    type: 'user_joined',
    userName: 'John Doe',
    timestamp: '2025-02-28T07:00:00Z',
  },
  {
    id: 'act-6',
    type: 'job_created',
    jobId: 'job-3',
    jobName: 'Presentation.pptx',
    timestamp: '2025-02-28T06:45:00Z',
  },
];

describe('RecentActivity', () => {
  describe('Rendering', () => {
    it('should render component container', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
    });

    it('should render activity list', () => {
      render(<RecentActivity activities={mockActivities} />);

      const list = document.querySelector('ul.divide-y');
      expect(list).toBeInTheDocument();
    });

    it('should render activity items', () => {
      render(<RecentActivity activities={mockActivities.slice(0, 3)} />);

      expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      expect(screen.getByText('Invoice.pdf')).toBeInTheDocument();
    });
  });

  describe('Empty State', () => {
    it('should render empty state when no activities', () => {
      render(<RecentActivity activities={[]} />);

      expect(screen.getByText('No recent activity')).toBeInTheDocument();
    });

    it('should render empty state icon', () => {
      render(<RecentActivity activities={[]} />);

      const icon = document.querySelector('svg.text-gray-400');
      expect(icon).toBeInTheDocument();
    });
  });

  describe('Activity Display', () => {
    it('should display job name for job activities', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      expect(screen.getByText('Invoice.pdf')).toBeInTheDocument();
      expect(screen.getByText('Presentation.pptx')).toBeInTheDocument();
    });

    it('should display activity type label', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('Job Completed')).toBeInTheDocument();
      expect(screen.getByText('Job Failed')).toBeInTheDocument();
      expect(screen.getByText('Agent Connected')).toBeInTheDocument();
    });

    it('should display printer name for job activities', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getAllByText(/on Office HP/i).length).toBeGreaterThan(0);
      expect(screen.getAllByText(/on Design Canon/i).length).toBeGreaterThan(0);
    });

    it('should display user name for user activities', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('John Doe')).toBeInTheDocument();
    });

    it('should display error details when present', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('Out of paper')).toBeInTheDocument();
    });

    it('should limit items to maxItems prop', () => {
      render(<RecentActivity activities={mockActivities} maxItems={3} />);

      // Should only show 3 activities
      expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      // The 4th activity should not be visible
      const warehouseText = screen.queryByText('Warehouse Printer');
      expect(warehouseText).not.toBeInTheDocument();
    });

    it('should respect maxItems default of 5', () => {
      render(<RecentActivity activities={mockActivities} />);

      // With 6 activities and default max of 5, the 6th shouldn't show
      expect(screen.queryByText('Presentation.pptx')).not.toBeInTheDocument();
    });
  });

  describe('Activity Status Indicators', () => {
    it('should render status dots', () => {
      render(<RecentActivity activities={mockActivities} />);

      const dots = document.querySelectorAll('.w-2.h-2.rounded-full');
      expect(dots.length).toBeGreaterThan(0);
    });

    it('should apply correct color for job_completed', () => {
      render(<RecentActivity activities={mockActivities.slice(0, 1)} />);

      const dot = document.querySelector('.bg-green-500');
      expect(dot).toBeInTheDocument();
    });

    it('should apply correct color for job_failed', () => {
      const failedActivities = mockActivities.filter(a => a.type === 'job_failed');
      render(<RecentActivity activities={failedActivities} />);

      const dot = document.querySelector('.bg-red-500');
      expect(dot).toBeInTheDocument();
    });

    it('should apply correct color for agent_connected', () => {
      const agentActivities = mockActivities.filter(a => a.type === 'agent_connected');
      render(<RecentActivity activities={agentActivities} />);

      const dot = document.querySelector('.bg-green-500');
      expect(dot).toBeInTheDocument();
    });

    it('should apply correct color for printer_offline', () => {
      const printerActivities = mockActivities.filter(a => a.type === 'printer_offline');
      render(<RecentActivity activities={printerActivities} />);

      const dot = document.querySelector('.bg-orange-500');
      expect(dot).toBeInTheDocument();
    });
  });

  describe('Timestamp Display', () => {
    it('should display relative timestamps', () => {
      render(<RecentActivity activities={mockActivities} />);

      // Just check that some time elements are present
      const timeElements = document.querySelectorAll('text-xs.text-gray-500');
      expect(timeElements.length).toBeGreaterThan(0);
    });

    it('should show "Just now" for very recent activities', () => {
      const recentActivity: Activity = {
        id: 'act-recent',
        type: 'job_created',
        timestamp: new Date().toISOString(),
      };

      render(<RecentActivity activities={[recentActivity]} />);

      expect(screen.getByText('Just now')).toBeInTheDocument();
    });
  });

  describe('Loading State', () => {
    it('should render loading skeleton', () => {
      render(<RecentActivity activities={[]} loading={true} maxItems={3} />);

      // Should have skeleton items
      const skeletons = document.querySelectorAll('.animate-pulse');
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it('should render correct number of skeleton items', () => {
      render(<RecentActivity activities={[]} loading={true} maxItems={3} />);

      const skeletons = document.querySelectorAll('.animate-pulse');
      expect(skeletons.length).toBe(3); // maxItems controls skeleton count
    });

    it('should not show empty state when loading', () => {
      render(<RecentActivity activities={[]} loading={true} />);

      expect(screen.queryByText('No recent activity')).not.toBeInTheDocument();
    });

    it('should not render activities when loading', () => {
      render(<RecentActivity activities={mockActivities} loading={true} />);

      expect(screen.queryByText('Quarterly Report.pdf')).not.toBeInTheDocument();
    });
  });

  describe('View All Button', () => {
    it('should render view all button when activities exceed maxItems', () => {
      render(<RecentActivity activities={mockActivities} maxItems={3} onViewAll={vi.fn()} />);

      expect(screen.getByText('View all')).toBeInTheDocument();
    });

    it('should not render view all button when activities fit maxItems', () => {
      render(<RecentActivity activities={mockActivities.slice(0, 3)} maxItems={5} onViewAll={vi.fn()} />);

      expect(screen.queryByText('View all')).not.toBeInTheDocument();
    });

    it('should not render view all button when onViewAll not provided', () => {
      render(<RecentActivity activities={mockActivities} maxItems={3} />);

      expect(screen.queryByText('View all')).not.toBeInTheDocument();
    });

    it('should call onViewAll when button is clicked', async () => {
      const user = userEvent.setup();
      const handleViewAll = vi.fn();

      render(<RecentActivity activities={mockActivities} maxItems={3} onViewAll={handleViewAll} />);

      await user.click(screen.getByText('View all'));

      expect(handleViewAll).toHaveBeenCalledTimes(1);
    });
  });

  describe('Activity Icons', () => {
    it('should render icon container for each activity', () => {
      render(<RecentActivity activities={mockActivities} />);

      const iconContainers = document.querySelectorAll('.w-10.h-10.rounded-full');
      expect(iconContainers.length).toBeGreaterThan(0);
    });

    it('should apply correct background color based on activity type', () => {
      render(<RecentActivity activities={mockActivities} />);

      // Check for various color backgrounds
      expect(document.querySelector('.bg-green-100')).toBeInTheDocument();
      expect(document.querySelector('.bg-red-100')).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <RecentActivity activities={mockActivities} className="custom-class" />
      );

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('custom-class');
    });

    it('should apply base card styling', () => {
      const { container } = render(<RecentActivity activities={mockActivities} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('bg-white');
      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('border');
    });

    it('should apply hover effect to activity items', () => {
      const { container } = render(<RecentActivity activities={mockActivities} />);

      const items = container.querySelectorAll('.hover\\:bg-gray-50');
      expect(items.length).toBeGreaterThan(0);
    });
  });

  describe('Header', () => {
    it('should render header with title', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
    });

    it('should have proper header styling', () => {
      const { container } = render(<RecentActivity activities={mockActivities} />);

      const header = container.querySelector('.p-5.border-b');
      expect(header).toBeInTheDocument();
    });
  });

  describe('Activity Details', () => {
    it('should display details when present', () => {
      render(<RecentActivity activities={mockActivities} />);

      expect(screen.getByText('Out of paper')).toBeInTheDocument();
    });

    it('should not display details when not present', () => {
      const activitiesWithoutDetails: Activity[] = [
        {
          id: 'act-1',
          type: 'job_created',
          jobId: 'job-1',
          jobName: 'Test.pdf',
          timestamp: '2025-02-28T10:00:00Z',
        },
      ];

      render(<RecentActivity activities={activitiesWithoutDetails} />);

      // Details should only show for the failed job
      const detailsElements = document.querySelectorAll('.text-xs.text-gray-500');
      const detailsWithText = Array.from(detailsElements).filter(el =>
        el.textContent && el.textContent.trim().length > 0 && !el.textContent.includes('ago')
      );
      expect(detailsWithText.length).toBe(0);
    });
  });

  describe('Different Activity Types', () => {
    it('should handle job_created type', () => {
      const activity: Activity = {
        id: 'act-1',
        type: 'job_created',
        jobId: 'job-1',
        jobName: 'New Job.pdf',
        timestamp: '2025-02-28T10:00:00Z',
      };

      render(<RecentActivity activities={[activity]} />);

      expect(screen.getByText('Job Created')).toBeInTheDocument();
    });

    it('should handle job_cancelled type', () => {
      const activity: Activity = {
        id: 'act-1',
        type: 'job_cancelled',
        jobId: 'job-1',
        jobName: 'Cancelled Job.pdf',
        timestamp: '2025-02-28T10:00:00Z',
      };

      render(<RecentActivity activities={[activity]} />);

      expect(screen.getByText('Job Cancelled')).toBeInTheDocument();
    });

    it('should handle printer_online type', () => {
      const activity: Activity = {
        id: 'act-1',
        type: 'printer_online',
        printerId: 'printer-1',
        printerName: 'Office Printer',
        timestamp: '2025-02-28T10:00:00Z',
      };

      render(<RecentActivity activities={[activity]} />);

      expect(screen.getByText('Printer Online')).toBeInTheDocument();
    });

    it('should handle agent_disconnected type', () => {
      const activity: Activity = {
        id: 'act-1',
        type: 'agent_disconnected',
        agentId: 'agent-1',
        timestamp: '2025-02-28T10:00:00Z',
      };

      render(<RecentActivity activities={[activity]} />);

      expect(screen.getByText('Agent Disconnected')).toBeInTheDocument();
    });

    it('should handle user_invited type', () => {
      const activity: Activity = {
        id: 'act-1',
        type: 'user_invited',
        invitedUserEmail: 'new@example.com',
        timestamp: '2025-02-28T10:00:00Z',
      };

      render(<RecentActivity activities={[activity]} />);

      expect(screen.getByText('User Invited')).toBeInTheDocument();
    });
  });
});
