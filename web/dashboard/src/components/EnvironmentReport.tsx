import type { EnvironmentReport as EnvironmentReportType } from '@/types';

interface EnvironmentReportProps {
  report: EnvironmentReportType;
  isLoading?: boolean;
}

export const EnvironmentReport = ({ report, isLoading }: EnvironmentReportProps) => {
  if (isLoading) {
    return (
      <div className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 rounded-xl p-6 animate-pulse">
        <div className="h-6 bg-gray-300 dark:bg-gray-700 rounded w-1/3 mb-4" />
        <div className="grid grid-cols-3 gap-4">
          <div className="h-20 bg-gray-300 dark:bg-gray-700 rounded" />
          <div className="h-20 bg-gray-300 dark:bg-gray-700 rounded" />
          <div className="h-20 bg-gray-300 dark:bg-gray-700 rounded" />
        </div>
      </div>
    );
  }

  const formatNumber = (num: number): string => {
    return new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 }).format(num);
  };

  return (
    <div className="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 rounded-xl p-6">
      <div className="flex items-center gap-2 mb-4">
        <svg className="w-6 h-6 text-green-600 dark:text-green-400" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Environmental Impact
        </h2>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
              />
            </svg>
          }
          label="Pages Printed"
          value={formatNumber(report.pagesPrinted)}
          color="text-blue-600 dark:text-blue-400"
          bgColor="bg-blue-100 dark:bg-blue-900/30"
        />
        <MetricCard
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3.055 11H5a2 2 0 012 2v1a2 2 0 002 2 2 2 0 012 2v2.945M8 3.935V5.5A2.5 2.5 0 0010.5 8h.5a2 2 0 012 2 2 2 0 104 0 2 2 0 012-2h1.064M15 20.488V18a2 2 0 012-2h3.064M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
          }
          label="CO₂ Saved"
          value={`${formatNumber(report.co2Grams)}g`}
          color="text-green-600 dark:text-green-400"
          bgColor="bg-green-100 dark:bg-green-900/30"
        />
        <MetricCard
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
              />
            </svg>
          }
          label="Trees Saved"
          value={formatNumber(report.treesSaved)}
          color="text-emerald-600 dark:text-emerald-400"
          bgColor="bg-emerald-100 dark:bg-emerald-900/30"
        />
      </div>

      <div className="mt-4 p-3 bg-white/50 dark:bg-gray-900/50 rounded-lg">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          By using cloud printing, you've saved approximately{' '}
          <span className="font-semibold text-green-600 dark:text-green-400">
            {formatNumber(report.treesSaved)} trees
          </span>{' '}
          and avoided{' '}
          <span className="font-semibold text-green-600 dark:text-green-400">
            {formatNumber(report.co2Grams)}g
          </span>{' '}
          of CO₂ emissions this {report.period}.
        </p>
      </div>
    </div>
  );
};

interface MetricCardProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  color: string;
  bgColor: string;
}

const MetricCard = ({ icon, label, value, color, bgColor }: MetricCardProps) => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg p-4 flex items-center gap-4 shadow-sm">
      <div className={`${bgColor} ${color} p-3 rounded-lg`}>{icon}</div>
      <div>
        <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{value}</p>
        <p className="text-sm text-gray-500 dark:text-gray-400">{label}</p>
      </div>
    </div>
  );
};
