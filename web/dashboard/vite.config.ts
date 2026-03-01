import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig(({ mode }) => ({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api/v1/auth': {
        target: 'http://localhost:18001',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/v1\/auth/, '/auth'),
      },
      '/api/v1/printers': {
        target: 'http://localhost:8002',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/v1\/printers/, '/printers'),
      },
      '/api/v1/jobs': {
        target: 'http://localhost:8003',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/v1\/jobs/, '/jobs'),
      },
      '/api/v1/agents': {
        target: 'http://localhost:8002',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/v1\/agents/, '/agents'),
      },
      '/ws': {
        target: 'ws://localhost:8005',
        ws: true,
      },
    },
  },
  css: {
    postcss: './postcss.config.js',
    modules: {
      localsConvention: 'camelCase',
    },
  },
  build: {
    target: 'es2020',
    minify: 'esbuild',
    sourcemap: false,
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks: {
          // Core vendor chunks - using object syntax to avoid circular dependencies
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          'vendor-state': ['@tanstack/react-query', 'zustand'],
          'vendor-charts': ['recharts'],
          'vendor-utils': ['date-fns'],
        },
        chunkFileNames: 'assets/js/[name]-[hash].js',
        entryFileNames: 'assets/js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'assets/css/[name]-[hash][extname]';
          }
          return 'assets/[name]-[hash][extname]';
        },
      },
    },
    cssCodeSplit: true,
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    tsconfig: './tsconfig.test.json',
  },
  ...(mode === 'analyze' && {
    build: {
      rollupOptions: {
        output: {
          manualChunks: undefined,
        },
      },
    },
  }),
}));
