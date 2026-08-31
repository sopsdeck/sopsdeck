import { defineConfig } from '@playwright/test';

const driveURL = (process.env.SOPSDECK_DRIVE_URL ?? 'http://127.0.0.1:4174').replace(/\/$/, '');
const driveListen = new URL(driveURL).host;
const driveBin = process.env.SOPSDECK_BIN;
const driveCmd = driveBin
  ? `"${driveBin}" drive --demo --listen ${driveListen} --ui desktop/src`
  : `go run ./cmd/sopsdeck drive --demo --listen ${driveListen} --ui desktop/src`;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  use: {
    baseURL: driveURL,
    trace: 'off',
  },
  webServer: {
    command: driveCmd,
    url: `${driveURL}/health`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
    env: {
      ...process.env,
      PATH: `/opt/homebrew/bin:/usr/local/bin:${process.env.PATH ?? ''}`,
      SOPSDECK_DEMO_USER: 'checkout',
      SOPSDECK_TEAM_ROOT: '',
      SOPSDECK_DEV_PROJECT: '',
      GIT_TERMINAL_PROMPT: '0',
      GIT_CONFIG_COUNT: '1',
      GIT_CONFIG_KEY_0: 'commit.gpgsign',
      GIT_CONFIG_VALUE_0: 'false',
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
