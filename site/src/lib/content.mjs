import { parseChangelog } from '../../../scripts/changelog-notes.mjs';
import { mdToHtml } from '../../../scripts/site-pages.mjs';

import changelog from '../../../CHANGELOG.md?raw';
import context from '../../../CONTEXT.md?raw';
import assets from '../../../docs/assets.md?raw';
import features from '../../../docs/features.md?raw';
import guide from '../../../docs/guide.md?raw';
import seams from '../../../docs/seams.md?raw';
import versioning from '../../../docs/versioning.md?raw';
import catalog from '../../../docs/assets/catalog.json';

export function repoFile(relativePath) {
  return {
    'CHANGELOG.md': changelog,
    'CONTEXT.md': context,
    'docs/assets.md': assets,
    'docs/features.md': features,
    'docs/guide.md': guide,
    'docs/seams.md': seams,
    'docs/versioning.md': versioning,
  }[relativePath];
}

export function repoJson(relativePath) {
  if (relativePath === 'docs/assets/catalog.json') return catalog;
  return JSON.parse(repoFile(relativePath));
}

export function docHtml(relativePath) {
  return mdToHtml(repoFile(relativePath).replace(/^# .+\n+/, ''));
}

export function changelogSections() {
  return parseChangelog(changelog);
}
