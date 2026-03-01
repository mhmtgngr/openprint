/**
 * LoginForm Component Tests
 * Comprehensive tests for the login form component
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { LoginForm } from './LoginForm';

describe('LoginForm', () => {
  const mockOnSubmit = vi.fn();

  beforeEach(() => {
    mockOnSubmit.mockClear();
  });

  describe('Rendering', () => {
    it('should render email and password input fields', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/email address/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    });

    it('should render submit button with correct text', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
    });

    it('should render email input with correct attributes', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      expect(emailInput).toHaveAttribute('type', 'email');
      expect(emailInput).toHaveAttribute('required');
      expect(emailInput).toHaveAttribute('autoComplete', 'email');
    });

    it('should render password input with correct attributes', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const passwordInput = screen.getByLabelText(/password/i);
      expect(passwordInput).toHaveAttribute('type', 'password');
      expect(passwordInput).toHaveAttribute('required');
      expect(passwordInput).toHaveAttribute('minLength', '8');
      expect(passwordInput).toHaveAttribute('autoComplete', 'current-password');
    });

    it('should render placeholder text in inputs', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/password/i);

      expect(emailInput).toHaveAttribute('placeholder', 'you@example.com');
      expect(passwordInput).toHaveAttribute('placeholder', '••••••••');
    });
  });

  describe('Form Validation', () => {
    it('should prevent submission with empty fields', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const submitButton = screen.getByRole('button', { name: /sign in/i });
      await user.click(submitButton);

      // HTML5 validation should prevent submission
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });

    it('should enforce password minimum length of 8 characters', async () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const passwordInput = screen.getByLabelText(/password/i);
      expect(passwordInput).toHaveAttribute('minLength', '8');
    });
  });

  describe('Form Submission', () => {
    it('should call onSubmit with email and password when valid form is submitted', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(emailInput, 'test@example.com');
      await user.type(passwordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledTimes(1);
        expect(mockOnSubmit).toHaveBeenCalledWith('test@example.com', 'password123');
      });
    });

    it('should allow submission with exactly 8 character password', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(emailInput, 'test@example.com');
      await user.type(passwordInput, '12345678');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith('test@example.com', '12345678');
      });
    });

    it('should allow submission with longer password', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/password/i);
      const submitButton = screen.getByRole('button', { name: /sign in/i });

      await user.type(emailInput, 'test@example.com');
      await user.type(passwordInput, 'very-secure-password-123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith('test@example.com', 'very-secure-password-123');
      });
    });
  });

  describe('Error Handling', () => {
    it('should display error message when error prop is provided', () => {
      render(<LoginForm onSubmit={mockOnSubmit} error="Invalid credentials" />);

      expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
    });

    it('should render error in alert-styled container', () => {
      render(<LoginForm onSubmit={mockOnSubmit} error="Invalid credentials" />);

      const errorElement = screen.getByText('Invalid credentials').closest('div');
      expect(errorElement).toHaveClass(/bg-red-100/);
    });

    it('should not display error when error prop is null', () => {
      render(<LoginForm onSubmit={mockOnSubmit} error={null} />);

      expect(screen.queryByText(/invalid/i)).not.toBeInTheDocument();
    });

    it('should not display error when error prop is not provided', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      expect(screen.queryByText(/invalid/i)).not.toBeInTheDocument();
    });
  });

  describe('Loading State', () => {
    it('should show loading text when isLoading is true', () => {
      render(<LoginForm onSubmit={mockOnSubmit} isLoading={true} />);

      expect(screen.getByRole('button', { name: /please wait/i/i })).toBeInTheDocument();
    });

    it('should show normal button text when isLoading is false', () => {
      render(<LoginForm onSubmit={mockOnSubmit} isLoading={false} />);

      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
    });

    it('should show normal button text when isLoading is not provided', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
    });

    it('should disable submit button when isLoading is true', () => {
      render(<LoginForm onSubmit={mockOnSubmit} isLoading={true} />);

      const submitButton = screen.getByRole('button', { name: /please wait/i/i });
      expect(submitButton).toBeDisabled();
    });

    it('should not disable submit button when isLoading is false', () => {
      render(<LoginForm onSubmit={mockOnSubmit} isLoading={false} />);

      const submitButton = screen.getByRole('button', { name: /sign in/i });
      expect(submitButton).not.toBeDisabled();
    });

    it('should apply disabled styling when isLoading is true', () => {
      render(<LoginForm onSubmit={mockOnSubmit} isLoading={true} />);

      const submitButton = screen.getByRole('button', { name: /please wait/i/i });
      expect(submitButton).toHaveClass(/disabled:opacity-50/);
    });
  });

  describe('User Interaction', () => {
    it('should update email input value when user types', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      await user.type(emailInput, 'user@example.com');

      expect(emailInput).toHaveValue('user@example.com');
    });

    it('should update password input value when user types', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const passwordInput = screen.getByLabelText(/password/i);
      await user.type(passwordInput, 'secret123');

      expect(passwordInput).toHaveValue('secret123');
    });

    it('should handle form submission via Enter key', async () => {
      const user = userEvent.setup();
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/password/i);

      await user.type(emailInput, 'test@example.com');
      await user.type(passwordInput, 'password123{Enter}');

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith('test@example.com', 'password123');
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper label associations for inputs', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/password/i);

      expect(emailInput).toHaveAttribute('id', 'login-email');
      expect(passwordInput).toHaveAttribute('id', 'login-password');
    });

    it('should have proper form structure', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const form = screen.getByLabelText(/email address/i).closest('form');
      expect(form).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply gradient styling to submit button', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const submitButton = screen.getByRole('button', { name: /sign in/i });
      expect(submitButton).toHaveClass(/bg-gradient-to-r/);
    });

    it('should apply focus ring styling to inputs', () => {
      render(<LoginForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      expect(emailInput).toHaveClass(/focus:ring-2/);
    });
  });
});
