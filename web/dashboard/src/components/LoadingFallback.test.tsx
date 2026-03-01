import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import {
  LoadingFallback,
  PageLoadingFallback,
  InlineLoadingFallback,
} from './LoadingFallback';

describe('LoadingFallback', () => {
  describe('LoadingFallback', () => {
    it('should render spinner with default message', () => {
      render(<LoadingFallback />);
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
      expect(screen.getByTestId('loading-message')).toHaveTextContent('Loading...');
    });

    it('should render custom message', () => {
      render(<LoadingFallback message="Please wait..." />);
      expect(screen.getByTestId('loading-message')).toHaveTextContent('Please wait...');
    });

    it('should render small size spinner', () => {
      const { container: _container } = render(<LoadingFallback size="sm" />);
      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toHaveClass('w-8 h-8 border-2');
    });

    it('should render medium size spinner (default)', () => {
      const { container: _container } = render(<LoadingFallback size="md" />);
      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toHaveClass('w-12 h-12 border-4');
    });

    it('should render large size spinner', () => {
      const { container: _container } = render(<LoadingFallback size="lg" />);
      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toHaveClass('w-16 h-16 border-4');
    });

    it('should have correct CSS classes for full-screen layout', () => {
      const { container } = render(<LoadingFallback />);
      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('min-h-screen', 'flex', 'items-center', 'justify-center');
    });

    it('should apply dark mode classes', () => {
      const { container } = render(<LoadingFallback />);
      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('bg-gray-50', 'dark:bg-gray-900');
    });
  });

  describe('PageLoadingFallback', () => {
    it('should render with default page loading message', () => {
      render(<PageLoadingFallback />);
      expect(screen.getByTestId('page-loading')).toBeInTheDocument();
      expect(screen.getByText('Loading page...')).toBeInTheDocument();
    });

    it('should render custom message', () => {
      render(<PageLoadingFallback message="Loading dashboard..." />);
      expect(screen.getByText('Loading dashboard...')).toBeInTheDocument();
    });

    it('should have padding for page-level loading', () => {
      const { container } = render(<PageLoadingFallback />);
      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center', 'p-8');
    });

    it('should have smaller spinner than LoadingFallback', () => {
      const { container } = render(<PageLoadingFallback />);
      const spinner = container.querySelector('.animate-spin') as HTMLElement;
      expect(spinner).toHaveClass('w-8 h-8', 'border-3');
    });
  });

  describe('InlineLoadingFallback', () => {
    it('should render inline loading state', () => {
      render(<InlineLoadingFallback />);
      expect(screen.getByTestId('inline-loading')).toBeInTheDocument();
    });

    it('should render without message by default', () => {
      render(<InlineLoadingFallback />);
      const message = screen.queryByText(/loading/i);
      expect(message).not.toBeInTheDocument();
    });

    it('should render with custom message', () => {
      render(<InlineLoadingFallback message="Saving..." />);
      expect(screen.getByText('Saving...')).toBeInTheDocument();
    });

    it('should have horizontal layout', () => {
      const { container } = render(<InlineLoadingFallback />);
      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('flex', 'items-center');
    });

    it('should have smallest spinner for inline use', () => {
      const { container } = render(<InlineLoadingFallback />);
      const spinner = container.querySelector('.animate-spin') as HTMLElement;
      expect(spinner).toHaveClass('w-4 h-4', 'border-2');
    });

    it('should have vertical padding', () => {
      const { container } = render(<InlineLoadingFallback />);
      const wrapper = container.firstChild as HTMLElement;
      expect(wrapper).toHaveClass('py-4');
    });
  });

  describe('Accessibility', () => {
    it('should have accessible loading messages', () => {
      render(<LoadingFallback message="Loading your data" />);
      const message = screen.getByText('Loading your data');
      expect(message).toHaveClass('text-gray-600', 'dark:text-gray-400');
    });

    it('should have focusable spinner element for screen readers', () => {
      render(<LoadingFallback />);
      const spinner = screen.getByTestId('loading-spinner');
      expect(spinner).toBeInTheDocument();
    });
  });
});
