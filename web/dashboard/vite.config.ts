import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
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
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    tsconfig: './tsconfig.test.json',
  },
});
