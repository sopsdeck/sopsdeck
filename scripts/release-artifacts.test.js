import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const yaml = readFileSync(join(root, '.github/workflows/release.yml'), 'utf8');
const html = readFileSync(join(root, 'site/index.html'), 'utf8');
const wrangler = readFileSync(join(root, 'wrangler.jsonc'), 'utf8');

test('release workflow attaches Windows and Linux CLI binaries', () => {
  expect(yaml).toContain('sopsdeck-windows-amd64.exe');
  expect(yaml).toContain('sopsdeck-linux-amd64');
  expect(yaml).toContain('sopsdeck-linux-arm64');
  expect(yaml).not.toContain('darwin');
});

test('landing download points at GitHub Releases for Windows and Linux', () => {
  expect(html).toContain('releases/latest/download/sopsdeck-windows-amd64.exe');
  expect(html).toContain('releases/latest/download/sopsdeck-linux-amd64');
  expect(html).toContain('walkthrough.webm');
  expect(html).toContain('./scripts/dev');
});

test('Wrangler serves site/ as static assets', () => {
  expect(wrangler).toContain('"directory": "./site"');
});
