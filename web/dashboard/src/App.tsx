import { Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { PageLoadingFallback } from './components/LoadingFallback';
import { ErrorBoundary } from './components/ErrorBoundary';
import { routes, notFoundComponent as NotFound } from './router/routes';
import { getGuard } from './router/guards';

const SuspenseWrapper = ({ children }: { children: React.ReactNode }) => (
  <Suspense fallback={<PageLoadingFallback />}>
    <ErrorBoundary>
      {children}
    </ErrorBoundary>
  </Suspense>
);

function App() {
  return (
    <ErrorBoundary>
      <Suspense fallback={<PageLoadingFallback />}>
        <Routes>
          {routes.map(({ path, auth, component: Component }) => {
            const Guard = getGuard(auth);
            return (
              <Route
                key={path}
                path={path}
                element={
                  <Guard>
                    <SuspenseWrapper>
                      <Component />
                    </SuspenseWrapper>
                  </Guard>
                }
              />
            );
          })}

          {/* Default redirect */}
          <Route path="/" element={<Navigate to="/dashboard" replace />} />

          {/* 404 - Not Found */}
          <Route
            path="*"
            element={
              <SuspenseWrapper>
                <NotFound />
              </SuspenseWrapper>
            }
          />
        </Routes>
      </Suspense>
    </ErrorBoundary>
  );
}

export default App;
