import { expect, test } from 'bun:test';
import { selectedProjectFiles } from './main.js';

test('selectedProjectFiles excludes unchecked keyless candidates', () => {
  const row = (path, checked, allKeys = []) => ({
    input: { value: path, checked },
    keyInputs: [],
    allKeys,
  });

  expect(
    selectedProjectFiles([
      row('apps/wiki/tsconfig.json', false),
      row('apps/wiki/.env', true, ['FIREBASE_DOT_JSON']),
      row('README.md', true),
    ]),
  ).toEqual([
    { path: 'apps/wiki/.env', keys: ['FIREBASE_DOT_JSON'] },
    { path: 'README.md', keys: [] },
  ]);
});
