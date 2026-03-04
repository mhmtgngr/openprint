/**
 * ComplianceChecklist Unit Tests
 */

import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { ComplianceChecklist } from './ComplianceChecklist';

describe('ComplianceChecklist', () => {
  const mockChecklist = [
    { name: 'Access Control', status: 'pass' as const },
    { name: 'Audit Logging', status: 'pass' as const },
    { name: 'Data Encryption', status: 'fail' as const },
    { name: 'Incident Response', status: 'warning' as const },
    { name: 'Security Training', status: 'pending' as const },
  ];

  it('should render checklist items', () => {
    render(<ComplianceChecklist checklist={mockChecklist} />);

    expect(screen.getByText('Access Control')).toBeInTheDocument();
    expect(screen.getByText('Audit Logging')).toBeInTheDocument();
    expect(screen.getByText('Data Encryption')).toBeInTheDocument();
    expect(screen.getByText('Incident Response')).toBeInTheDocument();
    expect(screen.getByText('Security Training')).toBeInTheDocument();
  });

  it('should display status counts', () => {
    render(<ComplianceChecklist checklist={mockChecklist} />);

    expect(screen.getByText('2 passed')).toBeInTheDocument();
    expect(screen.getByText('1 failed')).toBeInTheDocument();
    expect(screen.getByText('1 warnings')).toBeInTheDocument();
    expect(screen.getByText('1 pending')).toBeInTheDocument();
  });

  it('should show empty state when no checklist items', () => {
    render(<ComplianceChecklist checklist={[]} />);

    expect(
      screen.getByText(/Click "Run Checklist" to verify compliance status/)
    ).toBeInTheDocument();
  });

  it('should call onRun when button is clicked', async () => {
    const onRun = vi.fn();
    const user = userEvent.setup();

    render(<ComplianceChecklist checklist={[]} onRun={onRun} />);

    const runButton = screen.getByRole('button', { name: /Run Checklist/i });
    await user.click(runButton);

    expect(onRun).toHaveBeenCalledTimes(1);
  });

  it('should show loading state when running', () => {
    render(<ComplianceChecklist checklist={[]} onRun={vi.fn()} isRunning={true} />);

    expect(screen.getByText(/Running\.\.\./i)).toBeInTheDocument();
  });

  it('should display correct status for each item', () => {
    render(<ComplianceChecklist checklist={mockChecklist} />);

    const passBadge = screen.getByText('Pass');
    const failBadge = screen.getByText('Fail');
    const warningBadge = screen.getByText('Warning');
    const pendingBadge = screen.getByText('Pending');

    expect(passBadge).toBeInTheDocument();
    expect(failBadge).toBeInTheDocument();
    expect(warningBadge).toBeInTheDocument();
    expect(pendingBadge).toBeInTheDocument();
  });

  it('should have correct test ids', () => {
    render(<ComplianceChecklist checklist={mockChecklist} />);

    expect(screen.getByTestId('compliance-checklist')).toBeInTheDocument();

    const items = screen.getAllByTestId('checklist-item');
    expect(items).toHaveLength(5);
  });

  it('should display description for checklist items', () => {
    const checklistWithDescriptions = [
      {
        name: 'Access Control',
        status: 'pass' as const,
        description: 'Verify user access controls are in place',
      },
    ];

    render(<ComplianceChecklist checklist={checklistWithDescriptions} />);

    expect(
      screen.getByText('Verify user access controls are in place')
    ).toBeInTheDocument();
  });
});
