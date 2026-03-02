/**
 * StatCard Component Tests
 * Unit tests for the dashboard stat card component
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import { StatCard } from './StatCard';

// Test icon component
const TestIcon = () => (
  <svg data-testid="test-icon">
    <circle cx="10" cy="10" r="10" />
  </svg>
);

describe('StatCard', () => {
  describe('Rendering', () => {
    it('should render title and value', () => {
      render(<StatCard title="Total Jobs" value={42} />);

      expect(screen.getByText('Total Jobs')).toBeInTheDocument();
      expect(screen.getByText('42')).toBeInTheDocument();
    });

    it('should render value as number when passed as number', () => {
      render(<StatCard title="Test" value={1234} />);

      expect(screen.getByText('1,234')).toBeInTheDocument();
    });

    it('should render value as string when passed as string', () => {
      render(<StatCard title="Test" value="custom" />);

      expect(screen.getByText('custom')).toBeInTheDocument();
    });

    it('should render unit when provided', () => {
      render(<StatCard title="Test" value={42} unit="jobs" />);

      expect(screen.getByText('jobs')).toBeInTheDocument();
    });

    it('should render icon when provided', () => {
      render(<StatCard title="Test" value={42} icon={TestIcon} />);

      expect(screen.getByTestId('test-icon')).toBeInTheDocument();
    });

    it('should apply color variant classes', () => {
      const { container } = render(<StatCard title="Test" value={42} color="green" icon={TestIcon} />);

      // Check for green color classes on the icon container
      expect(container.querySelector('.bg-green-100')).toBeInTheDocument();
    });
  });

  describe('Trend Indicator', () => {
    it('should render positive trend with upward arrow', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 12, label: 'from last week', isPositive: true }}
        />
      );

      expect(screen.getByText('12%')).toBeInTheDocument();
      expect(screen.getByText('from last week')).toBeInTheDocument();
    });

    it('should apply positive trend styling', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 8, label: 'vs yesterday', isPositive: true }}
        />
      );

      const trendText = screen.getByText('8%').closest('span');
      expect(trendText).toHaveClass(/text-green-600/);
    });

    it('should render negative trend with downward arrow', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 5, label: 'from last week', isPositive: false }}
        />
      );

      expect(screen.getByText('5%')).toBeInTheDocument();
      expect(screen.getByText('from last week')).toBeInTheDocument();
    });

    it('should apply negative trend styling', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 3, label: 'decline', isPositive: false }}
        />
      );

      const trendText = screen.getByText('3%').closest('span');
      expect(trendText).toHaveClass(/text-red-600/);
    });

    it('should not render trend when not provided', () => {
      render(<StatCard title="Test" value={42} />);

      expect(screen.queryByText(/%/)).not.toBeInTheDocument();
    });

    it('should display absolute value for positive trends', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 15, label: 'increase', isPositive: true }}
        />
      );

      expect(screen.getByText('15%')).toBeInTheDocument();
    });

    it('should display absolute value for negative trends', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 20, label: 'decrease', isPositive: false }}
        />
      );

      expect(screen.getByText('20%')).toBeInTheDocument();
    });
  });

  describe('Loading State', () => {
    it('should render skeleton when loading is true', () => {
      render(<StatCard title="Test" value={42} loading={true} />);

      // The loading card has animate-pulse class
      const card = document.querySelector('.animate-pulse');
      expect(card).toBeInTheDocument();
    });

    it('should not render value when loading', () => {
      render(<StatCard title="Test" value={42} loading={true} />);

      expect(screen.queryByText('42')).not.toBeInTheDocument();
    });

    it('should not render trend when loading', () => {
      render(
        <StatCard
          title="Test"
          value={42}
          trend={{ value: 10, label: 'test', isPositive: true }}
          loading={true}
        />
      );

      expect(screen.queryByText('10%')).not.toBeInTheDocument();
    });

    it('should render loading placeholders', () => {
      render(<StatCard title="Test" value={42} loading={true} />);

      // Check for placeholder elements
      const placeholders = document.querySelectorAll('.bg-gray-200');
      expect(placeholders.length).toBeGreaterThan(0);
    });
  });

  describe('Color Variants', () => {
    it('should apply blue color classes', () => {
      const { container } = render(<StatCard title="Test" value={42} color="blue" icon={TestIcon} />);

      expect(container.querySelector('.bg-blue-100')).toBeInTheDocument();
    });

    it('should apply green color classes', () => {
      const { container } = render(<StatCard title="Test" value={42} color="green" icon={TestIcon} />);

      expect(container.querySelector('.bg-green-100')).toBeInTheDocument();
    });

    it('should apply purple color classes', () => {
      const { container } = render(<StatCard title="Test" value={42} color="purple" icon={TestIcon} />);

      expect(container.querySelector('.bg-purple-100')).toBeInTheDocument();
    });

    it('should apply orange color classes', () => {
      const { container } = render(<StatCard title="Test" value={42} color="orange" icon={TestIcon} />);

      expect(container.querySelector('.bg-orange-100')).toBeInTheDocument();
    });

    it('should apply red color classes', () => {
      const { container } = render(<StatCard title="Test" value={42} color="red" icon={TestIcon} />);

      expect(container.querySelector('.bg-red-100')).toBeInTheDocument();
    });

    it('should use blue as default color', () => {
      const { container } = render(<StatCard title="Test" value={42} icon={TestIcon} />);

      expect(container.querySelector('.bg-blue-100')).toBeInTheDocument();
    });
  });

  describe('Icon Rendering', () => {
    it('should render icon with correct background', () => {
      const { container } = render(
        <StatCard title="Test" value={42} color="green" icon={TestIcon} />
      );

      const iconContainer = container.querySelector('.bg-green-100');
      expect(iconContainer).toBeInTheDocument();
      expect(iconContainer?.querySelector('svg')).toBeInTheDocument();
    });

    it('should not render icon when not provided', () => {
      const { container } = render(<StatCard title="Test" value={42} />);

      expect(container.querySelector('svg')).not.toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <StatCard title="Test" value={42} className="custom-class" />
      );

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('custom-class');
    });

    it('should apply base card styling', () => {
      const { container } = render(<StatCard title="Test" value={42} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('bg-white');
      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('border');
    });

    it('should apply hover effect', () => {
      const { container } = render(<StatCard title="Test" value={42} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('hover:shadow-md');
    });
  });

  describe('Value Formatting', () => {
    it('should format large numbers with locale', () => {
      render(<StatCard title="Test" value={1000000} />);

      expect(screen.getByText('1,000,000')).toBeInTheDocument();
    });

    it('should format decimal numbers', () => {
      render(<StatCard title="Test" value={1234.56} />);

      // toLocaleString rounds to integer by default
      expect(screen.getByText('1,235')).toBeInTheDocument();
    });

    it('should handle zero value', () => {
      render(<StatCard title="Test" value={0} />);

      expect(screen.getByText('0')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper heading structure', () => {
      render(<StatCard title="Total Jobs" value={42} />);

      const title = screen.getByText('Total Jobs');
      expect(title.tagName).toBe('P');
    });

    it('should have proper value display', () => {
      render(<StatCard title="Test" value={42} />);

      const value = screen.getByText('42');
      expect(value.tagName).toBe('H3');
    });
  });
});
