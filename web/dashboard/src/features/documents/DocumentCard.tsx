/**
 * DocumentCard Component - Displays a document preview with metadata
 */

import type { Document } from './types';
import { formatFileSize, formatDate, getDocumentIconType } from './types';

interface DocumentCardProps {
  document: Document;
  onClick?: () => void;
  onDownload?: () => void;
  onDelete?: () => void;
  isDeleting?: boolean;
}

export const DocumentCard = ({
  document,
  onClick,
  onDownload,
  onDelete,
  isDeleting = false,
}: DocumentCardProps) => {
  const iconType = getDocumentIconType(document.contentType);

  const handleDownload = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDownload?.();
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDelete?.();
  };

  return (
    <div
      className={`
        bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
        p-4 hover:shadow-md dark:hover:shadow-gray-900/50 transition-all
        ${onClick ? 'cursor-pointer' : ''}
      `}
      onClick={onClick}
    >
      {/* Header with icon and name */}
      <div className="flex items-start gap-3 mb-3">
        {/* Document icon */}
        <div className="w-12 h-12 rounded-lg flex items-center justify-center bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 flex-shrink-0">
          <DocumentIcon type={iconType} />
        </div>

        {/* Name and actions */}
        <div className="flex-1 min-w-0">
          <h3 className="font-medium text-gray-900 dark:text-gray-100 truncate" title={document.name}>
            {document.name}
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
            {formatFileSize(document.size)}
          </p>
        </div>

        {/* Actions */}
        <div
          className="flex items-center gap-1 flex-shrink-0"
          onClick={(e) => e.stopPropagation()}
        >
          {onDownload && (
            <button
              onClick={handleDownload}
              className="p-1.5 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              title="Download document"
            >
              <DownloadIcon className="w-4 h-4" />
            </button>
          )}
          {onDelete && (
            <button
              onClick={handleDelete}
              disabled={isDeleting}
              className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Delete document"
            >
              {isDeleting ? (
                <SpinnerIcon className="w-4 h-4 animate-spin" />
              ) : (
                <TrashIcon className="w-4 h-4" />
              )}
            </button>
          )}
        </div>
      </div>

      {/* Document metadata */}
      <div className="space-y-2 text-sm">
        <div className="flex justify-between text-gray-600 dark:text-gray-400">
          <span>Type</span>
          <span className="text-gray-900 dark:text-gray-100 capitalize">
            {getContentTypeLabel(document.contentType)}
          </span>
        </div>
        <div className="flex justify-between text-gray-600 dark:text-gray-400">
          <span>Uploaded</span>
          <span className="text-gray-900 dark:text-gray-100">
            {formatDate(document.createdAt)}
          </span>
        </div>
        {document.checksum && (
          <div className="flex justify-between text-gray-600 dark:text-gray-400">
            <span>Checksum</span>
            <span className="text-gray-900 dark:text-gray-100 font-mono text-xs truncate max-w-[120px]" title={document.checksum}>
              {document.checksum.slice(0, 8)}...
            </span>
          </div>
        )}
      </div>

      {/* Footer with encryption status */}
      {document.isEncrypted !== undefined && (
        <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
          <div className="flex items-center gap-2 text-xs">
            {document.isEncrypted ? (
              <>
                <LockIcon className="w-3.5 h-3.5 text-green-600 dark:text-green-400" />
                <span className="text-green-600 dark:text-green-400">Encrypted</span>
              </>
            ) : (
              <>
                <UnlockIcon className="w-3.5 h-3.5 text-gray-400" />
                <span className="text-gray-500 dark:text-gray-400">Not encrypted</span>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

// Helper to get content type label
const getContentTypeLabel = (contentType: string): string => {
  const labels: Record<string, string> = {
    'application/pdf': 'PDF',
    'image/jpeg': 'JPEG Image',
    'image/png': 'PNG Image',
    'image/gif': 'GIF Image',
    'image/webp': 'WebP Image',
    'text/plain': 'Text',
    'application/msword': 'Word Document',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'Word Document',
    'application/vnd.ms-excel': 'Excel Spreadsheet',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'Excel Spreadsheet',
  };
  return labels[contentType] || contentType.split('/')[1] || 'Unknown';
};

// Icons
const DocumentIcon = ({ type }: { type: string }) => {
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
    case 'text':
      return (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-6 h-6">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
      );
    case 'doc':
      return (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-6 h-6">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
      );
    case 'sheet':
      return (
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-6 h-6">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
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

const LockIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
    />
  </svg>
);

const UnlockIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M8 11V7a4 4 0 118 0m-4 8v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2z"
    />
  </svg>
);
