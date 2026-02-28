/**
 * Document storage types for the OpenPrint Dashboard
 */

// Document type representing a stored document
export interface Document {
  id: string;
  name: string;
  contentType: string;
  size: number;
  checksum?: string;
  userEmail?: string;
  createdAt: string;
  expiresAt?: string | null;
  isEncrypted?: boolean; // UI flag for encryption status
}

// Upload metadata for tracking upload progress
export interface UploadMetadata {
  file: File;
  progress: number;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
  documentId?: string;
}

// Document list filtering and sorting
export interface DocumentListParams {
  userEmail?: string;
  limit?: number;
  offset?: number;
  search?: string;
}

// Document list response
export interface DocumentListResponse {
  documents: Document[];
  count: number;
}

// Upload request data
export interface UploadDocumentRequest {
  file: File;
  userEmail?: string;
}

// Document preview info
export interface DocumentPreview {
  type: 'pdf' | 'image' | 'text' | 'unknown';
  thumbnail?: string;
  pageCount?: number;
}

// File size formatting constants
export const FILE_SIZE_UNITS = ['B', 'KB', 'MB', 'GB'] as const;
export const MAX_UPLOAD_SIZE = 100 * 1024 * 1024; // 100MB default

// Supported file types for upload
export const SUPPORTED_FILE_TYPES = [
  'application/pdf',
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'text/plain',
  'application/msword',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/vnd.ms-excel',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
] as const;

// Map content types to icons
export const DOCUMENT_TYPE_ICONS: Record<string, string> = {
  'application/pdf': 'pdf',
  'image/jpeg': 'image',
  'image/png': 'image',
  'image/gif': 'image',
  'image/webp': 'image',
  'text/plain': 'text',
  'application/msword': 'doc',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'doc',
  'application/vnd.ms-excel': 'sheet',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'sheet',
};

// Get icon type for a document
export const getDocumentIconType = (contentType: string): string => {
  return DOCUMENT_TYPE_ICONS[contentType] || 'file';
};

// Format file size for display
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';

  let unitIndex = 0;
  let size = bytes;

  while (size >= 1024 && unitIndex < FILE_SIZE_UNITS.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(size < 10 ? 1 : 0)} ${FILE_SIZE_UNITS[unitIndex]}`;
};

// Format date for display
export const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
};

// Check if file type is supported
export const isFileTypeSupported = (file: File): boolean => {
  return SUPPORTED_FILE_TYPES.includes(file.type as any);
};
