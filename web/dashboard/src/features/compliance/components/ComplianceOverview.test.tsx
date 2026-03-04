/**
 * ComplianceOverview Unit Tests
 */

import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ComplianceOverview } from './index';

describe('ComplianceOverview', () => {
  const mockOverview = {
    fedramp: { status: 'compliant' as const, last_audit: '2024-01-15' },
    hipaa: { status: 'compliant' as const, last_audit: '2024-01-15' },
    gdpr: { status: 'compliant' as const, last_audit: '2024-01-15' },
    soc2: { status: 'in_progress' as const, last_audit: '2024-01-15' },
    total_logs: 1523,
    compliant_standards: 3,
    pending_actions: 5,
  };

  it('should render compliance frameworks', () => {
    render(<ComplianceOverview data={mockOverview} />);

    expect(screen.getByText('FedRAMP')).toBeInTheDocument();
    expect(screen.getByText('HIPAA')).toBeInTheDocument();
    expect(screen.getByText('GDPR')).toBeInTheDocument();
    expect(screen.getByText('SOC 2')).toBeInTheDocument();
  });

  it('should render statistics cards', () => {
    render(<ComplianceOverview data={mockOverview} />);

    expect(screen.getByText('Total Audit Logs')).toBeInTheDocument();
    expect(screen.getByText('1523')).toBeInTheDocument();
    expect(screen.getByText('Compliant Standards')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('Pending Actions')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('should render framework status badges', () => {
    render(<ComplianceOverview data={mockOverview} />);

    expect(screen.getByText('Compliant')).toBeInTheDocument();
    expect(screen.getByText('In Progress')).toBeInTheDocument();
  });

  it('should have correct test ids for framework cards', () => {
    render(<ComplianceOverview data={mockOverview} />);

    expect(screen.getByTestId('framework-card-fedramp')).toBeInTheDocument();
    expect(screen.getByTestId('framework-card-hipaa')).toBeInTheDocument();
    expect(screen.getByTestId('framework-card-gdpr')).toBeInTheDocument();
    expect(screen.getByTestId('framework-card-soc2')).toBeInTheDocument();
  });

  it('should have correct test ids for status badges', () => {
    render(<ComplianceOverview data={mockOverview} />);

    expect(screen.getByTestId('fedramp-status')).toBeInTheDocument();
    expect(screen.getByTestId('hipaa-status')).toBeInTheDocument();
    expect(screen.getByTestId('gdpr-status')).toBeInTheDocument();
    expect(screen.getByTestId('soc2-status')).toBeInTheDocument();
  });

  it('should render loading state', () => {
    render(<ComplianceOverview isLoading={true} />);

    expect(screen.getByTestId('compliance-overview-loading')).toBeInTheDocument();
  });

  it('should render error state', () => {
    render(<ComplianceOverview error="Failed to load" />);

    expect(screen.getByTestId('compliance-overview-error')).toBeInTheDocument();
    expect(screen.getByText('Failed to load')).toBeInTheDocument();
  });

  it('should render last audit dates', () => {
    render(<ComplianceOverview data={mockOverview} />);

    expect(screen.getByText(/Last audit:/)).toBeInTheDocument();
  });
});
