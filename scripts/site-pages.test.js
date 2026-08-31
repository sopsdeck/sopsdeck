import { expect, test } from 'bun:test';

import { headingId, rewriteDocHrefs } from './site-pages.mjs';

test('headingId slugs guide titles', () => {
  expect(headingId('Lock files')).toBe('lock-files');
  expect(headingId('Encrypted in place')).toBe('encrypted-in-place');
});

test('rewriteDocHrefs no longer points at the roadmap', () => {
  expect(rewriteDocHrefs('../.scratch/sopsdeck-product/map.md')).toBe('/docs/');
});
