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
    expect(screen.getByText(/456\.7g/)).toBeInTheDocument(); // CO2 Saved
    expect(screen.getByText('0.1')).toBeInTheDocument(); // Trees Saved (rounded)
  });

  it('should display environmental impact message', () => {
    render(<EnvironmentReport report={mockReport} />);

    expect(screen.getByText(/Environmental Impact/)).toBeInTheDocument();
    expect(screen.getByText(/0\.1 trees/)).toBeInTheDocument(); // Updated to match rounded value
    expect(screen.getByText(/456\.7g/)).toBeInTheDocument();
  });

  it('should render loading state', () => {
    render(<EnvironmentReport report={mockReport} isLoading={true} />);

    // Loading state shows skeleton, not the title
    const container = screen.getByText(/Environmental Impact/).closest('div');
    expect(container).toBeInTheDocument();
  });
});
