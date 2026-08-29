#!/usr/bin/env bun

import { copyFileSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { spawnSync } from 'node:child_process';

import { chromium } from '@playwright/test';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const brand = join(root, 'brand');
const svgDir = join(brand, 'svg');
const templates = join(brand, 'templates');
const outDir = join(brand, 'export');

mkdirSync(outDir, { recursive: true });

function pngToIco(pngs) {
  const header = 6 + 16 * pngs.length;
  let offset = header;
  const total = header + pngs.reduce((sum, png) => sum + png.length, 0);
  const buf = Buffer.alloc(total);
  buf.writeUInt16LE(0, 0);
  buf.writeUInt16LE(1, 2);
  buf.writeUInt16LE(pngs.length, 4);
  let entry = 6;
  const bodies = [];
  for (const png of pngs) {
    const size = png.length;
    const ihdr = png.indexOf(Buffer.from('IHDR'));
    const width = ihdr === -1 ? 32 : png.readUInt32BE(ihdr + 4);
    const height = ihdr === -1 ? 32 : png.readUInt32BE(ihdr + 8);
    buf.writeUInt8(width >= 256 ? 0 : width, entry);
    buf.writeUInt8(height >= 256 ? 0 : height, entry + 1);
    buf.writeUInt8(0, entry + 2);
    buf.writeUInt8(0, entry + 3);
    buf.writeUInt16LE(1, entry + 4);
    buf.writeUInt16LE(32, entry + 6);
    buf.writeUInt32LE(size, entry + 8);
    buf.writeUInt32LE(offset, entry + 12);
    bodies.push({ offset, png });
    offset += size;
    entry += 16;
  }

  for (const body of bodies) {
    body.png.copy(buf, body.offset);
  }

  return buf;
}

async function waitForFonts(page) {
  await page.evaluate(async () => {
    await document.fonts.ready;
    await new Promise((resolve) => {
      setTimeout(resolve, 150);
    });
  });
}

async function rasterizeSvg(page, svgPath, dest, width, height, transparent = true) {
  const svg = readFileSync(svgPath, 'utf8');
  await page.setViewportSize({ width, height });
  await page.setContent(
    `<!doctype html><html><head><style>
      html,body{margin:0;width:${width}px;height:${height}px;background:${transparent ? 'transparent' : '#101828'};}
      svg{display:block;width:${width}px;height:${height}px;}
    </style></head><body>${svg}</body></html>`,
    { waitUntil: 'load' },
  );
  await page.screenshot({ path: dest, omitBackground: transparent });
}

async function rasterizeTemplate(page, htmlPath, dest) {
  const html = readFileSync(htmlPath, 'utf8');
  const width = Number(/data-width="(\d+)"/.exec(html)?.[1] ?? 0);
  const height = Number(/data-height="(\d+)"/.exec(html)?.[1] ?? 0);
  const transparent = html.includes('data-transparent="true"');
  await page.setViewportSize({ width, height });
  await page.goto(pathToFileURL(htmlPath).href, { waitUntil: 'networkidle' });
  await waitForFonts(page);
  const lockup = page.locator('.lockup');
  if ((await lockup.count()) > 0) {
    await lockup.screenshot({ path: dest, omitBackground: transparent });
    return;
  }

  await page.screenshot({ path: dest, omitBackground: transparent });
}

const browser = await chromium.launch();
const page = await browser.newPage();

const svgJobs = [
  ['icon.svg', 'app-icon.png', 1024, 1024, false],
  ['icon.svg', 'github-org-avatar.png', 1024, 1024, false],
  ['icon.svg', 'github-org-avatar-512.png', 512, 512, false],
  ['icon.svg', 'apple-touch-icon.png', 180, 180, false],
  ['icon.svg', 'pwa-512.png', 512, 512, false],
  ['icon.svg', 'pwa-192.png', 192, 192, false],
  ['icon-light.svg', 'icon-light.png', 1024, 1024, false],
  ['favicon.svg', 'favicon-32.png', 32, 32, true],
  ['favicon.svg', 'favicon-16.png', 16, 16, true],
  ['mark-paper-mint.svg', 'mark-paper-mint-512.png', 512, 512, true],
  ['mark-ink-blue.svg', 'mark-ink-blue-512.png', 512, 512, true],
];

for (const [src, dest, width, height, transparent] of svgJobs) {
  await rasterizeSvg(page, join(svgDir, src), join(outDir, dest), width, height, transparent);
}

const templateJobs = [
  ['github-hero.html', 'github-social-preview.png'],
  ['github-readme-hero.html', 'github-readme-hero.png'],
  ['og-image.html', 'og-image.png'],
  ['social-banner.html', 'social-banner.png'],
  ['lockup-dark.html', 'lockup-dark.png'],
  ['lockup-light.html', 'lockup-light.png'],
];

for (const [src, dest] of templateJobs) {
  await rasterizeTemplate(page, join(templates, src), join(outDir, dest));
}

await browser.close();

writeFileSync(
  join(outDir, 'favicon.ico'),
  pngToIco([
    readFileSync(join(outDir, 'favicon-16.png')),
    readFileSync(join(outDir, 'favicon-32.png')),
  ]),
);

const site = join(root, 'site');
copyFileSync(join(svgDir, 'favicon.svg'), join(site, 'favicon.svg'));
copyFileSync(join(svgDir, 'safari-mask.svg'), join(site, 'safari-mask.svg'));
copyFileSync(join(outDir, 'favicon.ico'), join(site, 'favicon.ico'));
copyFileSync(join(outDir, 'apple-touch-icon.png'), join(site, 'apple-touch-icon.png'));
copyFileSync(join(outDir, 'og-image.png'), join(site, 'og.png'));
copyFileSync(join(svgDir, 'favicon.svg'), join(root, 'desktop/src/favicon.svg'));

const tauri = spawnSync(
  'bunx',
  [
    'tauri',
    'icon',
    join(outDir, 'app-icon.png'),
    '--output',
    join(root, 'desktop/src-tauri/icons'),
  ],
  { cwd: join(root, 'desktop'), encoding: 'utf8' },
);

for (const path of ['64x64.png', 'android', 'ios']) {
  rmSync(join(root, 'desktop/src-tauri/icons', path), { force: true, recursive: true });
}

if (tauri.status !== 0) {
  process.stderr.write(tauri.stderr || tauri.stdout);
  process.exit(tauri.status ?? 1);
}

process.stdout.write(`wrote brand exports to ${outDir}\n`);
