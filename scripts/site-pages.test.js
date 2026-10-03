import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { transform } from '@astrojs/compiler-rs';
import { validateConfig } from 'astro/config';
import { experimental_AstroContainer as AstroContainer } from 'astro/container';

import config from '../site/astro.config.mjs';
import { headingId, mdHeadings, mdToHtml, rewriteDocHrefs } from './site-pages.mjs';

test('headingId slugs guide titles', () => {
  expect(headingId('Lock files')).toBe('lock-files');
  expect(headingId('Encrypted in place')).toBe('encrypted-in-place');
});

test('mdHeadings lists guide sections', () => {
  expect(mdHeadings('# Using\n\n## Get started\n\n## Access\n')).toEqual([
    { title: 'Get started', id: 'get-started' },
    { title: 'Access', id: 'access' },
  ]);
});

test('rewriteDocHrefs no longer points at the roadmap', () => {
  expect(rewriteDocHrefs('../.scratch/sopsdeck-product/map.md')).toBe('/docs/');
});

test('rewriteDocHrefs keeps contributor glossary links on the user docs', () => {
  expect(rewriteDocHrefs('../CONTEXT.md')).toBe('/docs/');
});

test('site rendering keeps spaces around links split across lines', async () => {
  const normalized = await validateConfig(
    config,
    new URL('../site/', import.meta.url).pathname,
    'test',
  );
  const source = readFileSync(new URL('../site/src/pages/index.astro', import.meta.url), 'utf8');
  const paragraph = source.match(/<p class="security-footnote">[\s\S]*?<\/p>/)[0];
  const { code } = transform(paragraph, {
    compact: normalized.compressHTML,
    astroGlobalArgs: JSON.stringify(normalized.site),
    internalURL: import.meta.resolve('astro/compiler-runtime'),
    resolvePath: (specifier) => specifier,
  });
  const { default: component } = await import(
    `data:text/javascript;base64,${Buffer.from(code).toString('base64')}`
  );
  const container = await AstroContainer.create();
  const text = (await container.renderToString(component))
    .replaceAll(/<[^>]+>/g, '')
    .replaceAll(/\s+/g, ' ');
  expect(text).toContain('covered in what Sopsdeck stores');
  expect(text).toContain('SOPS threat model and the age documentation');
});

test('docs keep inline code and link URLs literal while formatting prose', () => {
  expect(
    mdToHtml(
      '`SOPS_AGE_KEY`, `SOPS_AGE_KEY_FILE`, and `SOPS_AGE_KEY_CMD`. ' +
        '`**literal** [link](url) <tag> & value` with _emphasis_ and **bold**. ' +
        '[the `GH_TOKEN` key](https://example.com/SOPS_AGE_KEY?x=1&y=2)',
    ),
  ).toBe(
    '<p><code>SOPS_AGE_KEY</code>, <code>SOPS_AGE_KEY_FILE</code>, and <code>SOPS_AGE_KEY_CMD</code>. ' +
      '<code>**literal** [link](url) &lt;tag&gt; &amp; value</code> with <em>emphasis</em> and <strong>bold</strong>. ' +
      '<a href="https://example.com/SOPS_AGE_KEY?x=1&amp;y=2">the <code>GH_TOKEN</code> key</a></p>',
  );
});
