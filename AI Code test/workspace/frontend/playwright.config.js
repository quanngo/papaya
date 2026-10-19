import { defineConfig, devices } from '@playwright/test'

// E2E runs against a real stack. Start the backend serving the built app:
//   (cd workspace/frontend && npm run build)
//   (cd workspace/backend && go run .)          # server.static-dir: ../frontend/dist
// then: E2E_BASE_URL=http://localhost:8080 npm run e2e
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  retries: 0,
  use: {
    baseURL: process.env.E2E_BASE_URL,
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
