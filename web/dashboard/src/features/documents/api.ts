/**
 * API functions for document storage management
 * Communicates with storage-service endpoints
 */

import type {
  Document,
  DocumentListParams,
  DocumentListResponse,
  UploadDocumentRequest,
} from './types';

const API_BASE_URL = import.meta.env.VITE_STORAGE_API_URL || import.meta.env.VITE_API_URL || '/api';

const getAccessToken = (): string | null => {
  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored) as { accessToken: string };
    return tokens.accessToken;
  }
  return null;
};

const fetchWithAuth = async (
  url: string,
  options: RequestInit = {}
): Promise<Response> => {
  const token = getAccessToken();

  if (token) {
    options.headers = {
      ...options.headers,
      Authorization: `Bearer ${token}`,
    };
  }

  return fetch(url, options);
};

const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = (await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }))) as { code: string; message: string; details?: Record<string, unknown> };
    throw new Error(error.message || 'API request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

// Transform API response to Document type
const toDocument = (apiDoc: Record<string, unknown>): Document => ({
  id: apiDoc.document_id as string,
  name: apiDoc.name as string,
  contentType: apiDoc.content_type as string,
  size: apiDoc.size as number,
  checksum: apiDoc.checksum as string | undefined,
  userEmail: apiDoc.user_email as string | undefined,
  createdAt: apiDoc.created_at as string,
  expiresAt: (apiDoc.expires_at as string | null) ?? undefined,
  isEncrypted: false, // This would be determined by the API if encryption is used
});

// Documents API
export const documentsApi = {
  /**
   * List all documents with optional filtering
   * GET /documents
   */
  async list(params?: DocumentListParams): Promise<DocumentListResponse> {
    const queryParams = new URLSearchParams();

    if (params?.userEmail) {
      queryParams.append('user_email', params.userEmail);
    }
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const url = `${API_BASE_URL}/documents${queryParams.toString() ? `?${queryParams.toString()}` : ''}`;
    const response = await fetchWithAuth(url);
    const data = await handleResponse<{ documents: unknown[]; count: number }>(response);

    return {
      documents: data.documents.map((doc: unknown) => toDocument(doc as Record<string, unknown>)),
      count: data.count,
    };
  },

  /**
   * Get a single document by ID
   * GET /documents/{id}/metadata
   */
  async get(id: string): Promise<Document> {
    const response = await fetchWithAuth(`${API_BASE_URL}/documents/${id}/metadata`);
    const data = await handleResponse<Record<string, unknown>>(response);
    return toDocument(data);
  },

  /**
   * Upload a new document
   * POST /documents
   */
  async upload(request: UploadDocumentRequest, onProgress?: (progress: number) => void): Promise<Document> {
    const formData = new FormData();
    formData.append('file', request.file);

    if (request.userEmail) {
      formData.append('user_email', request.userEmail);
    }

    // Use XMLHttpRequest for upload progress
    return new Promise((resolve, reject) => {
      const token = getAccessToken();
      const xhr = new XMLHttpRequest();

      // Track upload progress
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable && onProgress) {
          const progress = Math.round((event.loaded / event.total) * 100);
          onProgress(progress);
        }
      });

      // Handle completion
      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            const data = JSON.parse(xhr.responseText);
            resolve(toDocument(data));
          } catch {
            reject(new Error('Failed to parse response'));
          }
        } else {
          try {
            const error = JSON.parse(xhr.responseText);
            reject(new Error(error.message || 'Upload failed'));
          } catch {
            reject(new Error(`Upload failed with status ${xhr.status}`));
          }
        }
      });

      // Handle errors
      xhr.addEventListener('error', () => {
        reject(new Error('Network error during upload'));
      });

      xhr.addEventListener('abort', () => {
        reject(new Error('Upload was cancelled'));
      });

      // Open and send request
      xhr.open('POST', `${API_BASE_URL}/documents`);

      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      }

      xhr.send(formData);
    });
  },

  /**
   * Download a document file
   * GET /documents/{id}
   */
  async download(id: string, _filename?: string): Promise<Blob> {
    const token = getAccessToken();
    const headers: HeadersInit = {};

    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE_URL}/documents/${id}`, { headers });

    if (!response.ok) {
      const error = (await response.json().catch(() => ({
        code: 'unknown_error',
        message: 'An unknown error occurred',
      }))) as { code: string; message: string };
      throw new Error(error.message || 'Download failed');
    }

    return response.blob();
  },

  /**
   * Get download URL for a document (useful for direct links)
   */
  getDownloadUrl(id: string): string {
    return `${API_BASE_URL}/documents/${id}`;
  },

  /**
   * Download a document and trigger browser download
   */
  async downloadAndSave(id: string, filename: string): Promise<void> {
    const blob = await this.download(id);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  },

  /**
   * Delete a document
   * DELETE /documents/{id}
   */
  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/documents/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Search documents by name
   */
  async search(query: string, params?: Omit<DocumentListParams, 'search'>): Promise<Document[]> {
    const result = await this.list(params);
    const searchLower = query.toLowerCase();

    return result.documents.filter((doc) =>
      doc.name.toLowerCase().includes(searchLower)
    );
  },
};

export default documentsApi;
