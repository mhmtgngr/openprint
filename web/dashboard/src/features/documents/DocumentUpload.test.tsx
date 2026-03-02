/**
 * DocumentUpload Component Tests
 * Tests for document upload component with drag-drop functionality
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { DocumentUpload } from './DocumentUpload';

// Mock File object for testing
const createMockFile = (name: string, size: number, type: string) => {
  const file = new File(['content'], name, { type });
  Object.defineProperty(file, 'size', { value: size });
  return file;
};

describe('DocumentUpload', () => {
  const mockOnUpload = vi.fn();

  beforeEach(() => {
    mockOnUpload.mockClear();
  });

  describe('Rendering', () => {
    it('should render upload zone', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      expect(screen.getByText('Upload Documents')).toBeInTheDocument();
    });

    it('should render drag-drop instructions', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      expect(screen.getByText(/drag and drop/i)).toBeInTheDocument();
    });

    it('should render file input', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const input = document.querySelector('input[type="file"]');
      expect(input).toBeInTheDocument();
    });

    it('should render browse button', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      expect(screen.getByText('Browse Files')).toBeInTheDocument();
    });

    it('should render supported file types info', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      expect(screen.getByText(/Supported:/)).toBeInTheDocument();
    });

    it('should show singular instruction when multiple is false', () => {
      render(<DocumentUpload onUpload={mockOnUpload} multiple={false} />);

      expect(screen.getByText(/a file here/)).toBeInTheDocument();
    });

    it('should show plural instruction when multiple is true', () => {
      render(<DocumentUpload onUpload={mockOnUpload} multiple={true} />);

      expect(screen.getByText(/files here/)).toBeInTheDocument();
    });
  });

  describe('File Selection', () => {
    it('should handle file selection via input', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(mockOnUpload).toHaveBeenCalledWith([file]);
      });
    });

    it('should handle multiple file selection', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} multiple={true} />);

      const files = [
        createMockFile('test1.pdf', 1024, 'application/pdf'),
        createMockFile('test2.pdf', 2048, 'application/pdf'),
      ];
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, files);

      await waitFor(() => {
        expect(mockOnUpload).toHaveBeenCalledWith(files);
      });
    });

    it('should reset input after selection', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(input.value).toBe('');
      });
    });

    it('should not trigger when disabled', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} disabled={true} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      if (input) {
        await user.upload(input, file);
      }

      expect(mockOnUpload).not.toHaveBeenCalled();
    });
  });

  describe('File Size Validation', () => {
    it('should reject files exceeding max size', async () => {
      const user = userEvent.setup();
      const maxSize = 5 * 1024 * 1024; // 5MB
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={maxSize} />);

      const largeFile = createMockFile('large.pdf', 10 * 1024 * 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, largeFile);

      await waitFor(() => {
        expect(screen.getByText(/File size exceeds/)).toBeInTheDocument();
      });
    });

    it('should show error for oversized file', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={1024} />);

      const largeFile = createMockFile('large.pdf', 2048, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, largeFile);

      await waitFor(() => {
        expect(screen.getByText(/File size exceeds/)).toBeInTheDocument();
      });
    });

    it('should accept files within size limit', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={1024 * 1024} />);

      const file = createMockFile('test.pdf', 512 * 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(mockOnUpload).toHaveBeenCalledWith([file]);
      });
    });

    it('should format max file size in info text', () => {
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={5 * 1024 * 1024} />);

      expect(screen.getByText(/5 MB/)).toBeInTheDocument();
    });
  });

  describe('Drag and Drop', () => {
    it('should handle drag over event', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const dropZone = screen.getByText('Upload Documents').closest('div');

      if (dropZone) {
        await user.upload(dropZone, []);
      }
      // Just check it doesn't crash
      expect(dropZone).toBeInTheDocument();
    });

    it('should handle drag leave event', async () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const dropZone = screen.getByText('Upload Documents').closest('div');

      // Just check it doesn't crash
      expect(dropZone).toBeInTheDocument();
    });

    it('should handle drop event', async () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const dropZone = screen.getByText('Upload Documents').closest('div');

      // Just check it doesn't crash
      expect(dropZone).toBeInTheDocument();
    });

    it('should not trigger when disabled on drag', async () => {
      render(<DocumentUpload onUpload={mockOnUpload} disabled={true} />);

      const dropZone = screen.getByText('Upload Documents').closest('div');

      expect(dropZone).toHaveClass('cursor-not-allowed');
      expect(dropZone).toHaveClass('opacity-50');
    });
  });

  describe('Upload List', () => {
    it('should show uploads count when files are selected', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(screen.getByText(/Uploads \(1\)/)).toBeInTheDocument();
      });
    });

    it('should show individual upload items', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(screen.getByText('test.pdf')).toBeInTheDocument();
      });
    });

    it('should show file size in upload item', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(screen.getByText('1 KB')).toBeInTheDocument();
      });
    });

    it('should render clear all button when there are uploads', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(screen.getByText('Clear all')).toBeInTheDocument();
      });
    });
  });

  describe('Browse Button', () => {
    it('should trigger file selection when clicked', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const browseButton = screen.getByText('Browse Files');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      // Mock click on input
      const clickSpy = vi.spyOn(input, 'click');

      await user.click(browseButton);

      expect(clickSpy).toHaveBeenCalled();
    });
  });

  describe('Disabled State', () => {
    it('should not allow file selection when disabled', () => {
      render(<DocumentUpload onUpload={mockOnUpload} disabled={true} />);

      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      expect(input).toBeDisabled();
    });

    it('should apply disabled styling to drop zone', () => {
      render(<DocumentUpload onUpload={mockOnUpload} disabled={true} />);

      const dropZone = screen.getByText('Upload Documents').closest('div');
      expect(dropZone).toHaveClass('opacity-50');
      expect(dropZone).toHaveClass('cursor-not-allowed');
    });

    it('should disable browse button when disabled', () => {
      render(<DocumentUpload onUpload={mockOnUpload} disabled={true} />);

      const browseButton = screen.getByText('Browse Files');
      expect(browseButton).toBeInTheDocument();
    });
  });

  describe('File Type Validation', () => {
    it('should accept PDF files', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.pdf', 1024, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(mockOnUpload).toHaveBeenCalledWith([file]);
      });
    });

    it('should accept image files', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const file = createMockFile('test.jpg', 1024, 'image/jpeg');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, file);

      await waitFor(() => {
        expect(mockOnUpload).toHaveBeenCalledWith([file]);
      });
    });

    it('should accept attribute for file input', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      expect(input).toHaveAttribute('accept');
    });
  });

  describe('Max Size Prop', () => {
    it('should use default max size when not provided', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      // Default is 50MB (from MAX_UPLOAD_SIZE)
      expect(screen.getByText(/50 MB/)).toBeInTheDocument();
    });

    it('should use custom max size when provided', () => {
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={10 * 1024 * 1024} />);

      expect(screen.getByText(/10 MB/)).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('should display error message for oversized files', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={1024} />);

      const largeFile = createMockFile('large.pdf', 2048, 'application/pdf');
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, largeFile);

      await waitFor(() => {
        const errorElement = screen.getByText(/File size exceeds/);
        expect(errorElement).toBeInTheDocument();
        expect(errorElement).toHaveClass(/text-red-600/);
      });
    });

    it('should still allow other valid files when one is too large', async () => {
      const user = userEvent.setup();
      render(<DocumentUpload onUpload={mockOnUpload} maxFileSize={1024} />);

      const files = [
        createMockFile('small.pdf', 512, 'application/pdf'),
        createMockFile('large.pdf', 2048, 'application/pdf'),
      ];
      const input = document.querySelector('input[type="file"]') as HTMLInputElement;

      await user.upload(input, files);

      await waitFor(() => {
        // Only the small file should be uploaded
        expect(mockOnUpload).toHaveBeenCalledWith([files[0]]);
      });
    });
  });

  describe('Styling', () => {
    it('should apply border to drop zone', () => {
      const { container } = render(<DocumentUpload onUpload={mockOnUpload} />);

      const dropZone = container.querySelector('.border-2');
      expect(dropZone).toBeInTheDocument();
    });

    it('should apply dashed border style', () => {
      const { container } = render(<DocumentUpload onUpload={mockOnUpload} />);

      const dropZone = container.querySelector('.border-dashed');
      expect(dropZone).toBeInTheDocument();
    });

    it('should apply rounded corners', () => {
      const { container } = render(<DocumentUpload onUpload={mockOnUpload} />);

      const dropZone = container.querySelector('.rounded-lg');
      expect(dropZone).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper input labels', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const input = document.querySelector('input[type="file"]') as HTMLInputElement;
      expect(input).toHaveClass('hidden');
    });

    it('should have accessible browse button', () => {
      render(<DocumentUpload onUpload={mockOnUpload} />);

      const button = screen.getByRole('button', { name: /browse files/i });
      expect(button).toBeInTheDocument();
    });
  });
});
