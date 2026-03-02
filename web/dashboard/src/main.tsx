import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import './index.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        // Retry on network errors or 5xx errors up to 3 times
        if (failureCount >= 3) return false;
        const err = error as { status?: number };
        return !err.status || err.status >= 500;
      },
      refetchOnWindowFocus: false,
      staleTime: 5000, // Consider data fresh for 5 seconds
      gcTime: 300000, // Keep unused data in cache for 5 minutes
    },
    mutations: {
      retry: 1,
    },
  },
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>
);
