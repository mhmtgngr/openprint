/**
 * RiskAssessment Unit Tests
 */

import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { RiskAssessment } from './RiskAssessment';

describe('RiskAssessment', () => {
  it('should render empty state initially', () => {
    render(<RiskAssessment />);

    expect(
      screen.getByText(/Click "Run Assessment" to evaluate security risks/)
    ).toBeInTheDocument();
  });

  it('should display risk score when provided', () => {
    render(<RiskAssessment riskScore={25} level="low" />);

    expect(screen.getByText('25')).toBeInTheDocument();
    expect(screen.getByText('Low Risk')).toBeInTheDocument();
    expect(screen.getByText('Good security posture')).toBeInTheDocument();
  });

  it('should display medium risk correctly', () => {
    render(<RiskAssessment riskScore={45} level="medium" />);

    expect(screen.getByText('45')).toBeInTheDocument();
    expect(screen.getByText('Medium Risk')).toBeInTheDocument();
    expect(
      screen.getByText('Some improvements needed')
    ).toBeInTheDocument();
  });

  it('should display high risk correctly', () => {
    render(<RiskAssessment riskScore={75} level="high" />);

    expect(screen.getByText('75')).toBeInTheDocument();
    expect(screen.getByText('High Risk')).toBeInTheDocument();
    expect(screen.getByText('Immediate action required')).toBeInTheDocument();
  });

  it('should display mitigation suggestions', () => {
    const mitigations = [
      'Enable two-factor authentication',
      'Implement IP whitelist',
      'Review audit logs weekly',
    ];

    render(
      <RiskAssessment riskScore={25} level="low" mitigations={mitigations} />
    );

    expect(screen.getByText('Enable two-factor authentication')).toBeInTheDocument();
    expect(screen.getByText('Implement IP whitelist')).toBeInTheDocument();
    expect(screen.getByText('Review audit logs weekly')).toBeInTheDocument();
  });

  it('should display recommended actions header when mitigations exist', () => {
    render(
      <RiskAssessment
        riskScore={25}
        level="low"
        mitigations={['Enable 2FA']}
      />
    );

    expect(screen.getByText('Recommended Actions')).toBeInTheDocument();
  });

  it('should call onRun when button is clicked', async () => {
    const onRun = vi.fn();
    const user = userEvent.setup();

    render(<RiskAssessment onRun={onRun} />);

    const runButton = screen.getByRole('button', { name: /Run Assessment/i });
    await user.click(runButton);

    expect(onRun).toHaveBeenCalledTimes(1);
  });

  it('should show loading state when running', () => {
    render(<RiskAssessment onRun={vi.fn()} isRunning={true} />);

    expect(screen.getByText(/Running\.\.\./i)).toBeInTheDocument();
  });

  it('should have correct test id', () => {
    render(<RiskAssessment />);

    expect(screen.getByTestId('risk-assessment-section')).toBeInTheDocument();
  });

  it('should display risk score bar', () => {
    render(<RiskAssessment riskScore={50} level="medium" />);

    expect(screen.getByTestId('risk-score')).toBeInTheDocument();
    expect(screen.getByTestId('risk-score-bar')).toBeInTheDocument();
  });

  it('should display risk mitigations section with items', () => {
    render(
      <RiskAssessment
        riskScore={30}
        level="medium"
        mitigations={['Fix vulnerability', 'Update system']}
      />
    );

    expect(screen.getByTestId('risk-mitigations')).toBeInTheDocument();
    expect(screen.getByText('Fix vulnerability')).toBeInTheDocument();
    expect(screen.getByText('Update system')).toBeInTheDocument();
  });
});
