import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const yaml = readFileSync(join(root, '.github/workflows/release.yml'), 'utf8');
const landing = readFileSync(join(root, 'site/src/pages/index.astro'), 'utf8');
const astroConfig = readFileSync(join(root, 'site/astro.config.mjs'), 'utf8');
const wrangler = readFileSync(join(root, 'wrangler.jsonc'), 'utf8');

test('Playwright cleanup is isolated from the persistent team studio', () => {
  const config = readFileSync(join(root, 'playwright.config.js'), 'utf8');
  expect(config).toContain("outputDir: './test-results/playwright'");
});

test('homepage navigation links to the GitHub repository', () => {
  const layout = readFileSync(join(root, 'site/src/layouts/SiteLayout.astro'), 'utf8');
  expect(layout).toContain('href="https://github.com/sopsdeck/sopsdeck"');
});

test('release workflow attaches native CLI binaries', () => {
  expect(yaml).toContain('sopsdeck-darwin-amd64');
  expect(yaml).toContain('sopsdeck-darwin-arm64');
  expect(yaml).toContain('sopsdeck-windows-amd64.exe');
  expect(yaml).toContain('sopsdeck-linux-amd64');
  expect(yaml).toContain('sopsdeck-linux-arm64');
});

test('release workflow publishes the npm launcher when configured', () => {
  expect(yaml).toContain('secrets.NPM_TOKEN');
  expect(yaml).toContain('npm publish --access public');
});

test('npm package exposes browser launcher aliases', () => {
  const packageJson = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'));
  expect(packageJson.private).toBeUndefined();
  expect(packageJson.bin.sopsdeck).toBe('./bin/sopsdeck.mjs');
  expect(packageJson.bin.sd).toBe('./bin/sopsdeck.mjs');
  expect(packageJson.files).toContain('desktop/src');
});

test('landing page documents the npm browser install', () => {
  expect(landing).toContain('/assets/landing-editor.jpg');
  expect(landing).toContain('npm install -D @sopsdeck/sopsdeck');
  expect(landing).toContain('npx sopsdeck .');
  expect(landing).toContain('/docs/guide.html#rename-keys');
});

test('site uses the Astro Cloudflare adapter', () => {
  expect(astroConfig).toContain("import cloudflare from '@astrojs/cloudflare'");
  expect(astroConfig).toContain('adapter: cloudflare()');
});

test('Wrangler serves the Astro Worker and assets', () => {
  expect(wrangler).toContain('"main": "./site/dist/server/entry.mjs"');
  expect(wrangler).toContain('"directory": "./site/dist/client"');
});
