/**
 * DocumentViewer Component - Displays detailed document information with preview
 */

import { useState, useEffect } from 'react';
import type { Document as DocType } from './types';
import { formatFileSize, getDocumentIconType } from './types';
import { documentsApi } from './api';

interface DocumentViewerProps {
  document: DocType;
  onClose?: () => void;
  onDelete?: () => void;
  isDeleting?: boolean;
}

export const DocumentViewer = ({
  document: doc,
  onClose,
  onDelete,
  isDeleting = false,
}: DocumentViewerProps) => {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const iconType = getDocumentIconType(doc.contentType);
  const canPreview = ['pdf', 'image'].includes(iconType);

  useEffect(() => {
    // Create preview URL for supported types
    if (canPreview) {
      loadPreview();
    }

    return () => {
      // Cleanup object URL
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [doc.id, canPreview]);

  const loadPreview = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const blob = await documentsApi.download(doc.id);
      const url = URL.createObjectURL(blob);
      setPreviewUrl(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load preview');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDownload = async () => {
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

  const handleDelete = () => {
    onDelete?.();
  };

  const getContentTypeLabel = () => {
    const labels: Record<string, string> = {
      'application/pdf': 'PDF Document',
      'image/jpeg': 'JPEG Image',
      'image/png': 'PNG Image',
      'image/gif': 'GIF Image',
      'image/webp': 'WebP Image',
      'text/plain': 'Plain Text',
      'application/msword': 'Word Document (Legacy)',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'Word Document',
      'application/vnd.ms-excel': 'Excel Spreadsheet (Legacy)',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'Excel Spreadsheet',
    };
    return labels[doc.contentType] || doc.contentType;
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400">
              <DocumentTypeIcon type={iconType} />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 truncate max-w-md">
                {doc.name}
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {formatFileSize(doc.size)} · {getContentTypeLabel()}
              </p>
            </div>
          </div>

          <button
            onClick={onClose}
            className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          >
            <CloseIcon className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Preview */}
          {canPreview && (
            <div className="mb-6">
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Preview
              </h3>

              <div className="bg-gray-100 dark:bg-gray-900 rounded-lg p-4 min-h-[200px] flex items-center justify-center">
                {isLoading && (
                  <div className="text-center">
                    <SpinnerIcon className="w-8 h-8 text-blue-600 dark:text-blue-400 animate-spin mx-auto mb-2" />
                    <p className="text-sm text-gray-500 dark:text-gray-400">Loading preview...</p>
                  </div>
                )}

                {error && (
                  <div className="text-center">
                    <ErrorIcon className="w-8 h-8 text-red-500 mx-auto mb-2" />
                    <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                    <button
                      onClick={loadPreview}
                      className="mt-2 text-sm text-blue-600 dark:text-blue-400 hover:underline"
                    >
                      Retry
                    </button>
                  </div>
                )}

                {previewUrl && !isLoading && !error && (
                  <>
                    {iconType === 'pdf' ? (
                      <iframe
                        src={previewUrl}
                        className="w-full h-[500px] rounded border-0"
                        title={doc.name}
                      />
                    ) : (
                      <img
                        src={previewUrl}
                        alt={doc.name}
                        className="max-w-full max-h-[500px] rounded object-contain"
                      />
                    )}
                  </>
                )}

                {!canPreview && !isLoading && !error && (
                  <div className="text-center">
                    <NoPreviewIcon className="w-8 h-8 text-gray-400 mx-auto mb-2" />
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Preview not available for this file type
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Metadata */}
          <div>
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Document Details
            </h3>

            <div className="bg-gray-50 dark:bg-gray-900/50 rounded-lg">
              <MetadataRow label="File name" value={doc.name} />
              <MetadataRow label="File size" value={formatFileSize(doc.size)} />
              <MetadataRow label="Content type" value={doc.contentType} />
              <MetadataRow
                label="Upload date"
                value={new Date(doc.createdAt).toLocaleString()}
              />
              {doc.userEmail && (
                <MetadataRow label="Uploaded by" value={doc.userEmail} />
              )}
              {doc.checksum && (
                <MetadataRow
                  label="Checksum (SHA-256)"
                  value={doc.checksum}
                  monospace
                  truncate
                />
              )}
              {doc.expiresAt && (
                <MetadataRow
                  label="Expires"
                  value={new Date(doc.expiresAt).toLocaleString()}
                  highlight={new Date(doc.expiresAt) < new Date()}
                />
              )}
              {doc.isEncrypted !== undefined && (
                <MetadataRow
                  label="Encryption"
                  value={doc.isEncrypted ? 'Enabled' : 'Disabled'}
                  highlight={!doc.isEncrypted}
                />
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between p-4 border-t border-gray-200 dark:border-gray-700">
          <div className="text-sm text-gray-500 dark:text-gray-400">
            ID: <code className="text-xs font-mono bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">
              {doc.id}
            </code>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={handleDownload}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
            >
              <DownloadIcon className="w-4 h-4" />
              Download
            </button>

            {onDelete && (
              <button
                onClick={handleDelete}
                disabled={isDeleting}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeleting ? (
                  <SpinnerIcon className="w-4 h-4 animate-spin" />
                ) : (
                  <TrashIcon className="w-4 h-4" />
                )}
                Delete
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

interface MetadataRowProps {
  label: string;
  value: string;
  monospace?: boolean;
  truncate?: boolean;
  highlight?: boolean;
}

const MetadataRow = ({ label, value, monospace, truncate, highlight }: MetadataRowProps) => {
  return (
    <div className={`flex items-center justify-between py-3 px-4 border-b border-gray-200 dark:border-gray-700 last:border-b-0`}>
      <span className="text-sm text-gray-600 dark:text-gray-400">{label}</span>
      <span
        className={`
          text-sm text-gray-900 dark:text-gray-100
          ${monospace ? 'font-mono text-xs' : ''}
          ${truncate ? 'max-w-[200px] truncate' : ''}
          ${highlight ? 'text-yellow-600 dark:text-yellow-400' : ''}
        `}
        title={truncate ? value : undefined}
      >
        {value}
      </span>
    </div>
  );
};

// Icons
const DocumentTypeIcon = ({ type }: { type: string }) => {
  switch (type) {
    case 'pdf':
      return (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-6 h-6">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
          />
        </svg>
      );
    case 'image':
      return (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-6 h-6">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
      );
    default:
      return (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-6 h-6">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
          />
        </svg>
      );
  }
};

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

const DownloadIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
    />
  </svg>
);

const TrashIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
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

const NoPreviewIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
    />
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
    />
  </svg>
);
