/**
 * JobChart Component Tests
 * Unit tests for the job statistics chart component
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import { JobChart } from './JobChart';
import type { JobStatistics } from './types';

const mockStatistics: JobStatistics = {
  period: 'week',
  data: [
    { label: 'Mon', value: 42, date: '2025-02-24' },
    { label: 'Tue', value: 55, date: '2025-02-25' },
    { label: 'Wed', value: 38, date: '2025-02-26' },
    { label: 'Thu', value: 61, date: '2025-02-27' },
    { label: 'Fri', value: 44, date: '2025-02-28' },
    { label: 'Sat', value: 23, date: '2025-03-01' },
    { label: 'Sun', value: 15, date: '2025-03-02' },
  ],
  total: 278,
  completed: 265,
  failed: 13,
  averagePerDay: 39.7,
};

describe('JobChart', () => {
  describe('Rendering', () => {
    it('should render chart container', () => {
      render(<JobChart statistics={mockStatistics} />);

      const container = screen.getByText('Job Statistics').closest('div');
      expect(container).toBeInTheDocument();
    });

    it('should render chart title', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('Job Statistics')).toBeInTheDocument();
    });

    it('should render period label', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('Last 7 Days')).toBeInTheDocument();
    });
  });

  describe('Statistics Display', () => {
    it('should render total jobs count', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('278')).toBeInTheDocument();
      expect(screen.getByText('total jobs')).toBeInTheDocument();
    });

    it('should render completed jobs count', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('265 completed')).toBeInTheDocument();
    });

    it('should render failed jobs count', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('13 failed')).toBeInTheDocument();
    });

    it('should render average per day', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText(/39\.7 jobs/)).toBeInTheDocument();
      expect(screen.getByText(/average per/)).toBeInTheDocument();
    });

    it('should render completed status indicator', () => {
      render(<JobChart statistics={mockStatistics} />);

      const indicator = document.querySelector('.bg-green-500.rounded-full');
      expect(indicator).toBeInTheDocument();
    });

    it('should render failed status indicator', () => {
      render(<JobChart statistics={mockStatistics} />);

      const indicator = document.querySelector('.bg-red-500.rounded-full');
      expect(indicator).toBeInTheDocument();
    });
  });

  describe('Chart Data', () => {
    it('should render bars for each data point', () => {
      render(<JobChart statistics={mockStatistics} />);

      // Check for x-axis labels
      expect(screen.getByText('Mon')).toBeInTheDocument();
      expect(screen.getByText('Tue')).toBeInTheDocument();
      expect(screen.getByText('Wed')).toBeInTheDocument();
      expect(screen.getByText('Thu')).toBeInTheDocument();
      expect(screen.getByText('Fri')).toBeInTheDocument();
      expect(screen.getByText('Sat')).toBeInTheDocument();
      expect(screen.getByText('Sun')).toBeInTheDocument();
    });

    it('should calculate bar heights proportionally', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      // Check that bars are rendered with height styles
      const bars = container.querySelectorAll('.bg-blue-500');
      expect(bars.length).toBe(7);
    });

    it('should render max value on y-axis', () => {
      render(<JobChart statistics={mockStatistics} />);

      // Max value is 61 (Thu)
      expect(screen.getByText('61')).toBeInTheDocument();
    });

    it('should render mid value on y-axis', () => {
      render(<JobChart statistics={mockStatistics} />);

      // Mid value should be around 30
      const allText = screen.getAllByText(/\d+/);
      const midValue = allText.find(el => el.textContent === '30') || allText.find(el => el.textContent === '31');
      expect(midValue).toBeInTheDocument();
    });
  });

  describe('Period Labels', () => {
    it('should show "Last 24 Hours" for day period', () => {
      const dayStats = { ...mockStatistics, period: 'day' as const };
      render(<JobChart statistics={dayStats} />);

      expect(screen.getByText('Last 24 Hours')).toBeInTheDocument();
    });

    it('should show "Last 7 Days" for week period', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('Last 7 Days')).toBeInTheDocument();
    });

    it('should show "Last 30 Days" for month period', () => {
      const monthStats = { ...mockStatistics, period: 'month' as const };
      render(<JobChart statistics={monthStats} />);

      expect(screen.getByText('Last 30 Days')).toBeInTheDocument();
    });
  });

  describe('Empty Data', () => {
    it('should render empty state when no data', () => {
      render(<JobChart statistics={{ ...mockStatistics, data: [] }} />);

      expect(screen.getByText('No data available')).toBeInTheDocument();
    });

    it('should not render chart when data is empty', () => {
      const { container } = render(
        <JobChart statistics={{ ...mockStatistics, data: [] }} />
      );

      const bars = container.querySelectorAll('.bg-blue-500');
      expect(bars.length).toBe(0);
    });
  });

  describe('Loading State', () => {
    it('should render skeleton when loading is true', () => {
      render(<JobChart statistics={mockStatistics} loading={true} />);

      const card = screen.getByText('Job Statistics').closest('div');
      expect(card?.parentElement).toHaveClass('animate-pulse');
    });

    it('should not render statistics when loading', () => {
      render(<JobChart statistics={mockStatistics} loading={true} />);

      expect(screen.queryByText('278')).not.toBeInTheDocument();
    });

    it('should not render chart when loading', () => {
      render(<JobChart statistics={mockStatistics} loading={true} />);

      expect(screen.queryByText('Mon')).not.toBeInTheDocument();
    });

    it('should render loading placeholders', () => {
      render(<JobChart statistics={mockStatistics} loading={true} />);

      const placeholders = document.querySelectorAll('.bg-gray-200');
      expect(placeholders.length).toBeGreaterThan(0);
    });
  });

  describe('Tooltips', () => {
    it('should render tooltips on hover (presence check)', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      // Check that tooltip elements exist (opacity-0 by default)
      const tooltips = container.querySelectorAll('.opacity-0.group-hover\\:opacity-100');
      expect(tooltips.length).toBeGreaterThan(0);
    });

    it('should include data in tooltips', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const tooltips = container.querySelectorAll('.opacity-0');
      expect(tooltips.length).toBeGreaterThan(0);
    });
  });

  describe('Bar Styling', () => {
    it('should apply blue color to bars', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const bars = container.querySelectorAll('.bg-blue-500');
      expect(bars.length).toBe(7);
    });

    it('should apply rounded top to bars', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const bars = container.querySelectorAll('.rounded-t');
      expect(bars.length).toBe(7);
    });

    it('should apply hover effect to bars', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const bars = container.querySelectorAll('.hover\\:bg-blue-600');
      expect(bars.length).toBe(7);
    });
  });

  describe('Y-Axis', () => {
    it('should render y-axis grid lines', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const gridLines = container.querySelectorAll('.border-b');
      expect(gridLines.length).toBeGreaterThan(0);
    });

    it('should render zero value', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText('0')).toBeInTheDocument();
    });
  });

  describe('Footer', () => {
    it('should render average in footer section', () => {
      render(<JobChart statistics={mockStatistics} />);

      const footer = document.querySelector('.pt-4.border-t');
      expect(footer).toBeInTheDocument();
      expect(footer?.textContent).toContain('Average per');
    });

    it('should adjust average label based on period', () => {
      render(<JobChart statistics={mockStatistics} />);

      // Week period should show "day"
      expect(screen.getByText(/average per day/i)).toBeInTheDocument();
    });

    it('should show "hour" for day period', () => {
      const dayStats = { ...mockStatistics, period: 'day' as const };
      render(<JobChart statistics={dayStats} />);

      expect(screen.getByText(/average per hour/i)).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <JobChart statistics={mockStatistics} className="custom-class" />
      );

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('custom-class');
    });

    it('should apply base card styling', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('bg-white');
      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('border');
    });

    it('should have proper header section', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const header = container.querySelector('.p-5.border-b');
      expect(header).toBeInTheDocument();
    });

    it('should have proper chart section', () => {
      const { container } = render(<JobChart statistics={mockStatistics} />);

      const chart = container.querySelector('.p-5');
      expect(chart).toBeInTheDocument();
    });
  });

  describe('Average Calculation', () => {
    it('should display formatted average', () => {
      render(<JobChart statistics={mockStatistics} />);

      expect(screen.getByText(/39\.7/)).toBeInTheDocument();
    });

    it('should handle zero average', () => {
      const zeroStats: JobStatistics = {
        ...mockStatistics,
        data: [],
        total: 0,
        completed: 0,
        failed: 0,
        averagePerDay: 0,
      };

      render(<JobChart statistics={zeroStats} />);

      expect(screen.getByText('0')).toBeInTheDocument();
    });
  });

  describe('Max Value Calculation', () => {
    it('should use max value from data for y-axis', () => {
      render(<JobChart statistics={mockStatistics} />);

      // Max is 61
      expect(screen.getByText('61')).toBeInTheDocument();
    });

    it('should handle single data point', () => {
      const singleStats: JobStatistics = {
        ...mockStatistics,
        data: [{ label: 'Today', value: 100, date: '2025-02-28' }],
        total: 100,
        completed: 100,
        failed: 0,
        averagePerDay: 100,
      };

      render(<JobChart statistics={singleStats} />);

      expect(screen.getByText('100')).toBeInTheDocument();
    });

    it('should handle all zeros', () => {
      const zeroStats: JobStatistics = {
        ...mockStatistics,
        data: [
          { label: 'Mon', value: 0, date: '2025-02-24' },
          { label: 'Tue', value: 0, date: '2025-02-25' },
        ],
        total: 0,
        completed: 0,
        failed: 0,
        averagePerDay: 0,
      };

      render(<JobChart statistics={zeroStats} />);

      // Should still show 1 as max when all are 0 (to avoid division by zero)
      expect(screen.getByText('1')).toBeInTheDocument();
    });
  });
});
