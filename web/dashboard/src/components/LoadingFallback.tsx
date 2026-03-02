interface LoadingFallbackProps {
  message?: string;
  size?: 'sm' | 'md' | 'lg';
}

export function LoadingFallback({ message = 'Loading...', size = 'md' }: LoadingFallbackProps) {
  const sizeClasses = {
    sm: 'w-8 h-8 border-2',
    md: 'w-12 h-12 border-4',
    lg: 'w-16 h-16 border-4',
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="flex flex-col items-center gap-4">
        <div
          className={`${sizeClasses[size]} border-blue-600 border-t-transparent rounded-full animate-spin`}
          data-testid="loading-spinner"
        />
        {message && (
          <p className="text-gray-600 dark:text-gray-400 text-sm" data-testid="loading-message">
            {message}
          </p>
        )}
      </div>
    </div>
  );
}

export function PageLoadingFallback({ message = 'Loading page...' }: Omit<LoadingFallbackProps, 'size'>) {
  return (
    <div className="flex items-center justify-center p-8" data-testid="page-loading">
      <div className="flex flex-col items-center gap-3">
        <div className="w-8 h-8 border-3 border-blue-600 border-t-transparent rounded-full animate-spin" />
        <p className="text-gray-500 dark:text-gray-400 text-sm">{message}</p>
      </div>
    </div>
  );
}

export function InlineLoadingFallback({ message }: Omit<LoadingFallbackProps, 'size'>) {
  return (
    <div className="flex items-center justify-center py-4" data-testid="inline-loading">
      <div className="flex items-center gap-2">
        <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
        {message && <span className="text-gray-500 text-sm">{message}</span>}
      </div>
    </div>
  );
}
