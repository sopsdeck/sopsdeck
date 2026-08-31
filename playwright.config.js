import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  use: {
    baseURL: process.env.SOPSDECK_DRIVE_URL ?? 'http://127.0.0.1:4174',
    trace: 'off',
  },
  webServer: {
    command: 'go run ./cmd/sopsdeck drive --demo --listen 127.0.0.1:4174 --ui desktop/src',
    url: 'http://127.0.0.1:4174/health',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: {
      ...process.env,
      SOPSDECK_DEMO_USER: 'checkout',
    },
  },
  projects: [
    { name: 'smoke', testMatch: 'smoke.spec.js' },
    { name: 'chrome', testMatch: 'chrome.spec.js' },
    {
      name: 'demo',
      testMatch: 'demo.spec.js',
      timeout: 180_000,
      use: {
        launchOptions: { slowMo: 180 },
      },
    },
  ],
});
