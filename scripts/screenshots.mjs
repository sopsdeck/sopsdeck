#!/usr/bin/env bun

import { strict as assert } from 'node:assert';
import { execFileSync, spawn } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium, expect } from '@playwright/test';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const scratch = mkdtempSync(join(tmpdir(), 'sopsdeck-screenshots-'));
const assets = join(root, 'site/public/assets');
const bin = join(scratch, process.platform === 'win32' ? 'sopsdeck.exe' : 'sopsdeck');
const images = ['landing-editor.jpg', 'feature-unused.jpg', 'feature-rename.jpg'];
let server;
let serverClosed;
let browser;

try {
  execFileSync(process.execPath, ['node_modules/@playwright/test/cli.js', 'install', 'chromium'], {
    cwd: root,
    stdio: 'inherit',
  });
  execFileSync('go', ['build', '-o', bin, './cmd/sopsdeck'], { cwd: root, stdio: 'inherit' });
  server = spawn(bin, ['drive', '--demo', '--listen', '127.0.0.1:0', '--ui', 'desktop/src'], {
    cwd: root,
    stdio: ['ignore', 'pipe', 'inherit'],
    env: {
      ...process.env,
      TMPDIR: scratch,
      TMP: scratch,
      TEMP: scratch,
      SOPSDECK_TEAM_ROOT: '',
      SOPSDECK_DEV_PROJECT: '',
      SOPSDECK_DEMO_USER: 'checkout',
    },
  });
  serverClosed = new Promise((resolve) => server.once('close', resolve));
  const url = await new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error('Demo startup timed out')), 120_000);
    let output = '';
    server.stdout.on('data', (data) => {
      output += data;
      const address = /listening on (http:\/\/127\.0\.0\.1:\d+)/.exec(output)?.[1];
      if (address) {
        clearTimeout(timeout);
        resolve(address);
      }
    });
    server.once('error', (error) => {
      clearTimeout(timeout);
      reject(error);
    });
    server.once('close', (code) => {
      clearTimeout(timeout);
      reject(new Error(`Demo exited with code ${code}`));
    });
  });

  const demo = await (await fetch(`${url}/demo`)).json();
  // The server owns this throwaway Project; never reuse a running app or team studio.
  assert(
    demo.project.startsWith(`${scratch}${sep}`),
    'Demo Project is outside the temporary directory',
  );
  const path = join(demo.project, '.env.production');
  for (const [key, value] of [
    ['DATABASE_URL', 'postgres://demo@localhost/app'],
    ['LEGACY_API_TOKEN', 'demo_unused_token'],
  ]) {
    const response = await fetch(`${url}/invoke`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ cmd: 'set_managed_key', path, key, value }),
    });
    assert(response.ok, await response.text());
  }
  mkdirSync(join(demo.project, 'src'), { recursive: true });
  writeFileSync(
    join(demo.project, 'src/billing.js'),
    'const stripe = process.env.STRIPE_SECRET;\n',
  );
  writeFileSync(
    join(demo.project, 'src/database.js'),
    'const database = process.env.DATABASE_URL;\n',
  );

  browser = await chromium.launch();
  const page = await browser.newPage({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
    colorScheme: 'light',
    reducedMotion: 'reduce',
  });
  await page.goto(url);
  await page.getByTestId('dev-banner').evaluate((banner) => banner.remove());
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('key-name')).toHaveCount(3);
  const names = page.getByTestId('key-name');
  await expect(page.locator('.unused')).toHaveCount(1);
  await expect(
    page
      .getByTestId('key-row')
      .filter({ has: page.locator('.unused') })
      .getByTestId('key-name'),
  ).toHaveValue('LEGACY_API_TOKEN');
  assert.deepEqual(await names.evaluateAll((inputs) => inputs.map((input) => input.value).sort()), [
    'DATABASE_URL',
    'LEGACY_API_TOKEN',
    'STRIPE_SECRET',
  ]);
  for (const field of await page.getByTestId('key-value').all()) {
    await expect(field).toHaveValue(/^•+$/);
  }
  await page.getByTestId('inspector-toggle-project').click();
  await page.evaluate(() => document.fonts.ready);
  await page.mouse.move(0, 0);
  const options = { type: 'jpeg', quality: 90, animations: 'disabled', caret: 'hide' };
  await page.screenshot({
    ...options,
    path: join(scratch, images[0]),
    clip: { x: 0, y: 0, width: 1440, height: 550 },
  });
  const keys = await page.locator('.key-head').boundingBox();
  assert(keys, 'Key table is missing');
  await page.screenshot({
    ...options,
    path: join(scratch, images[1]),
    clip: { x: Math.floor(keys.x), y: Math.floor(keys.y), width: 390, height: 210 },
  });

  const stripeIndex = await names.evaluateAll((inputs) =>
    inputs.findIndex((input) => input.value === 'STRIPE_SECRET'),
  );
  assert(stripeIndex >= 0, 'STRIPE_SECRET is missing');
  const stripe = names.nth(stripeIndex);
  await stripe.fill('STRIPE_API_KEY');
  await page.getByTestId('save').click();
  const dialog = page.getByTestId('save-preview-dialog');
  await expect(dialog).toBeVisible();
  await expect(page.getByTestId('save-preview')).toHaveText(
    'Rename STRIPE_SECRET → STRIPE_API_KEY',
  );
  await dialog.getByRole('checkbox', { name: 'Rewrite references' }).check();
  await page.mouse.move(0, 0);
  await dialog.screenshot({ ...options, path: join(scratch, images[2]) });

  mkdirSync(assets, { recursive: true });
  for (const image of images) copyFileSync(join(scratch, image), join(assets, image));
  console.log(`Wrote ${images.join(', ')} to site/public/assets/`);
} finally {
  await browser?.close();
  server?.kill();
  await serverClosed;
  rmSync(scratch, { recursive: true, force: true });
}
