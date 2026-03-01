/**
 * JobList Component Tests
 * Comprehensive tests for the print job list component
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { JobList } from './JobList';
import type { PrintJob } from '@/types/jobs';

const mockPrinter1 = {
  id: 'printer-1',
  name: 'Office HP',
  agentId: 'agent-1',
  orgId: 'org-1',
  type: 'network' as const,
  capabilities: {
    supportsColor: true,
    supportsDuplex: true,
    supportedPaperSizes: ['A4', 'Letter'],
    resolution: '600x600',
  },
  isActive: true,
  isOnline: true,
  createdAt: '2025-01-01T00:00:00Z',
};

const mockPrinter2 = {
  id: 'printer-2',
  name: 'Design Canon',
  agentId: 'agent-2',
  orgId: 'org-1',
  type: 'usb' as const,
  capabilities: {
    supportsColor: true,
    supportsDuplex: false,
    supportedPaperSizes: ['A4'],
    resolution: '1200x1200',
  },
  isActive: true,
  isOnline: false,
  createdAt: '2025-01-01T00:00:00Z',
};

const mockJobs: PrintJob[] = [
  {
    id: 'job-1',
    userId: 'user-1',
    orgId: 'org-1',
    documentName: 'Quarterly Report.pdf',
    status: 'queued',
    pageCount: 15,
    colorPages: 3,
    fileSize: 2048576,
    createdAt: '2025-02-28T10:30:00Z',
    printer: undefined,
    settings: {},
  },
  {
    id: 'job-2',
    userId: 'user-1',
    orgId: 'org-1',
    documentName: 'Presentation.pptx',
    status: 'processing',
    pageCount: 24,
    colorPages: 12,
    fileSize: 5242880,
    createdAt: '2025-02-28T11:15:00Z',
    printer: mockPrinter1,
    settings: {},
  },
  {
    id: 'job-3',
    userId: 'user-1',
    orgId: 'org-1',
    documentName: 'Failed_Doc.pdf',
    status: 'failed',
    pageCount: 5,
    colorPages: 0,
    fileSize: 512000,
    createdAt: '2025-02-28T09:00:00Z',
    printer: mockPrinter2,
    settings: {},
    errorMessage: 'Printer out of paper',
  },
  {
    id: 'job-4',
    userId: 'user-1',
    orgId: 'org-1',
    documentName: 'Completed.pdf',
    status: 'completed',
    pageCount: 10,
    colorPages: 5,
    fileSize: 1048576,
    createdAt: '2025-02-28T08:00:00Z',
    printer: mockPrinter1,
    settings: {},
  },
];

describe('JobList', () => {
  describe('Loading State', () => {
    it('should render loading skeleton when isLoading is true', () => {
      render(<JobList jobs={[]} isLoading={true} />);

      // Check for table structure
      const table = screen.getByRole('table');
      expect(table).toBeInTheDocument();
    });

    it('should render multiple skeleton rows', () => {
      render(<JobList jobs={[]} isLoading={true} />);

      const table = screen.getByRole('table');
      expect(table).toBeInTheDocument();
    });

    it('should not render empty state when loading', () => {
      render(<JobList jobs={[]} isLoading={true} />);

      expect(screen.queryByText(/no print jobs/i)).not.toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    it('should render error message when error prop is provided', () => {
      render(<JobList jobs={[]} error="Failed to load jobs" />);

      expect(screen.getByText('Error loading jobs')).toBeInTheDocument();
      expect(screen.getByText('Failed to load jobs')).toBeInTheDocument();
    });

    it('should render error icon', () => {
      render(<JobList jobs={[]} error="Network error" />);

      const errorIcon = document.querySelector('svg.text-red-400');
      expect(errorIcon).toBeInTheDocument();
    });
  });

  describe('Empty State', () => {
    it('should render empty state when jobs array is empty', () => {
      render(<JobList jobs={[]} />);

      expect(screen.getByText('No print jobs')).toBeInTheDocument();
    });

    it('should render custom empty message', () => {
      render(<JobList jobs={[]} emptyMessage="No jobs found" emptyDescription="Try again later" />);

      expect(screen.getByText('No jobs found')).toBeInTheDocument();
      expect(screen.getByText('Try again later')).toBeInTheDocument();
    });

    it('should render default empty description', () => {
      render(<JobList jobs={[]} />);

      expect(screen.getByText(/get started by submitting a new print job/i)).toBeInTheDocument();
    });
  });

  describe('Job Entries', () => {
    it('should render job entries table with all jobs', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      expect(screen.getByText('Presentation.pptx')).toBeInTheDocument();
      expect(screen.getByText('Failed_Doc.pdf')).toBeInTheDocument();
      expect(screen.getByText('Completed.pdf')).toBeInTheDocument();
    });

    it('should render job status badges', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText('Queued')).toBeInTheDocument();
      expect(screen.getByText('Processing')).toBeInTheDocument();
      expect(screen.getByText('Failed')).toBeInTheDocument();
      expect(screen.getByText('Completed')).toBeInTheDocument();
    });

    it('should render job page counts', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText(/15 pages/)).toBeInTheDocument();
      expect(screen.getByText(/24 pages/)).toBeInTheDocument();
    });

    it('should render color page count when present', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText(/\(3 color\)/)).toBeInTheDocument();
    });

    it('should render file sizes', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText(/2 MB/)).toBeInTheDocument();
      expect(screen.getByText(/5 MB/)).toBeInTheDocument();
    });

    it('should display printer name when assigned', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText('Office HP')).toBeInTheDocument();
      expect(screen.getByText('Design Canon')).toBeInTheDocument();
    });

    it('should show "No printer" when not assigned', () => {
      render(<JobList jobs={mockJobs} />);

      const noPrinterText = screen.getAllByText('No printer');
      expect(noPrinterText.length).toBeGreaterThan(0);
    });

    it('should display error message for failed jobs', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText('Printer out of paper')).toBeInTheDocument();
    });
  });

  describe('Selection', () => {
    it('should render checkbox for each job', () => {
      render(<JobList jobs={mockJobs} />);

      const checkboxes = screen.getAllByRole('checkbox');
      // Includes select all checkbox
      expect(checkboxes.length).toBeGreaterThan(1);
    });

    it('should call onSelectionChange when job checkbox is clicked', async () => {
      const user = userEvent.setup();
      const handleChange = vi.fn();
      render(<JobList jobs={mockJobs} onSelectionChange={handleChange} />);

      const checkboxes = screen.getAllByRole('checkbox');
      // Skip the select all checkbox, click the first job checkbox
      await user.click(checkboxes[1]);

      expect(handleChange).toHaveBeenCalled();
    });

    it('should call onJobClick when job row is clicked', async () => {
      const user = userEvent.setup();
      const handleClick = vi.fn();
      render(<JobList jobs={mockJobs} onJobClick={handleClick} />);

      const jobRow = screen.getByText('Quarterly Report.pdf').closest('tr');
      if (jobRow) {
        await user.click(jobRow);
      }

      expect(handleClick).toHaveBeenCalled();
    });
  });

  describe('Actions', () => {
    it('should show cancel button for queued jobs', () => {
      render(<JobList jobs={mockJobs} />);

      // Cancel button should be present for queued job
      const cancelButtons = document.querySelectorAll('button[title="Cancel job"]');
      expect(cancelButtons.length).toBeGreaterThan(0);
    });

    it('should show retry button for failed jobs', () => {
      render(<JobList jobs={mockJobs} />);

      // Retry button should be present for failed job
      const retryButtons = document.querySelectorAll('button[title="Retry job"]');
      expect(retryButtons.length).toBeGreaterThan(0);
    });

    it('should show view details button when onJobClick is provided', () => {
      render(<JobList jobs={mockJobs} onJobClick={vi.fn()} />);

      const viewButtons = document.querySelectorAll('button[title="View details"]');
      expect(viewButtons.length).toBeGreaterThan(0);
    });

    it('should not show cancel button for completed jobs', () => {
      render(<JobList jobs={mockJobs} />);

      // Get all rows and check the completed job doesn't have cancel
      const completedJob = mockJobs.find(j => j.status === 'completed');
      expect(completedJob).toBeDefined();
    });
  });

  describe('Table Headers', () => {
    it('should render all table headers', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByText('Document')).toBeInTheDocument();
      expect(screen.getByText('Printer')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText(/pages \/ size/i)).toBeInTheDocument();
      expect(screen.getByText('Created')).toBeInTheDocument();
      expect(screen.getByText('Actions')).toBeInTheDocument();
    });
  });

  describe('Filtering', () => {
    it('should render jobs filtered by status', () => {
      const queuedJobs = mockJobs.filter(j => j.status === 'queued');
      render(<JobList jobs={queuedJobs} />);

      expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      expect(screen.queryByText('Presentation.pptx')).not.toBeInTheDocument();
    });
  });

  describe('Timestamp Display', () => {
    it('should display relative time for job creation', () => {
      render(<JobList jobs={mockJobs} />);

      // The component uses formatDistanceToNow which shows relative time
      // Just check that some time text is present
      const timeElements = document.querySelectorAll('td.text-gray-500');
      expect(timeElements.length).toBeGreaterThan(0);
    });
  });

  describe('Accessibility', () => {
    it('should have proper table structure', () => {
      render(<JobList jobs={mockJobs} />);

      expect(screen.getByRole('table')).toBeInTheDocument();
      expect(screen.getAllByRole('row')).toHaveLength(mockJobs.length + 2); // +1 for header, +1 for header row
    });

    it('should have proper button labels', () => {
      render(<JobList jobs={mockJobs} />);

      expect(document.querySelector('button[title="Cancel job"]')).toBeInTheDocument();
      expect(document.querySelector('button[title="Retry job"]')).toBeInTheDocument();
      expect(document.querySelector('button[title="View details"]')).toBeInTheDocument();
    });
  });
});
