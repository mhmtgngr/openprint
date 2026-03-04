/**
 * StatCard Unit Tests
 */

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StatCard } from './StatCard';

describe('StatCard', () => {
  it('should render title and value', () => {
    render(<StatCard title="Test Metric" value={42} />);

    expect(screen.getByText('Test Metric')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('should render value', () => {
    render(<StatCard title="Pages" value={100} />);

    expect(screen.getByText('100')).toBeInTheDocument();
  });

  it('should render trend indicator', () => {
    render(
      <StatCard
        title="Test Metric"
        value={100}
        trend={{ value: 10, label: 'vs last month', isPositive: true }}
      />
    );

    expect(screen.getByText('10%')).toBeInTheDocument();
    expect(screen.getByText('vs last month')).toBeInTheDocument();
  });

  it('should render positive trend with green color', () => {
    render(
      <StatCard
        title="Test Metric"
        value={100}
        trend={{ value: 10, label: 'vs last month', isPositive: true }}
      />
    );

    const trendElement = screen.getByText('10%').closest('span');
    expect(trendElement).toHaveClass('text-green-600');
  });

  it('should render negative trend with red color', () => {
    render(
      <StatCard
        title="Test Metric"
        value={100}
        trend={{ value: 5, label: 'vs last month', isPositive: false }}
      />
    );

    const trendElement = screen.getByText('5%').closest('span');
    expect(trendElement).toHaveClass('text-red-600');
  });

  it('should render loading state', () => {
    render(<StatCard title="Test Metric" value={0} loading={true} />);

    const card = screen.getByTestId('stat-card');
    expect(card).toHaveClass('animate-pulse');
  });

  it('should render icon', () => {
    const TestIcon = () => (
      <svg data-testid="test-icon">
        <circle cx="10" cy="10" r="10" />
      </svg>
    );

    render(<StatCard title="Test Metric" value={42} icon={TestIcon} />);

    expect(screen.getByTestId('test-icon')).toBeInTheDocument();
  });

  it('should apply correct color class for icon', () => {
    const TestIcon = () => <svg data-testid="test-icon" />;

    const { rerender } = render(
      <StatCard title="Test" value={1} icon={TestIcon} color="blue" />
    );
    expect(screen.getByTestId('test-icon').closest('div')).toHaveClass(
      'bg-blue-100'
    );

    rerender(<StatCard title="Test" value={1} icon={TestIcon} color="green" />);
    expect(screen.getByTestId('test-icon').closest('div')).toHaveClass(
      'bg-green-100'
    );
  });

  it('should apply custom className', () => {
    const { container } = render(
      <StatCard title="Test" value={1} className="custom-class" />
    );

    expect(container.firstChild).toHaveClass('custom-class');
  });
});
