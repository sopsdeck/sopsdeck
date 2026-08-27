import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  timeout: 60_000,
  use: {
    baseURL: process.env.SOPSDECK_DRIVE_URL ?? 'http://127.0.0.1:4174',
    trace: 'off',
  },
  webServer: {
    command: 'go run ./cmd/sopsdeck drive --demo --listen 127.0.0.1:4174 --ui desktop/src',
    url: 'http://127.0.0.1:4174/health',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
  projects: [
    { name: 'smoke', testMatch: 'smoke.spec.js' },
    { name: 'demo', testMatch: 'demo.spec.js' },
  ],
});
