import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { JobStatusBadge } from './JobStatusBadge';

describe('JobStatusBadge', () => {
  it('should render queued status', () => {
    render(<JobStatusBadge status="queued" />);
    expect(screen.getByText('Queued')).toBeInTheDocument();
  });

  it('should render processing status', () => {
    render(<JobStatusBadge status="processing" />);
    expect(screen.getByText('Processing')).toBeInTheDocument();
  });

  it('should render completed status', () => {
    render(<JobStatusBadge status="completed" />);
    expect(screen.getByText('Completed')).toBeInTheDocument();
  });

  it('should render failed status', () => {
    render(<JobStatusBadge status="failed" />);
    expect(screen.getByText('Failed')).toBeInTheDocument();
  });

  it('should render cancelled status', () => {
    render(<JobStatusBadge status="cancelled" />);
    expect(screen.getByText('Cancelled')).toBeInTheDocument();
  });

  it('should apply custom className', () => {
    const { container } = render(<JobStatusBadge status="completed" className="custom-class" />);
    expect(container.firstChild).toHaveClass('custom-class');
  });
});
