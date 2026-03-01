import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { EnvironmentReport } from './EnvironmentReport';

const mockReport = {
  pagesPrinted: 1234,
  co2Grams: 456.7,
  treesSaved: 0.12,
  period: '30d',
};

describe('EnvironmentReport', () => {
  it('should render environmental metrics', () => {
    render(<EnvironmentReport report={mockReport} />);

    expect(screen.getByText('1,234')).toBeInTheDocument(); // Pages Printed (formatted with comma)
    // Use getAllByText for CO2 since it appears in multiple places
    expect(screen.getAllByText(/456\.7g/)).toHaveLength(2);
    expect(screen.getByText('0.1')).toBeInTheDocument(); // Trees Saved (rounded)
  });

  it('should display environmental impact message', () => {
    render(<EnvironmentReport report={mockReport} />);

    expect(screen.getByText(/Environmental Impact/)).toBeInTheDocument();
    expect(screen.getByText(/0\.1 trees/)).toBeInTheDocument(); // Updated to match rounded value
    expect(screen.getByText(/of CO₂ emissions/)).toBeInTheDocument();
  });

  it('should render loading state', () => {
    const { container } = render(<EnvironmentReport report={mockReport} isLoading={true} />);

    // Loading state shows skeleton with animate-pulse class
    const skeletonContainer = container.querySelector('.animate-pulse');
    expect(skeletonContainer).toBeInTheDocument();
    // The title is not shown during loading
    expect(screen.queryByText(/Environmental Impact/)).not.toBeInTheDocument();
  });

  it('should display metric cards with correct labels', () => {
    render(<EnvironmentReport report={mockReport} />);

    expect(screen.getByText('Pages Printed')).toBeInTheDocument();
    expect(screen.getByText(/CO₂ Saved/)).toBeInTheDocument();
    expect(screen.getByText('Trees Saved')).toBeInTheDocument();
  });

  it('should format large numbers correctly', () => {
    const largeReport = {
      pagesPrinted: 12345,
      co2Grams: 12345.67,
      treesSaved: 1.234,
      period: '30d' as const,
    };

    render(<EnvironmentReport report={largeReport} />);

    expect(screen.getByText('12,345')).toBeInTheDocument(); // Pages Printed
    // CO2 value appears in multiple places, use getAllByText
    expect(screen.getAllByText(/12,345\.7g/)).toHaveLength(2);
    expect(screen.getByText('1.2')).toBeInTheDocument(); // Trees Saved (rounded)
  });

  it('should display sustainability message', () => {
    render(<EnvironmentReport report={mockReport} />);

    expect(screen.getByText(/By using cloud printing/)).toBeInTheDocument();
    expect(screen.getByText(/you've saved approximately/)).toBeInTheDocument();
  });
});
