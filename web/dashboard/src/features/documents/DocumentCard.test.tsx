/**
 * DocumentCard Component Tests
 * Tests for document preview card component
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { DocumentCard } from './DocumentCard';
import type { Document } from './types';

const mockDocument: Document = {
  id: 'doc-1',
  name: 'Quarterly Report.pdf',
  size: 2048576,
  contentType: 'application/pdf',
  isEncrypted: true,
  createdAt: '2025-02-20T10:00:00Z',
  userEmail: 'admin@example.com',
};

const mockImageDocument: Document = {
  id: 'doc-2',
  name: 'Photo.jpg',
  size: 524288,
  contentType: 'image/jpeg',
  isEncrypted: false,
  createdAt: '2025-02-25T14:30:00Z',
  userEmail: 'user@example.com',
};

const mockWordDocument: Document = {
  id: 'doc-3',
  name: 'Contract.docx',
  size: 102400,
  contentType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  isEncrypted: true,
  createdAt: '2025-02-26T09:15:00Z',
  userEmail: 'admin@example.com',
  checksum: 'abc123def456',
};

describe('DocumentCard', () => {
  describe('Rendering', () => {
    it('should render document name', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
    });

    it('should render file size', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText('2 MB')).toBeInTheDocument();
    });

    it('should render content type label', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText('PDF')).toBeInTheDocument();
    });

    it('should render upload date', () => {
      render(<DocumentCard document={mockDocument} />);

      // The component uses formatDate which returns a formatted date
      expect(screen.getByText(/Uploaded/i)).toBeInTheDocument();
    });

    it('should render document icon', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const icon = container.querySelector('svg');
      expect(icon).toBeInTheDocument();
    });

    it('should apply card base styling', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('bg-white');
      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('border');
    });
  });

  describe('Document Icons', () => {
    it('should render PDF icon for PDF documents', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const icon = container.querySelector('svg');
      expect(icon).toBeInTheDocument();
    });

    it('should render image icon for image documents', () => {
      render(<DocumentCard document={mockImageDocument} />);

      expect(screen.getByText('JPEG Image')).toBeInTheDocument();
    });

    it('should render Word icon for Word documents', () => {
      render(<DocumentCard document={mockWordDocument} />);

      expect(screen.getByText('Word Document')).toBeInTheDocument();
    });
  });

  describe('Encryption Status', () => {
    it('should show encrypted status when document is encrypted', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText('Encrypted')).toBeInTheDocument();
    });

    it('should show lock icon for encrypted documents', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const lockIcon = container.querySelector('svg.text-green-600');
      expect(lockIcon).toBeInTheDocument();
    });

    it('should show not encrypted status when document is not encrypted', () => {
      render(<DocumentCard document={mockImageDocument} />);

      expect(screen.getByText('Not encrypted')).toBeInTheDocument();
    });

    it('should show unlock icon for non-encrypted documents', () => {
      const { container } = render(<DocumentCard document={mockImageDocument} />);

      const unlockIcon = container.querySelector('svg.text-gray-400');
      expect(unlockIcon).toBeInTheDocument();
    });

    it('should not render encryption section when isEncrypted is undefined', () => {
      const docWithoutEncryption = { ...mockDocument, isEncrypted: undefined };
      render(<DocumentCard document={docWithoutEncryption} />);

      expect(screen.queryByText('Encrypted')).not.toBeInTheDocument();
      expect(screen.queryByText('Not encrypted')).not.toBeInTheDocument();
    });
  });

  describe('Actions', () => {
    it('should render download button when onDownload is provided', () => {
      const handleDownload = vi.fn();
      render(<DocumentCard document={mockDocument} onDownload={handleDownload} />);

      const downloadButton = document.querySelector('button[title="Download document"]');
      expect(downloadButton).toBeInTheDocument();
    });

    it('should not render download button when onDownload is not provided', () => {
      render(<DocumentCard document={mockDocument} />);

      const downloadButton = document.querySelector('button[title="Download document"]');
      expect(downloadButton).not.toBeInTheDocument();
    });

    it('should call onDownload when download button is clicked', async () => {
      const user = userEvent.setup();
      const handleDownload = vi.fn();

      render(<DocumentCard document={mockDocument} onDownload={handleDownload} />);

      const downloadButton = document.querySelector('button[title="Download document"]') as HTMLElement;
      await user.click(downloadButton);

      expect(handleDownload).toHaveBeenCalledTimes(1);
    });

    it('should render delete button when onDelete is provided', () => {
      const handleDelete = vi.fn();
      render(<DocumentCard document={mockDocument} onDelete={handleDelete} />);

      const deleteButton = document.querySelector('button[title="Delete document"]');
      expect(deleteButton).toBeInTheDocument();
    });

    it('should not render delete button when onDelete is not provided', () => {
      render(<DocumentCard document={mockDocument} />);

      const deleteButton = document.querySelector('button[title="Delete document"]');
      expect(deleteButton).not.toBeInTheDocument();
    });

    it('should call onDelete when delete button is clicked', async () => {
      const user = userEvent.setup();
      const handleDelete = vi.fn();

      render(<DocumentCard document={mockDocument} onDelete={handleDelete} />);

      const deleteButton = document.querySelector('button[title="Delete document"]') as HTMLElement;
      await user.click(deleteButton);

      expect(handleDelete).toHaveBeenCalledTimes(1);
    });

    it('should show spinner when isDeleting is true', () => {
      const handleDelete = vi.fn();
      render(<DocumentCard document={mockDocument} onDelete={handleDelete} isDeleting={true} />);

      const spinner = document.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();
    });

    it('should disable delete button when isDeleting is true', () => {
      const handleDelete = vi.fn();
      render(<DocumentCard document={mockDocument} onDelete={handleDelete} isDeleting={true} />);

      const deleteButton = document.querySelector('button[title="Delete document"]') as HTMLElement;
      expect(deleteButton).toBeDisabled();
    });

    it('should not propagate click when action button is clicked', async () => {
      const user = userEvent.setup();
      const handleClick = vi.fn();
      const handleDownload = vi.fn();

      render(
        <DocumentCard document={mockDocument} onClick={handleClick} onDownload={handleDownload} />
      );

      const downloadButton = document.querySelector('button[title="Download document"]') as HTMLElement;
      await user.click(downloadButton);

      expect(handleDownload).toHaveBeenCalledTimes(1);
      expect(handleClick).not.toHaveBeenCalled();
    });
  });

  describe('Card Click', () => {
    it('should call onClick when card is clicked', async () => {
      const user = userEvent.setup();
      const handleClick = vi.fn();

      render(<DocumentCard document={mockDocument} onClick={handleClick} />);

      const card = screen.getByText('Quarterly Report.pdf').closest('div');
      await user.click(card!);

      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should apply cursor-pointer when onClick is provided', () => {
      const { container } = render(<DocumentCard document={mockDocument} onClick={vi.fn()} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('cursor-pointer');
    });
  });

  describe('Metadata Display', () => {
    it('should render type label', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText(/Type/i)).toBeInTheDocument();
    });

    it('should render uploaded label', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText(/Uploaded/i)).toBeInTheDocument();
    });

    it('should render checksum when present', () => {
      render(<DocumentCard document={mockWordDocument} />);

      expect(screen.getByText(/Checksum/i)).toBeInTheDocument();
      expect(screen.getByText(/abc123/)).toBeInTheDocument();
    });

    it('should not render checksum when not present', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.queryByText(/Checksum/i)).not.toBeInTheDocument();
    });

    it('should truncate checksum display', () => {
      render(<DocumentCard document={mockWordDocument} />);

      // Checksum should be truncated with ellipsis
      expect(screen.getByText(/abc123/)).toBeInTheDocument();
    });
  });

  describe('File Size Formatting', () => {
    it('should format bytes correctly', () => {
      render(<DocumentCard document={{ ...mockDocument, size: 1024 }} />);

      expect(screen.getByText('1 KB')).toBeInTheDocument();
    });

    it('should format KB correctly', () => {
      render(<DocumentCard document={{ ...mockDocument, size: 1536 }} />);

      expect(screen.getByText(/1\.5 KB/)).toBeInTheDocument();
    });

    it('should format MB correctly', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText('2 MB')).toBeInTheDocument();
    });

    it('should format GB correctly', () => {
      render(<DocumentCard document={{ ...mockDocument, size: 1073741824 }} />);

      expect(screen.getByText(/1 GB/)).toBeInTheDocument();
    });

    it('should handle zero size', () => {
      render(<DocumentCard document={{ ...mockDocument, size: 0 }} />);

      expect(screen.getByText('0 B')).toBeInTheDocument();
    });
  });

  describe('Content Type Labels', () => {
    it('should show PDF for PDF files', () => {
      render(<DocumentCard document={mockDocument} />);

      expect(screen.getByText('PDF')).toBeInTheDocument();
    });

    it('should show JPEG Image for JPEG files', () => {
      render(<DocumentCard document={mockImageDocument} />);

      expect(screen.getByText('JPEG Image')).toBeInTheDocument();
    });

    it('should show Word Document for DOCX files', () => {
      render(<DocumentCard document={mockWordDocument} />);

      expect(screen.getByText('Word Document')).toBeInTheDocument();
    });

    it('should show Excel Spreadsheet for XLSX files', () => {
      const excelDoc: Document = {
        ...mockDocument,
        contentType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      };
      render(<DocumentCard document={excelDoc} />);

      expect(screen.getByText('Excel Spreadsheet')).toBeInTheDocument();
    });

    it('should show Text for text files', () => {
      const textDoc: Document = {
        ...mockDocument,
        contentType: 'text/plain',
      };
      render(<DocumentCard document={textDoc} />);

      expect(screen.getByText('Text')).toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('should apply hover effect', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const card = container.firstChild as HTMLElement;
      expect(card).toHaveClass('hover:shadow-md');
    });

    it('should apply icon container styling', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const iconContainer = container.querySelector('.bg-blue-50');
      expect(iconContainer).toBeInTheDocument();
    });

    it('should apply action button hover effects', () => {
      const { container } = render(<DocumentCard document={mockDocument} onDownload={vi.fn()} />);

      const button = container.querySelector('.hover\\:text-blue-600');
      expect(button).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper button titles', () => {
      const handleDownload = vi.fn();
      const handleDelete = vi.fn();

      render(
        <DocumentCard document={mockDocument} onDownload={handleDownload} onDelete={handleDelete} />
      );

      expect(document.querySelector('button[title="Download document"]')).toBeInTheDocument();
      expect(document.querySelector('button[title="Delete document"]')).toBeInTheDocument();
    });

    it('should have proper heading for document name', () => {
      render(<DocumentCard document={mockDocument} />);

      const name = screen.getByText('Quarterly Report.pdf');
      expect(name.tagName).toBe('H3');
    });

    it('should have title attribute on document name for full filename', () => {
      render(<DocumentCard document={mockDocument} />);

      const name = screen.getByText('Quarterly Report.pdf');
      expect(name).toHaveAttribute('title', 'Quarterly Report.pdf');
    });
  });

  describe('Layout', () => {
    it('should have proper flex layout', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const header = container.querySelector('.flex.items-start.gap-3');
      expect(header).toBeInTheDocument();
    });

    it('should have proper metadata section', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const metadata = container.querySelector('.space-y-2');
      expect(metadata).toBeInTheDocument();
    });

    it('should have proper footer with encryption status', () => {
      const { container } = render(<DocumentCard document={mockDocument} />);

      const footer = container.querySelector('.mt-3.pt-3.border-t');
      expect(footer).toBeInTheDocument();
    });
  });

  describe('Different Document Types', () => {
    it('should handle PNG images', () => {
      const pngDoc: Document = {
        ...mockDocument,
        contentType: 'image/png',
      };
      render(<DocumentCard document={pngDoc} />);

      expect(screen.getByText('PNG Image')).toBeInTheDocument();
    });

    it('should handle GIF images', () => {
      const gifDoc: Document = {
        ...mockDocument,
        contentType: 'image/gif',
      };
      render(<DocumentCard document={gifDoc} />);

      expect(screen.getByText('GIF Image')).toBeInTheDocument();
    });

    it('should handle WebP images', () => {
      const webpDoc: Document = {
        ...mockDocument,
        contentType: 'image/webp',
      };
      render(<DocumentCard document={webpDoc} />);

      expect(screen.getByText('WebP Image')).toBeInTheDocument();
    });

    it('should handle legacy Word documents', () => {
      const docDoc: Document = {
        ...mockDocument,
        contentType: 'application/msword',
      };
      render(<DocumentCard document={docDoc} />);

      expect(screen.getByText('Word Document')).toBeInTheDocument();
    });

    it('should handle legacy Excel documents', () => {
      const xlsDoc: Document = {
        ...mockDocument,
        contentType: 'application/vnd.ms-excel',
      };
      render(<DocumentCard document={xlsDoc} />);

      expect(screen.getByText('Excel Spreadsheet')).toBeInTheDocument();
    });
  });

  describe('Empty/Edge Cases', () => {
    it('should handle documents without owner email', () => {
      const docWithoutOwner = { ...mockDocument, ownerEmail: undefined };
      expect(() => render(<DocumentCard document={docWithoutOwner} />)).not.toThrow();
    });

    it('should handle documents with very long names', () => {
      const longNameDoc = {
        ...mockDocument,
        name: 'A'.repeat(200) + '.pdf',
      };
      render(<DocumentCard document={longNameDoc} />);

      const nameElement = screen.getByText(/A+/);
      expect(nameElement).toHaveClass('truncate');
    });

    it('should handle documents with very large size', () => {
      const largeDoc = {
        ...mockDocument,
        size: 1099511627776, // 1 TB
      };
      render(<DocumentCard document={largeDoc} />);

      expect(screen.getByText(/TB/)).toBeInTheDocument();
    });
  });
});
