/**
 * DocumentUpload Component - Drag-drop file upload with progress tracking
 */

import { useState, useCallback, useRef } from 'react';
import type { UploadMetadata } from './types';
import { isFileTypeSupported, formatFileSize, MAX_UPLOAD_SIZE } from './types';

interface DocumentUploadProps {
  onUpload?: (files: File[]) => void;
  maxFileSize?: number;
  multiple?: boolean;
  disabled?: boolean;
}

export const DocumentUpload = ({
  onUpload,
  maxFileSize = MAX_UPLOAD_SIZE,
  multiple = true,
  disabled = false,
}: DocumentUploadProps) => {
  const [isDragging, setIsDragging] = useState(false);
  const [uploads, setUploads] = useState<UploadMetadata[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!disabled) {
      setIsDragging(true);
    }
  }, [disabled]);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    if (disabled) return;

    const droppedFiles = Array.from(e.dataTransfer.files);
    processFiles(droppedFiles);
  }, [disabled]);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    if (disabled || !e.target.files) return;

    const selectedFiles = Array.from(e.target.files);
    processFiles(selectedFiles);

    // Reset input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [disabled]);

  const processFiles = (files: File[]) => {
    const validFiles = files.filter(file => {
      // Check file size
      if (file.size > maxFileSize) {
        setUploads(prev => [...prev, {
          file,
          progress: 0,
          status: 'error',
          error: `File size exceeds ${formatFileSize(maxFileSize)} limit`,
        }]);
        return false;
      }

      // Check file type (optional warning, not blocking)
      if (!isFileTypeSupported(file)) {
        console.warn(`File type ${file.type} may not be supported`);
      }

      return true;
    });

    if (validFiles.length > 0) {
      const newUploads: UploadMetadata[] = validFiles.map(file => ({
        file,
        progress: 0,
        status: 'pending',
      }));

      setUploads(prev => [...prev, ...newUploads]);
      onUpload?.(validFiles);
    }
  };

  const updateUpload = (index: number, updates: Partial<UploadMetadata>) => {
    setUploads(prev => {
      const newUploads = [...prev];
      newUploads[index] = { ...newUploads[index], ...updates };
      return newUploads;
    });
  };

  const removeUpload = (index: number) => {
    setUploads(prev => prev.filter((_, i) => i !== index));
  };

  const triggerFileSelect = () => {
    fileInputRef.current?.click();
  };

  const clearUploads = () => setUploads([]);

  return (
    <div className="space-y-4">
      {/* Upload zone */}
      <div
        className={`
          relative border-2 border-dashed rounded-lg p-8 text-center transition-colors
          ${isDragging
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
          }
          ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
        `}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={disabled ? undefined : triggerFileSelect}
      >
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleFileSelect}
          multiple={multiple}
          accept=".pdf,.jpg,.jpeg,.png,.gif,.webp,.txt,.doc,.docx,.xls,.xlsx"
          disabled={disabled}
        />

        <UploadIcon className="mx-auto w-12 h-12 text-gray-400 dark:text-gray-500 mb-4" />

        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          Upload Documents
        </h3>

        <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
          {multiple
            ? 'Drag and drop files here, or click to select'
            : 'Drag and drop a file here, or click to select'}
        </p>

        <p className="text-xs text-gray-500 dark:text-gray-500">
          Supported: PDF, Images, Word, Excel, Text (Max {formatFileSize(maxFileSize)})
        </p>

        <button
          type="button"
          className="mt-4 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          disabled={disabled}
          onClick={(e) => {
            e.stopPropagation();
            triggerFileSelect();
          }}
        >
          Browse Files
        </button>
      </div>

      {/* Upload progress list */}
      {uploads.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100">
            Uploads ({uploads.length})
          </h4>

          <div className="space-y-2 max-h-64 overflow-y-auto">
            {uploads.map((upload, index) => (
              <UploadItem
                key={`${upload.file.name}-${index}`}
                upload={upload}
                onRemove={() => removeUpload(index)}
                onUpdate={(updates) => updateUpload(index, updates)}
              />
            ))}
          </div>

          {/* Clear all button */}
          <button
            type="button"
            className="mt-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
            onClick={clearUploads}
          >
            Clear all
          </button>
        </div>
      )}
    </div>
  );
};

interface UploadItemProps {
  upload: UploadMetadata;
  onRemove: () => void;
  onUpdate: (updates: Partial<UploadMetadata>) => void;
}

const UploadItem = ({ upload, onRemove }: UploadItemProps) => {
  const { file, progress, status, error } = upload;

  return (
    <div className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
      {/* File icon */}
      <div className="w-10 h-10 rounded flex items-center justify-center bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 flex-shrink-0">
        <FileIcon />
      </div>

      {/* File info and progress */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
            {file.name}
          </p>
          <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0 ml-2">
            {formatFileSize(file.size)}
          </span>
        </div>

        {/* Progress bar or status */}
        {status === 'uploading' && (
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
            <div
              className="bg-blue-600 h-1.5 rounded-full transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        )}

        {status === 'error' && (
          <p className="text-xs text-red-600 dark:text-red-400">{error}</p>
        )}

        {status === 'success' && (
          <p className="text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
            <SuccessIcon className="w-3 h-3" />
            Uploaded successfully
          </p>
        )}
      </div>

      {/* Remove button */}
      <button
        type="button"
        onClick={onRemove}
        className="p-1 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        title="Remove"
      >
        <CloseIcon className="w-4 h-4" />
      </button>
    </div>
  );
};

// Icons
const UploadIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
    />
  </svg>
);

const FileIcon = () => (
  <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" className="w-5 h-5">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
    />
  </svg>
);

const SuccessIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M5 13l4 4L19 7"
    />
  </svg>
);

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
