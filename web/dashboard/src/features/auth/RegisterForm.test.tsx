/**
 * RegisterForm Component Tests
 * Comprehensive tests for the registration form component
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { RegisterForm } from './RegisterForm';

describe('RegisterForm', () => {
  const mockOnSubmit = vi.fn();

  beforeEach(() => {
    mockOnSubmit.mockClear();
  });

  describe('Rendering', () => {
    it('should render all form fields', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/full name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/email address/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument();
    });

    it('should render submit button with correct text', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument();
    });

    it('should render all required field indicators', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmInput = screen.getByLabelText(/confirm password/i);

      expect(nameInput).toHaveAttribute('required');
      expect(emailInput).toHaveAttribute('required');
      expect(passwordInput).toHaveAttribute('required');
      expect(confirmInput).toHaveAttribute('required');
    });

    it('should render password inputs with minLength attribute', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmInput = screen.getByLabelText(/confirm password/i);

      expect(passwordInput).toHaveAttribute('minLength', '8');
      expect(confirmInput).toHaveAttribute('minLength', '8');
    });

    it('should render placeholder text', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      expect(screen.getByPlaceholderText('John Doe')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument();
    });
  });

  describe('Password Confirmation Validation', () => {
    it('should show error when passwords do not match', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(nameInput, 'John Doe');
      await user.type(emailInput, 'john@example.com');
      await user.type(passwordInput, 'password123');
      await user.type(confirmPasswordInput, 'different123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
      });
    });

    it('should not call onSubmit when passwords do not match', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(nameInput, 'John Doe');
      await user.type(emailInput, 'john@example.com');
      await user.type(passwordInput, 'password123');
      await user.type(confirmPasswordInput, 'different123');
      await user.click(submitButton);

      expect(mockOnSubmit).not.toHaveBeenCalled();
    });

    it('should allow submission when passwords match', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(nameInput, 'John Doe');
      await user.type(emailInput, 'john@example.com');
      await user.type(passwordInput, 'password123');
      await user.type(confirmPasswordInput, 'password123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith('John Doe', 'john@example.com', 'password123');
      });
    });
  });

  describe('Password Length Validation', () => {
    it('should show error when password is less than 8 characters', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(nameInput, 'John Doe');
      await user.type(emailInput, 'john@example.com');
      await user.type(passwordInput, 'pass123');
      await user.type(confirmPasswordInput, 'pass123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Password must be at least 8 characters')).toBeInTheDocument();
      });
    });

    it('should not call onSubmit when password is too short', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(nameInput, 'John Doe');
      await user.type(emailInput, 'john@example.com');
      await user.type(passwordInput, 'short');
      await user.type(confirmPasswordInput, 'short');
      await user.click(submitButton);

      expect(mockOnSubmit).not.toHaveBeenCalled();
    });
  });

  describe('Form Submission', () => {
    it('should call onSubmit with all form values', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(nameInput, 'Jane Doe');
      await user.type(emailInput, 'jane@example.com');
      await user.type(passwordInput, 'securepass123');
      await user.type(confirmPasswordInput, 'securepass123');
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledTimes(1);
        expect(mockOnSubmit).toHaveBeenCalledWith('Jane Doe', 'jane@example.com', 'securepass123');
      });
    });
  });

  describe('Error Handling', () => {
    it('should display error prop when provided', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} error="Email already exists" />);

      expect(screen.getByText('Email already exists')).toBeInTheDocument();
    });

    it('should display password validation error when present', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} error="Password must be at least 8 characters" />);

      expect(screen.getByText('Password must be at least 8 characters')).toBeInTheDocument();
    });

    it('should prioritize password error over prop error', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} error="Server error" />);

      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(passwordInput, 'short');
      await user.type(confirmInput, 'different');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
        expect(screen.queryByText('Server error')).not.toBeInTheDocument();
      });
    });

    it('should clear password error when passwords start matching', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmInput = screen.getByLabelText(/confirm password/i);
      const submitButton = screen.getByRole('button', { name: /create account/i });

      await user.type(passwordInput, 'password123');
      await user.type(confirmInput, 'different');
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('Passwords do not match')).toBeInTheDocument();
      });

      await user.clear(confirmInput);
      await user.type(confirmInput, 'password123');

      await waitFor(() => {
        expect(screen.queryByText('Passwords do not match')).not.toBeInTheDocument();
      });
    });
  });

  describe('Loading State', () => {
    it('should show loading text when isLoading is true', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} isLoading={true} />);

      expect(screen.getByRole('button', { name: /please wait/i/i })).toBeInTheDocument();
    });

    it('should disable submit button when loading', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} isLoading={true} />);

      const submitButton = screen.getByRole('button', { name: /please wait/i/i });
      expect(submitButton).toBeDisabled();
    });

    it('should not show loading text when isLoading is false', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} isLoading={false} />);

      expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument();
    });
  });

  describe('User Interaction', () => {
    it('should update input values as user types', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      await user.type(nameInput, 'Alice');

      expect(nameInput).toHaveValue('Alice');
    });

    it('should handle form submission via Enter key', async () => {
      const user = userEvent.setup();
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const nameInput = screen.getByLabelText(/full name/i);
      const emailInput = screen.getByLabelText(/email address/i);
      const passwordInput = screen.getByLabelText(/^password$/i);
      const confirmInput = screen.getByLabelText(/confirm password/i);

      await user.type(nameInput, 'Bob');
      await user.type(emailInput, 'bob@example.com');
      await user.type(passwordInput, 'password123');
      await user.type(confirmInput, 'password123{Enter}');

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith('Bob', 'bob@example.com', 'password123');
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper label associations', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/full name/i)).toHaveAttribute('id', 'register-name');
      expect(screen.getByLabelText(/email address/i)).toHaveAttribute('id', 'register-email');
      expect(screen.getByLabelText(/^password$/i)).toHaveAttribute('id', 'register-password');
      expect(screen.getByLabelText(/confirm password/i)).toHaveAttribute('id', 'register-confirm-password');
    });

    it('should have proper autoComplete attributes', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/full name/i)).toHaveAttribute('autoComplete', 'name');
      expect(screen.getByLabelText(/email address/i)).toHaveAttribute('autoComplete', 'email');
      expect(screen.getByLabelText(/^password$/i)).toHaveAttribute('autoComplete', 'new-password');
      expect(screen.getByLabelText(/confirm password/i)).toHaveAttribute('autoComplete', 'new-password');
    });
  });

  describe('Styling', () => {
    it('should apply gradient styling to submit button', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const submitButton = screen.getByRole('button', { name: /create account/i });
      expect(submitButton).toHaveClass(/bg-gradient-to-r/);
    });

    it('should apply focus ring styling to inputs', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} />);

      const emailInput = screen.getByLabelText(/email address/i);
      expect(emailInput).toHaveClass(/focus:ring-2/);
    });

    it('should apply error styling when error is present', () => {
      render(<RegisterForm onSubmit={mockOnSubmit} error="Test error" />);

      const errorDiv = screen.getByText('Test error').closest('div');
      expect(errorDiv).toHaveClass(/bg-red-100/);
    });
  });
});
