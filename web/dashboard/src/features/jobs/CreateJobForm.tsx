/**
 * CreateJobForm Component - Form for creating new print jobs with file upload
 */

import { useState, useRef, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { printersApi } from '@/services/api';
import { useCreateJob, useFileUpload } from './useJobs';
import type { CreateJobFormData, JobFormErrors } from '@/types/jobs';

interface CreateJobFormProps {
  onSuccess?: (jobId: string) => void;
  onCancel?: () => void;
  initialPrinterId?: string;
}

export const CreateJobForm = ({
  onSuccess,
  onCancel,
  initialPrinterId,
}: CreateJobFormProps) => {
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Fetch printers
  const { data: printers = [], isLoading: isLoadingPrinters } = useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
  });

  const createJobMutation = useCreateJob();
  const { uploadFile, progress, isUploading, error: uploadError } = useFileUpload();

  // Form state
  const [formData, setFormData] = useState<CreateJobFormData>({
    printerId: initialPrinterId || '',
    documentName: '',
    file: undefined,
    color: false,
    duplex: false,
    paperSize: 'A4',
    copies: 1,
    quality: 'standard',
    orientation: 'portrait',
  });

  const [errors, setErrors] = useState<JobFormErrors>({});
  const [touched, setTouched] = useState<Set<string>>(new Set());

  // Get available online printers
  const onlinePrinters = printers.filter((p) => p.isOnline && p.isActive);

  // Get selected printer capabilities
  const selectedPrinter = printers.find((p) => p.id === formData.printerId);

  // Update form defaults based on printer capabilities
  useEffect(() => {
    if (selectedPrinter?.capabilities) {
      const caps = selectedPrinter.capabilities;
      setFormData((prev) => ({
        ...prev,
        color: caps.supportsColor ? prev.color : false,
        duplex: caps.supportsDuplex ? prev.duplex : false,
        paperSize: caps.supportedPaperSizes?.includes(prev.paperSize || 'A4')
          ? prev.paperSize
          : caps.supportedPaperSizes?.[0] || 'A4',
      }));
    }
  }, [selectedPrinter]);

  const validateForm = (): boolean => {
    const newErrors: JobFormErrors = {};

    if (!formData.printerId) {
      newErrors.printerId = 'Please select a printer';
    }

    if (!formData.documentName.trim()) {
      newErrors.documentName = 'Please enter a document name';
    }

    if (!formData.file) {
      newErrors.file = 'Please select a file to print';
    } else {
      // Check file size (max 50MB)
      const MAX_FILE_SIZE = 50 * 1024 * 1024;
      if (formData.file.size > MAX_FILE_SIZE) {
        newErrors.file = 'File size must be less than 50MB';
      }

      // Check file type
      const allowedTypes = [
        'application/pdf',
        'application/msword',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'image/jpeg',
        'image/png',
      ];
      if (!allowedTypes.includes(formData.file.type)) {
        newErrors.file = 'Unsupported file type. Please use PDF, DOC, DOCX, JPG, or PNG.';
      }
    }

    if (formData.copies && formData.copies < 1) {
      newErrors.settings = 'Copies must be at least 1';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleFieldChange = <K extends keyof CreateJobFormData>(
    field: K,
    value: CreateJobFormData[K]
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error for this field
    if (errors[field as keyof JobFormErrors]) {
      setErrors((prev) => ({ ...prev, [field as keyof JobFormErrors]: undefined }));
    }
    setTouched((prev) => new Set([...prev, field]));
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setFormData((prev) => ({
        ...prev,
        file,
        documentName: prev.documentName || file.name.replace(/\.[^/.]+$/, ''),
      }));
      setErrors((prev) => ({ ...prev, file: undefined }));
    }
  };

  const handleRemoveFile = () => {
    setFormData((prev) => ({ ...prev, file: undefined }));
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    if (!formData.file) {
      return;
    }

    try {
      // Upload file and convert to base64
      const base64Data = await uploadFile(formData.file);

      // Create job
      const result = await createJobMutation.mutateAsync({
        printerId: formData.printerId,
        documentName: formData.documentName.trim(),
        fileData: base64Data,
        fileName: formData.file.name,
        fileSize: formData.file.size,
        settings: {
          color: formData.color,
          duplex: formData.duplex,
          paperSize: formData.paperSize,
          copies: formData.copies,
          quality: formData.quality,
          orientation: formData.orientation,
        },
      });

      onSuccess?.(result.id);
    } catch (err) {
      console.error('Failed to create job:', err);
    }
  };

  const isLoading = isUploading || createJobMutation.isPending || isLoadingPrinters;

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* File Upload */}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Document File <span className="text-red-500">*</span>
        </label>
        {!formData.file ? (
          <div
            onClick={() => fileInputRef.current?.click()}
            className={`
              border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors
              ${touched.has('file') && errors.file
                ? 'border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-900/10'
                : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
              }
            `}
          >
            <input
              ref={fileInputRef}
              type="file"
              onChange={handleFileChange}
              accept=".pdf,.doc,.docx,.jpg,.jpeg,.png"
              className="hidden"
              disabled={isLoading}
            />
            <FileIcon className="mx-auto h-12 w-12 text-gray-400" />
            <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
              <span className="font-medium text-blue-600 dark:text-blue-400">
                Click to upload
              </span>{' '}
              or drag and drop
            </p>
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-500">
              PDF, DOC, DOCX, JPG, PNG up to 50MB
            </p>
          </div>
        ) : (
          <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                  <FileIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {formData.file.name}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {formatFileSize(formData.file.size)}
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={handleRemoveFile}
                disabled={isLoading}
                className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors disabled:opacity-50"
              >
                <XIcon className="w-5 h-5" />
              </button>
            </div>
            {/* Upload Progress */}
            {(isUploading || uploadError) && (
              <div className="mt-3">
                {isUploading && (
                  <div className="flex items-center gap-2">
                    <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div
                        className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                        style={{ width: `${progress}%` }}
                      />
                    </div>
                    <span className="text-xs text-gray-500 dark:text-gray-400">{progress}%</span>
                  </div>
                )}
                {uploadError && (
                  <p className="text-sm text-red-600 dark:text-red-400">{uploadError}</p>
                )}
              </div>
            )}
          </div>
        )}
        {errors.file && (
          <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.file}</p>
        )}
      </div>

      {/* Document Name */}
      <div>
        <label htmlFor="documentName" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Document Name <span className="text-red-500">*</span>
        </label>
        <input
          type="text"
          id="documentName"
          value={formData.documentName}
          onChange={(e) => handleFieldChange('documentName', e.target.value)}
          onBlur={() => setTouched((prev) => new Set([...prev, 'documentName']))}
          placeholder="My Document"
          disabled={isLoading}
          className={`
            w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100
            ${errors.documentName ? 'border-red-300 dark:border-red-700' : 'border-gray-300 dark:border-gray-600'}
          `}
        />
        {errors.documentName && (
          <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.documentName}</p>
        )}
      </div>

      {/* Printer Selection */}
      <div>
        <label htmlFor="printerId" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Select Printer <span className="text-red-500">*</span>
        </label>
        {onlinePrinters.length === 0 && !isLoadingPrinters ? (
          <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
            <p className="text-sm text-yellow-800 dark:text-yellow-300">
              No online printers available. Please connect a printer to continue.
            </p>
          </div>
        ) : (
          <select
            id="printerId"
            value={formData.printerId}
            onChange={(e) => handleFieldChange('printerId', e.target.value)}
            disabled={isLoading}
            className={`
              w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100
              ${errors.printerId ? 'border-red-300 dark:border-red-700' : 'border-gray-300 dark:border-gray-600'}
            `}
          >
            <option value="">Select a printer...</option>
            {onlinePrinters.map((printer) => (
              <option key={printer.id} value={printer.id}>
                {printer.name}
                {printer.type && ` (${printer.type})`}
              </option>
            ))}
          </select>
        )}
        {errors.printerId && (
          <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.printerId}</p>
        )}
      </div>

      {/* Print Settings */}
      {selectedPrinter && (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-4">
            Print Settings
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Color Mode */}
            {selectedPrinter.capabilities?.supportsColor && (
              <div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.color}
                    onChange={(e) => handleFieldChange('color', e.target.checked)}
                    disabled={isLoading}
                    className="w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                  />
                  <span className="text-sm text-gray-700 dark:text-gray-300">Color Printing</span>
                </label>
              </div>
            )}

            {/* Duplex */}
            {selectedPrinter.capabilities?.supportsDuplex && (
              <div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.duplex}
                    onChange={(e) => handleFieldChange('duplex', e.target.checked)}
                    disabled={isLoading}
                    className="w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                  />
                  <span className="text-sm text-gray-700 dark:text-gray-300">Double-sided</span>
                </label>
              </div>
            )}

            {/* Paper Size */}
            {selectedPrinter.capabilities?.supportedPaperSizes && (
              <div>
                <label htmlFor="paperSize" className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                  Paper Size
                </label>
                <select
                  id="paperSize"
                  value={formData.paperSize}
                  onChange={(e) => handleFieldChange('paperSize', e.target.value)}
                  disabled={isLoading}
                  className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-100"
                >
                  {selectedPrinter.capabilities.supportedPaperSizes.map((size) => (
                    <option key={size} value={size}>
                      {size}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* Copies */}
            <div>
              <label htmlFor="copies" className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                Copies
              </label>
              <input
                type="number"
                id="copies"
                min="1"
                max="100"
                value={formData.copies}
                onChange={(e) => handleFieldChange('copies', parseInt(e.target.value) || 1)}
                disabled={isLoading}
                className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-100"
              />
            </div>

            {/* Quality */}
            <div>
              <label htmlFor="quality" className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                Quality
              </label>
              <select
                id="quality"
                value={formData.quality}
                onChange={(e) => handleFieldChange('quality', e.target.value)}
                disabled={isLoading}
                className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-100"
              >
                <option value="draft">Draft</option>
                <option value="standard">Standard</option>
                <option value="high">High</option>
              </select>
            </div>

            {/* Orientation */}
            <div>
              <label htmlFor="orientation" className="block text-sm text-gray-600 dark:text-gray-400 mb-1">
                Orientation
              </label>
              <select
                id="orientation"
                value={formData.orientation}
                onChange={(e) => handleFieldChange('orientation', e.target.value as 'portrait' | 'landscape')}
                disabled={isLoading}
                className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-100"
              >
                <option value="portrait">Portrait</option>
                <option value="landscape">Landscape</option>
              </select>
            </div>
          </div>

          {/* Cost/Environment estimate */}
          <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
            <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
              <InfoIcon className="w-4 h-4" />
              <span>
                Estimated pages: <strong>{formData.copies || 1}</strong> ×{' '}
                {formData.documentName ? '~1' : 'Unknown'} = <strong>~{formData.copies || 1}</strong> pages
              </span>
            </div>
          </div>
        </div>
      )}

      {/* General error */}
      {errors.settings && (
        <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-600 dark:text-red-400">{errors.settings}</p>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isLoading}
            className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
        )}
        <button
          type="submit"
          disabled={isLoading || onlinePrinters.length === 0}
          className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? (
            <>
              <SpinnerIcon className="w-4 h-4 animate-spin" />
              {isUploading ? 'Uploading...' : 'Creating Job...'}
            </>
          ) : (
            <>
              <PrintIcon className="w-4 h-4" />
              Print Document
            </>
          )}
        </button>
      </div>
    </form>
  );
};

// Helper functions
const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
};

// Icons
const FileIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
    />
  </svg>
);

const XIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const InfoIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const SpinnerIcon = ({ className }: { className: string }) => (
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

const PrintIcon = ({ className }: { className: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
    />
  </svg>
);

export default CreateJobForm;
