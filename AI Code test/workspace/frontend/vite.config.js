import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The API must be same-origin with the app (session cookie + sameOrigin check),
// so dev requests go through this proxy. changeOrigin stays false: the backend
// compares the Origin header with the Host header it receives.
const backend = process.env.VITE_BACKEND_URL || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: backend, changeOrigin: false },
      '/swagger': { target: backend, changeOrigin: false },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.js'],
    include: ['src/**/*.test.{js,jsx}'],
    css: { modules: { classNameStrategy: 'non-scoped' } },
  },
})
