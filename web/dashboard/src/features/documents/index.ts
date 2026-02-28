/**
 * Documents feature exports
 */

// Main components
export { Documents } from './Documents';
export { DocumentCard } from './DocumentCard';
export { DocumentUpload } from './DocumentUpload';
export { DocumentViewer } from './DocumentViewer';

// API
export { documentsApi } from './api';

// Types
export type {
  Document,
  UploadMetadata,
  DocumentListParams,
  DocumentListResponse,
  UploadDocumentRequest,
  DocumentPreview,
} from './types';

// Utilities
export {
  formatFileSize,
  formatDate,
  getDocumentIconType,
  isFileTypeSupported,
} from './types';

// Constants
export {
  FILE_SIZE_UNITS,
  MAX_UPLOAD_SIZE,
  SUPPORTED_FILE_TYPES,
  DOCUMENT_TYPE_ICONS,
} from './types';
