/**
 * ComplianceStatusBadge Unit Tests
 */

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ComplianceStatusBadge } from './ComplianceStatusBadge';

describe('ComplianceStatusBadge', () => {
  it('should render compliant status correctly', () => {
    render(<ComplianceStatusBadge status="compliant" />);
    expect(screen.getByText('Compliant')).toBeInTheDocument();
  });

  it('should render non_compliant status correctly', () => {
    render(<ComplianceStatusBadge status="non_compliant" />);
    expect(screen.getByText('Non-Compliant')).toBeInTheDocument();
  });

  it('should render in_progress status correctly', () => {
    render(<ComplianceStatusBadge status="in_progress" />);
    expect(screen.getByText('In Progress')).toBeInTheDocument();
  });

  it('should render pending status correctly', () => {
    render(<ComplianceStatusBadge status="pending" />);
    expect(screen.getByText('Pending')).toBeInTheDocument();
  });

  it('should render not_applicable status correctly', () => {
    render(<ComplianceStatusBadge status="not_applicable" />);
    expect(screen.getByText('N/A')).toBeInTheDocument();
  });

  it('should render unknown status correctly', () => {
    render(<ComplianceStatusBadge status="unknown" />);
    expect(screen.getByText('Unknown')).toBeInTheDocument();
  });

  it('should hide label when showLabel is false', () => {
    render(<ComplianceStatusBadge status="compliant" showLabel={false} />);
    expect(screen.queryByText('Compliant')).not.toBeInTheDocument();
  });

  it('should apply correct test id', () => {
    const { container } = render(
      <ComplianceStatusBadge status="compliant" data-testid="test-badge" />
    );
    expect(container.firstChild).toHaveAttribute('data-testid', 'test-badge');
  });

  it('should have correct data-testid for status', () => {
    render(<ComplianceStatusBadge status="compliant" />);
    const badge = screen.getByText('Compliant').closest('span');
    expect(badge).toHaveAttribute('data-testid', 'compliance-status-compliant');
  });
});
