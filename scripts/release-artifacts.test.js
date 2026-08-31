import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const yaml = readFileSync(join(root, '.github/workflows/release.yml'), 'utf8');
const html = readFileSync(join(root, 'site/index.html'), 'utf8');
const wrangler = readFileSync(join(root, 'wrangler.jsonc'), 'utf8');

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
  expect(html).toContain('walkthrough.webm');
  expect(html).toContain('npm install -D sopsdeck');
  expect(html).toContain('npx sopsdeck .');
});

test('Wrangler serves site/ as static assets', () => {
  expect(wrangler).toContain('"directory": "./site"');
});
