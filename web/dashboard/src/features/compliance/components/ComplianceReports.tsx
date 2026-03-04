/**
 * ComplianceReports Component
 * Lists compliance reports with download and generation options
 */

import { useState } from 'react';
import type { ComplianceReport, ComplianceFramework } from '../types';

export interface ComplianceReportsProps {
  reports?: ComplianceReport[];
  isLoading?: boolean;
  error?: string | null;
  onGenerate?: (params: GenerateReportParams) => void;
  onDownload?: (reportId: string, format: 'pdf' | 'json') => void;
  onDelete?: (reportId: string) => void;
}

export interface GenerateReportParams {
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  format?: 'pdf' | 'json';
}

export const ComplianceReports = ({
  reports = [],
  isLoading = false,
  error = null,
  onGenerate,
  onDownload,
  onDelete,
}: ComplianceReportsProps) => {
  const [isGenerating, setIsGenerating] = useState(false);
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [generateParams, setGenerateParams] = useState<GenerateReportParams>({
    framework: 'fedramp',
    period_start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
      .toISOString()
      .split('T')[0],
    period_end: new Date().toISOString().split('T')[0],
    format: 'pdf',
  });

  const frameworkConfig: Record<
    ComplianceFramework,
    { name: string; color: string; icon: React.ReactNode }
  > = {
    fedramp: {
      name: 'FedRAMP',
      color: 'bg-blue-100 dark:bg-blue-900/30',
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z"
          />
        </svg>
      ),
    },
    hipaa: {
      name: 'HIPAA',
      color: 'bg-green-100 dark:bg-green-900/30',
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"
          />
        </svg>
      ),
    },
    gdpr: {
      name: 'GDPR',
      color: 'bg-purple-100 dark:bg-purple-900/30',
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
      ),
    },
    soc2: {
      name: 'SOC 2',
      color: 'bg-amber-100 dark:bg-amber-900/30',
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
          />
        </svg>
      ),
    },
    all: {
      name: 'All',
      color: 'bg-gray-100 dark:bg-gray-700',
      icon: (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
          />
        </svg>
      ),
    },
  };

  const statusConfig: Record<
    'complete' | 'generating' | 'failed' | 'pending',
    { label: string; className: string }
  > = {
    complete: {
      label: 'Complete',
      className: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
    },
    generating: {
      label: 'Generating',
      className: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400',
    },
    failed: {
      label: 'Failed',
      className: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
    },
    pending: {
      label: 'Pending',
      className: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400',
    },
  };

  // Map ComplianceStatus to display status
  const getDisplayStatus = (status: string): 'complete' | 'generating' | 'failed' | 'pending' => {
    if (status === 'compliant' || status === 'complete') return 'complete';
    if (status === 'non_compliant' || status === 'failed') return 'failed';
    if (status === 'in_progress' || status === 'generating') return 'generating';
    return 'pending';
  };

  const handleGenerate = async () => {
    if (!onGenerate) return;
    setIsGenerating(true);
    try {
      await onGenerate(generateParams);
      setShowGenerateModal(false);
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <div
      className="space-y-4"
      data-testid="compliance-reports-section"
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Compliance Reports
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Generate and download compliance reports for various frameworks
          </p>
        </div>
        {onGenerate && (
          <button
            onClick={() => setShowGenerateModal(true)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
            data-testid="generate-report-button"
          >
            <PlusIcon className="w-4 h-4" />
            Generate Report
          </button>
        )}
      </div>

      {/* Error State */}
      {error && (
        <div
          className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4"
          data-testid="reports-error"
        >
          <p className="text-sm text-red-800 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Loading State */}
      {isLoading ? (
        <div className="space-y-3">
          {[...Array(3)].map((_, i) => (
            <div
              key={i}
              className="bg-white dark:bg-gray-800 rounded-lg p-4 animate-pulse"
            >
              <div className="flex items-center gap-4">
                <div className="h-10 w-10 bg-gray-200 dark:bg-gray-700 rounded-lg" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
                  <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/2" />
                </div>
                <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-16" />
              </div>
            </div>
          ))}
        </div>
      ) : reports.length === 0 ? (
        <div
          className="bg-white dark:bg-gray-800 rounded-xl p-12 text-center border border-gray-200 dark:border-gray-700"
          data-testid="reports-empty"
        >
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            No compliance reports
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Generate your first compliance report to get started
          </p>
        </div>
      ) : (
        /* Reports List */
        <div
          className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700"
          data-testid="reports-list"
        >
          {reports.map((report) => (
            <div
              key={report.id}
              className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              data-report-id={report.id}
              data-testid="report-item"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div
                    className={`p-2 rounded-lg ${frameworkConfig[report.framework]?.color}`}
                  >
                    {frameworkConfig[report.framework]?.icon}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900 dark:text-gray-100">
                      {frameworkConfig[report.framework]?.name} Report
                    </p>
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      {new Date(report.generated_at).toLocaleDateString()} •{' '}
                      {new Date(report.period_start).toLocaleDateString()} -{' '}
                      {new Date(report.period_end).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span
                    className={`text-xs px-2 py-1 rounded-full font-medium ${
                      statusConfig[getDisplayStatus(report.overall_status)]?.className || ''
                    }`}
                  >
                    {statusConfig[getDisplayStatus(report.overall_status)]?.label || report.overall_status}
                  </span>
                  {onDownload && getDisplayStatus(report.overall_status) === 'complete' && (
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => onDownload(report.id, 'pdf')}
                        className="p-2 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                        title="Download PDF"
                        data-testid={`download-${report.id}-pdf`}
                      >
                        <DownloadIcon className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => onDownload(report.id, 'json')}
                        className="p-2 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                        title="Download JSON"
                        data-testid={`download-${report.id}-json`}
                      >
                        <CodeIcon className="w-4 h-4" />
                      </button>
                    </div>
                  )}
                  {onDelete && (
                    <button
                      onClick={() => onDelete(report.id)}
                      className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                      title="Delete report"
                      data-testid={`delete-${report.id}`}
                    >
                      <TrashIcon className="w-4 h-4" />
                    </button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Generate Report Modal */}
      {showGenerateModal && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
          data-testid="generate-report-modal"
        >
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
              Generate Compliance Report
            </h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Framework
                </label>
                <select
                  value={generateParams.framework}
                  onChange={(e) =>
                    setGenerateParams({ ...generateParams, framework: e.target.value as ComplianceFramework })
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <option value="fedramp">FedRAMP</option>
                  <option value="hipaa">HIPAA</option>
                  <option value="gdpr">GDPR</option>
                  <option value="soc2">SOC 2</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Period Start
                </label>
                <input
                  type="date"
                  value={generateParams.period_start}
                  onChange={(e) =>
                    setGenerateParams({ ...generateParams, period_start: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Period End
                </label>
                <input
                  type="date"
                  value={generateParams.period_end}
                  onChange={(e) =>
                    setGenerateParams({ ...generateParams, period_end: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Format
                </label>
                <select
                  value={generateParams.format}
                  onChange={(e) =>
                    setGenerateParams({
                      ...generateParams,
                      format: e.target.value as 'pdf' | 'json',
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <option value="pdf">PDF</option>
                  <option value="json">JSON</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowGenerateModal(false)}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleGenerate}
                disabled={isGenerating}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg font-medium transition-colors"
              >
                {isGenerating ? 'Generating...' : 'Generate'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const PlusIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
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

const CodeIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
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

export default ComplianceReports;
