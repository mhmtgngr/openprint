/**
 * Documents Component - Main listing page for document storage
 */

import { useState, useEffect, useCallback } from 'react';
import type { Document as DocType, DocumentListParams } from './types';
import { documentsApi } from './api';
import { DocumentCard } from './DocumentCard';
import { DocumentUpload } from './DocumentUpload';
import { DocumentViewer } from './DocumentViewer';

interface DocumentsProps {
  userEmail?: string;
}

export const Documents = ({ userEmail }: DocumentsProps) => {
  const [documents, setDocuments] = useState<DocType[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDocument, setSelectedDocument] = useState<DocType | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [showUpload, setShowUpload] = useState(false);

  // Pagination
  const [limit] = useState(50);
  const [offset, setOffset] = useState(0);

  // Fetch documents
  const fetchDocuments = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const params: DocumentListParams = {
        limit,
        offset,
      };

      if (userEmail) {
        params.userEmail = userEmail;
      }

      const response = await documentsApi.list(params);
      setDocuments(response.documents);
      setTotalCount(response.count);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load documents');
    } finally {
      setIsLoading(false);
    }
  }, [userEmail, limit, offset]);

  useEffect(() => {
    fetchDocuments();
  }, [fetchDocuments]);

  // Handle file upload
  const handleUpload = async (files: File[]) => {
    for (const file of files) {
      try {
        await documentsApi.upload(
          { file, userEmail },
          (progress) => {
            // Update progress tracking if needed
            console.log(`Upload progress for ${file.name}: ${progress}%`);
          }
        );
      } catch (err) {
        console.error(`Failed to upload ${file.name}:`, err);
      }
    }

    // Refresh document list
    fetchDocuments();
    setShowUpload(false);
  };

  // Handle document download
  const handleDownload = async (doc: DocType) => {
    try {
      const blob = await documentsApi.download(doc.id, doc.name);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = doc.name;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Download failed:', err);
    }
  };

  // Handle document delete
  const handleDelete = async (doc: DocType) => {
    if (!confirm(`Are you sure you want to delete "${doc.name}"?`)) {
      return;
    }

    setDeletingId(doc.id);

    try {
      await documentsApi.delete(doc.id);
      setDocuments(prev => prev.filter(d => d.id !== doc.id));
      setTotalCount(prev => prev - 1);

      // Close viewer if open
      if (selectedDocument?.id === doc.id) {
        setSelectedDocument(null);
      }
    } catch (err) {
      console.error('Delete failed:', err);
      alert(err instanceof Error ? err.message : 'Failed to delete document');
    } finally {
      setDeletingId(null);
    }
  };

  // Filter documents by search query
  const filteredDocuments = documents.filter(doc =>
    doc.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Calculate storage stats
  const totalStorage = documents.reduce((sum, doc) => sum + doc.size, 0);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Documents
          </h1>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Manage your stored documents and files
          </p>
        </div>

        <button
          onClick={() => setShowUpload(!showUpload)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
        >
          {showUpload ? (
            <>
              <CloseIcon className="w-4 h-4" />
              Cancel
            </>
          ) : (
            <>
              <UploadIcon className="w-4 h-4" />
              Upload
            </>
          )}
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center">
              <DocumentIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {totalCount}
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400">Total Documents</p>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-green-50 dark:bg-green-900/30 flex items-center justify-center">
              <StorageIcon className="w-5 h-5 text-green-600 dark:text-green-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {formatBytes(totalStorage)}
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400">Total Storage</p>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-purple-50 dark:bg-purple-900/30 flex items-center justify-center">
              <EncryptedIcon className="w-5 h-5 text-purple-600 dark:text-purple-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {documents.filter(d => d.isEncrypted).length}
              </p>
              <p className="text-xs text-gray-600 dark:text-gray-400">Encrypted</p>
            </div>
          </div>
        </div>
      </div>

      {/* Upload Section */}
      {showUpload && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
          <DocumentUpload onUpload={handleUpload} />
        </div>
      )}

      {/* Search and Filters */}
      <div className="flex items-center gap-4">
        <div className="flex-1 relative">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Search documents..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder-gray-500 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        <button
          onClick={fetchDocuments}
          disabled={isLoading}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <RefreshIcon className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Error State */}
      {error && (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <div className="flex items-center gap-3">
            <ErrorIcon className="w-5 h-5 text-red-600 dark:text-red-400" />
            <div>
              <p className="font-medium text-red-900 dark:text-red-100">Error</p>
              <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            </div>
          </div>
        </div>
      )}

      {/* Loading State */}
      {isLoading && documents.length === 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-12 text-center">
          <SpinnerIcon className="w-8 h-8 text-blue-600 dark:text-blue-400 animate-spin mx-auto mb-4" />
          <p className="text-gray-600 dark:text-gray-400">Loading documents...</p>
        </div>
      )}

      {/* Empty State */}
      {!isLoading && filteredDocuments.length === 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-12 text-center">
          <EmptyIcon className="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
            {searchQuery ? 'No documents found' : 'No documents yet'}
          </h3>
          <p className="text-gray-600 dark:text-gray-400 mb-6">
            {searchQuery
              ? 'Try adjusting your search query'
              : 'Upload your first document to get started'}
          </p>
          {!searchQuery && (
            <button
              onClick={() => setShowUpload(true)}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
            >
              Upload Document
            </button>
          )}
        </div>
      )}

      {/* Document Grid */}
      {!isLoading && filteredDocuments.length > 0 && (
        <>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Showing {filteredDocuments.length} of {totalCount} documents
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {filteredDocuments.map((document) => (
              <DocumentCard
                key={document.id}
                document={document}
                onClick={() => setSelectedDocument(document)}
                onDownload={() => handleDownload(document)}
                onDelete={() => handleDelete(document)}
                isDeleting={deletingId === document.id}
              />
            ))}
          </div>

          {/* Pagination */}
          {totalCount > limit && (
            <div className="flex items-center justify-between">
              <button
                onClick={() => setOffset(Math.max(0, offset - limit))}
                disabled={offset === 0}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Previous
              </button>

              <span className="text-sm text-gray-600 dark:text-gray-400">
                Page {Math.floor(offset / limit) + 1} of {Math.ceil(totalCount / limit)}
              </span>

              <button
                onClick={() => setOffset(offset + limit)}
                disabled={offset + limit >= totalCount}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}

      {/* Document Viewer Modal */}
      {selectedDocument && (
        <DocumentViewer
          document={selectedDocument}
          onClose={() => setSelectedDocument(null)}
          onDelete={() => handleDelete(selectedDocument)}
          isDeleting={deletingId === selectedDocument.id}
        />
      )}
    </div>
  );
};

// Utility function to format bytes
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';

  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));

  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
};

// Icons
const CloseIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M6 18L18 6M6 6l12 12"
    />
  </svg>
);

const UploadIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
    />
  </svg>
);

const DocumentIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
    />
  </svg>
);

const StorageIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4"
    />
  </svg>
);

const EncryptedIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
    />
  </svg>
);

const SearchIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
    />
  </svg>
);

const RefreshIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
    />
  </svg>
);

const ErrorIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
    />
  </svg>
);

const SpinnerIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24">
    <circle
      className="opacity-25"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      strokeWidth="4"
    />
    <path
      className="opacity-75"
      fill="currentColor"
      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
    />
  </svg>
);

const EmptyIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M5 19a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1M5 19h14a2 2 0 002-2v-5a2 2 0 00-2-2H9a2 2 0 00-2 2v5a2 2 0 01-2 2z"
    />
  </svg>
);
