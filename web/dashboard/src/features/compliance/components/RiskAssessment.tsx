/**
 * RiskAssessment Component
 * Displays risk assessment score and mitigation suggestions
 */

export interface RiskAssessmentProps {
  riskScore?: number | null;
  level?: 'low' | 'medium' | 'high' | null;
  mitigations?: string[];
  isLoading?: boolean;
  onRun?: () => void;
  isRunning?: boolean;
}

export const RiskAssessment = ({
  riskScore = null,
  level = null,
  mitigations = [],
  isLoading = false,
  onRun,
  isRunning = false,
}: RiskAssessmentProps) => {
  const getRiskLevel = (score: number): 'low' | 'medium' | 'high' => {
    if (score < 30) return 'low';
    if (score < 60) return 'medium';
    return 'high';
  };

  const displayLevel = level || (riskScore !== null ? getRiskLevel(riskScore) : null);

  const levelConfig = {
    low: {
      color: 'text-green-600 dark:text-green-400',
      bg: 'bg-green-100 dark:bg-green-900/30',
      barColor: 'bg-green-500',
      label: 'Low Risk',
      description: 'Good security posture',
    },
    medium: {
      color: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-100 dark:bg-amber-900/30',
      barColor: 'bg-amber-500',
      label: 'Medium Risk',
      description: 'Some improvements needed',
    },
    high: {
      color: 'text-red-600 dark:text-red-400',
      bg: 'bg-red-100 dark:bg-red-900/30',
      barColor: 'bg-red-500',
      label: 'High Risk',
      description: 'Immediate action required',
    },
  };

  return (
    <div
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      data-testid="risk-assessment-section"
    >
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Risk Assessment
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Evaluate current security risks and get mitigation suggestions
          </p>
        </div>
        {onRun && (
          <button
            onClick={onRun}
            disabled={isRunning}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
            data-testid="run-risk-assessment-button"
          >
            {isRunning ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
                Running...
              </>
            ) : (
              'Run Assessment'
            )}
          </button>
        )}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent" />
        </div>
      ) : riskScore !== null && displayLevel ? (
        <div className="space-y-6">
          {/* Risk Score Display */}
          <div className="flex items-center gap-6">
            <div className="text-center" data-testid="risk-score">
              <div
                className={`text-5xl font-bold ${
                  levelConfig[displayLevel].color
                }`}
              >
                {riskScore}
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                Risk Score
              </p>
            </div>
            <div className="flex-1">
              <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  className={`h-full ${levelConfig[displayLevel].barColor} transition-all duration-500`}
                  style={{ width: `${riskScore}%` }}
                  data-testid="risk-score-bar"
                />
              </div>
              <div className="flex justify-between mt-2">
                <span className="text-sm text-gray-500 dark:text-gray-400">0</span>
                <span
                  className={`text-sm font-medium ${
                    levelConfig[displayLevel].color
                  }`}
                >
                  {levelConfig[displayLevel].label}
                </span>
                <span className="text-sm text-gray-500 dark:text-gray-400">100</span>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                {levelConfig[displayLevel].description}
              </p>
            </div>
          </div>

          {/* Mitigation Suggestions */}
          {mitigations.length > 0 && (
            <div data-testid="risk-mitigation-list">
              <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">
                Recommended Actions
              </h3>
              <ul className="space-y-2">
                {mitigations.map((mitigation, index) => (
                  <li
                    key={index}
                    className="flex items-start gap-2 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
                  >
                    <svg
                      className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                      />
                    </svg>
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      {mitigation}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      ) : (
        <div
          className="text-center py-8"
          data-testid="risk-assessment-empty"
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
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
            />
          </svg>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Click "Run Assessment" to evaluate security risks
          </p>
        </div>
      )}
    </div>
  );
};

export default RiskAssessment;
