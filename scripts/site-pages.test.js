import { expect, test } from 'bun:test';

import { headingId, mdHeadings, rewriteDocHrefs } from './site-pages.mjs';

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
