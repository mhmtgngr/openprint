/**
 * JobStatusBadge Component Tests
 * Comprehensive tests for the job status badge component
 */

import { render, screen } from '@/test/utils/test-utils';
import { describe, it, expect } from 'vitest';
import { JobStatusBadge } from './JobStatusBadge';

describe('JobStatusBadge', () => {
  describe('Status Rendering', () => {
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
  });

  describe('Status Dot', () => {
    it('should render status dot for each status', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const dot = container.querySelector('.w-1\\.5.h-1\\.5.rounded-full');
      expect(dot).toBeInTheDocument();
    });

    it('should render green dot for completed status', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const dot = container.querySelector('.bg-green-500');
      expect(dot).toBeInTheDocument();
    });

    it('should render gray dot for queued status', () => {
      const { container } = render(<JobStatusBadge status="queued" />);

      const dot = container.querySelector('.bg-gray-400');
      expect(dot).toBeInTheDocument();
    });

    it('should render blue dot for processing status', () => {
      const { container } = render(<JobStatusBadge status="processing" />);

      const dot = container.querySelector('.bg-blue-500');
      expect(dot).toBeInTheDocument();
    });

    it('should render red dot for failed status', () => {
      const { container } = render(<JobStatusBadge status="failed" />);

      const dot = container.querySelector('.bg-red-500');
      expect(dot).toBeInTheDocument();
    });

    it('should render gray dot for cancelled status', () => {
      const { container } = render(<JobStatusBadge status="cancelled" />);

      const dot = container.querySelector('.bg-gray-400');
      expect(dot).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply correct background color for each status', () => {
      const { container: completedContainer } = render(<JobStatusBadge status="completed" />);
      const { container: queuedContainer } = render(<JobStatusBadge status="queued" />);
      const { container: failedContainer } = render(<JobStatusBadge status="failed" />);

      expect(completedContainer.querySelector('.bg-green-100')).toBeInTheDocument();
      expect(queuedContainer.querySelector('.bg-gray-100')).toBeInTheDocument();
      expect(failedContainer.querySelector('.bg-red-100')).toBeInTheDocument();
    });

    it('should apply correct text color for each status', () => {
      render(<JobStatusBadge status="completed" />);

      const badge = screen.getByText('Completed').closest('span');
      expect(badge).toHaveClass(/text-green-/);
    });

    it('should apply rounded-full class', () => {
      const { container } = render(<JobStatusBadge status="queued" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge).toHaveClass('rounded-full');
    });

    it('should apply inline-flex class', () => {
      const { container } = render(<JobStatusBadge status="queued" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge).toHaveClass('inline-flex');
    });

    it('should apply font-medium class', () => {
      const { container } = render(<JobStatusBadge status="queued" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge).toHaveClass('font-medium');
    });
  });

  describe('Size Variants', () => {
    it('should render default padding', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge.className).toContain('px-2.5');
    });
  });

  describe('Custom ClassName', () => {
    it('should apply custom className', () => {
      const { container } = render(<JobStatusBadge status="completed" className="custom-class" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge).toHaveClass('custom-class');
    });

    it('should preserve default classes with custom className', () => {
      const { container } = render(<JobStatusBadge status="completed" className="custom-class" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge).toHaveClass('rounded-full');
      expect(badge).toHaveClass('custom-class');
    });
  });

  describe('Show Icon', () => {
    it('should not show icon by default', () => {
      const { container } = render(<JobStatusBadge status="processing" />);

      const icon = container.querySelector('svg.animate-spin');
      expect(icon).not.toBeInTheDocument();
    });
  });

  describe('Text Content', () => {
    it('should capitalize first letter of status text', () => {
      render(<JobStatusBadge status="queued" />);

      expect(screen.getByText('Queued')).toBeInTheDocument();
      expect(screen.queryByText('queued')).not.toBeInTheDocument();
    });

    it('should display all status labels correctly', () => {
      const { rerender } = render(<JobStatusBadge status="queued" />);

      expect(screen.getByText('Queued')).toBeInTheDocument();

      rerender(<JobStatusBadge status="processing" />);
      expect(screen.getByText('Processing')).toBeInTheDocument();

      rerender(<JobStatusBadge status="completed" />);
      expect(screen.getByText('Completed')).toBeInTheDocument();

      rerender(<JobStatusBadge status="failed" />);
      expect(screen.getByText('Failed')).toBeInTheDocument();

      rerender(<JobStatusBadge status="cancelled" />);
      expect(screen.getByText('Cancelled')).toBeInTheDocument();
    });
  });

  describe('Dark Mode Support', () => {
    it('should include dark mode classes', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge.className).toContain('dark:');
    });

    it('should apply dark mode background color', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge.className).toContain('dark:bg-green-900/30');
    });

    it('should apply dark mode text color', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge.className).toContain('dark:text-green-300');
    });
  });

  describe('Accessibility', () => {
    it('should be a semantic span element', () => {
      const { container } = render(<JobStatusBadge status="queued" />);

      const badge = container.querySelector('span');
      expect(badge).toBeInTheDocument();
    });

    it('should have descriptive text content', () => {
      render(<JobStatusBadge status="processing" />);

      expect(screen.getByText('Processing')).toBeInTheDocument();
    });
  });

  describe('Layout', () => {
    it('should have gap between dot and text', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge.className).toContain('gap-1.5');
    });

    it('should align items center', () => {
      const { container } = render(<JobStatusBadge status="completed" />);

      const badge = container.firstChild as HTMLElement;
      expect(badge).toHaveClass('items-center');
    });
  });
});
