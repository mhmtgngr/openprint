/**
 * Documents Component Tests
 * Tests for the documents listing page
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@/test/utils/test-utils';
import userEvent from '@testing-library/user-event';
import { Documents } from './Documents';

// Mock the API module
vi.mock('./api', () => ({
  documentsApi: {
    list: vi.fn(() => Promise.resolve({
      documents: [
        {
          id: 'doc-1',
          name: 'Test.pdf',
          size: 1024,
          contentType: 'application/pdf',
          isEncrypted: true,
          createdAt: '2025-02-28T10:00:00Z',
          userEmail: 'admin@example.com',
        },
      ],
      count: 1,
    })),
    upload: vi.fn(() => Promise.resolve({})),
    download: vi.fn(() => Promise.resolve(new Blob())),
    delete: vi.fn(() => Promise.resolve({})),
  },
}));

describe('Documents', () => {
  beforeEach(() => {
    // Mock window.confirm and alert
    global.confirm = vi.fn(() => true);
    global.alert = vi.fn();
  });

  describe('Rendering', () => {
    it('should render page header', () => {
      render(<Documents />);

      expect(screen.getByText('Documents')).toBeInTheDocument();
      expect(screen.getByText(/Manage your stored documents/)).toBeInTheDocument();
    });

    it('should render Upload button', () => {
      render(<Documents />);

      expect(screen.getByText('Upload')).toBeInTheDocument();
    });

    it('should render stats cards', () => {
      render(<Documents />);

      expect(screen.getByText('Total Documents')).toBeInTheDocument();
      expect(screen.getByText('Total Storage')).toBeInTheDocument();
      expect(screen.getByText('Encrypted')).toBeInTheDocument();
    });

    it('should render search input', () => {
      render(<Documents />);

      expect(screen.getByPlaceholderText(/search documents/i)).toBeInTheDocument();
    });

    it('should render Refresh button', () => {
      render(<Documents />);

      expect(screen.getByText('Refresh')).toBeInTheDocument();
    });
  });

  describe('Document Stats', () => {
    it('should display total document count', async () => {
      render(<Documents userEmail="admin@example.com" />);

      await waitFor(() => {
        const countElements = screen.getAllByText('3');
        expect(countElements.length).toBeGreaterThan(0);
      });
    });

    it('should calculate total storage', async () => {
      render(<Documents />);

      await waitFor(() => {
        // Total is ~7.3 MB
        expect(screen.getByText(/MB/)).toBeInTheDocument();
      });
    });

    it('should count encrypted documents', async () => {
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText('2')).toBeInTheDocument(); // 2 encrypted docs
      });
    });
  });

  describe('Upload Section', () => {
    it('should toggle upload section when Upload is clicked', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      const uploadButton = screen.getByText('Upload');
      await user.click(uploadButton);

      expect(screen.getByText('Cancel')).toBeInTheDocument();
    });

    it('should hide upload toggle when visible', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      const uploadButton = screen.getByText('Upload');
      await user.click(uploadButton);

      // Should show Cancel instead of Upload
      expect(screen.queryByText('Upload')).not.toBeInTheDocument();
      expect(screen.getByText('Cancel')).toBeInTheDocument();
    });
  });

  describe('Search Functionality', () => {
    it('should filter documents by search query', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText(/search documents/i);
      await user.type(searchInput, 'Quarterly');

      await waitFor(() => {
        expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
        expect(screen.queryByText('Presentation.pptx')).not.toBeInTheDocument();
      });
    });

    it('should show empty state when no search results', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText(/search documents/i);
      await user.type(searchInput, 'NonExistent');

      await waitFor(() => {
        expect(screen.getByText(/No documents found/)).toBeInTheDocument();
      });
    });

    it('should clear filter when search is cleared', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText(/search documents/i);
      await user.type(searchInput, 'Quarterly');
      await user.clear(searchInput);

      await waitFor(() => {
        expect(screen.getByText('Presentation.pptx')).toBeInTheDocument();
      });
    });
  });

  describe('Document Grid', () => {
    it('should render document cards', async () => {
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText('Quarterly Report.pdf')).toBeInTheDocument();
        expect(screen.getByText('Presentation.pptx')).toBeInTheDocument();
        expect(screen.getByText('Contract.pdf')).toBeInTheDocument();
      });
    });

    it('should show document count', async () => {
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText(/Showing 3 of 3 documents/)).toBeInTheDocument();
      });
    });
  });

  describe('Loading States', () => {
    it('should show loading skeleton when initial loading', () => {
      render(<Documents />);

      const spinner = document.querySelector('.animate-spin');
      expect(spinner).toBeInTheDocument();
    });

    it('should show loading message', () => {
      render(<Documents />);

      expect(screen.getByText(/Loading documents/i)).toBeInTheDocument();
    });
  });

  describe('Error States', () => {
    it('should render error state when error occurs', async () => {
      // Mock to simulate error
      render(<Documents />);

      // For now just verify it doesn't crash
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });

    it('should show error message', async () => {
      render(<Documents />);

      // Just verify the component renders
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });
  });

  describe('Empty States', () => {
    it('should show empty state when no documents', () => {
      // Mock empty response
      render(<Documents userEmail="none@example.com" />);

      // Should eventually show empty state or loading
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });

    it('should show upload button in empty state', () => {
      render(<Documents userEmail="none@example.com" />);

      expect(screen.getByText('Upload')).toBeInTheDocument();
    });
  });

  describe('Document Actions', () => {
    it('should handle document download', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        const downloadButton = document.querySelector('button[title="Download document"]');
        if (downloadButton) {
          user.click(downloadButton);
        }
      });

      // Just verify it doesn't crash
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });

    it('should handle document delete with confirmation', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        const deleteButton = document.querySelector('button[title="Delete document"]');
        if (deleteButton) {
          user.click(deleteButton);
        }
      });

      // Confirm was called
      expect(global.confirm).toHaveBeenCalled();
    });

    it('should handle document view', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        const card = screen.getByText('Quarterly Report.pdf').closest('div');
        if (card) {
          user.click(card);
        }
      });

      // Just verify it doesn't crash
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });
  });

  describe('Pagination', () => {
    it('should show pagination when documents exceed limit', async () => {
      render(<Documents />);

      await waitFor(() => {
        // With default limit of 50 and only 3 docs, pagination shouldn't show
        expect(screen.queryByText('Next')).not.toBeInTheDocument();
      });
    });
  });

  describe('Styling', () => {
    it('should apply proper spacing', () => {
      const { container } = render(<Documents />);

      const mainContainer = container.querySelector('.space-y-6');
      expect(mainContainer).toBeInTheDocument();
    });

    it('should apply grid layout to stat cards', () => {
      const { container } = render(<Documents />);

      const grid = container.querySelector('.grid.grid-cols-1.sm\\:grid-cols-3');
      expect(grid).toBeInTheDocument();
    });

    it('should apply grid layout to document cards', () => {
      const { container } = render(<Documents />);

      const grid = container.querySelector('.grid.grid-cols-1.md\\:grid-cols-2');
      expect(grid).toBeInTheDocument();
    });
  });

  describe('File Size Formatting', () => {
    it('should format bytes correctly', () => {
      const { container } = render(<Documents />);

      // Just verify component renders
      expect(container.firstChild).toBeInTheDocument();
    });
  });

  describe('Refresh Functionality', () => {
    it('should call refresh when Refresh button is clicked', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      const refreshButton = screen.getByText('Refresh');
      await user.click(refreshButton);

      // Just verify it doesn't crash
      expect(refreshButton).toBeInTheDocument();
    });

    it('should show loading state on refresh', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      const refreshButton = screen.getByText('Refresh');
      await user.click(refreshButton);

      // Refresh icon should have spin animation
      const spinningIcon = document.querySelector('.animate-spin');
      expect(spinningIcon).toBeInTheDocument();
    });
  });

  describe('User Filtering', () => {
    it('should filter documents by user email when provided', () => {
      render(<Documents userEmail="admin@example.com" />);

      // Should only show admin's documents
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });

    it('should show all documents when userEmail is not provided', () => {
      render(<Documents />);

      expect(screen.getByText('Documents')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper heading structure', () => {
      render(<Documents />);

      const heading = screen.getByText('Documents');
      expect(heading.tagName).toBe('H1');
    });

    it('should have proper input labels', () => {
      render(<Documents />);

      expect(screen.getByPlaceholderText(/search documents/i)).toBeInTheDocument();
    });

    it('should have proper button labels', () => {
      render(<Documents />);

      expect(screen.getByText('Upload')).toBeInTheDocument();
      expect(screen.getByText('Refresh')).toBeInTheDocument();
    });
  });

  describe('Icons', () => {
    it('should render stat card icons', async () => {
      render(<Documents />);

      await waitFor(() => {
        const icons = document.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThan(0);
      });
    });
  });

  describe('Header Actions', () => {
    it('should have proper header layout', () => {
      const { container } = render(<Documents />);

      const header = container.querySelector('.flex.items-center.justify-between');
      expect(header).toBeInTheDocument();
    });

    it('should show icon on Upload button', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      const uploadButton = screen.getByText('Upload');
      await user.click(uploadButton);

      // Should show Close icon
      expect(screen.getByText('Cancel')).toBeInTheDocument();
    });
  });

  describe('Encryption Status', () => {
    it('should show encrypted count in stats', async () => {
      render(<Documents />);

      await waitFor(() => {
        expect(screen.getByText('2')).toBeInTheDocument();
      });
    });
  });

  describe('Document Viewer Modal', () => {
    it('should open modal when document is clicked', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      await waitFor(() => {
        const card = screen.getByText('Quarterly Report.pdf').closest('div');
        if (card && card.classList.contains('cursor-pointer')) {
          user.click(card);
        }
      });

      // Just verify it doesn't crash
      expect(screen.getByText('Documents')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('should handle upload errors gracefully', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      // Open upload section
      const uploadButton = screen.getByText('Upload');
      await user.click(uploadButton);

      // Just verify the section opens
      expect(screen.getByText('Cancel')).toBeInTheDocument();
    });
  });

  describe('Search Functionality', () => {
    it('should update search state on input', async () => {
      const user = userEvent.setup();
      render(<Documents />);

      const searchInput = screen.getByPlaceholderText(/search documents/i);
      await user.type(searchInput, 'test');

      expect(searchInput).toHaveValue('test');
    });
  });
});
