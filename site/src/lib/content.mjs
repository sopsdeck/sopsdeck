import { parseChangelog } from '../../../scripts/changelog-notes.mjs';
import { mdToHtml } from '../../../scripts/site-pages.mjs';

import changelog from '../../../CHANGELOG.md?raw';
import context from '../../../CONTEXT.md?raw';
import assets from '../../../docs/assets.md?raw';
import features from '../../../docs/features.md?raw';
import seams from '../../../docs/seams.md?raw';
import versioning from '../../../docs/versioning.md?raw';
import roadmap from '../../../.scratch/sopsdeck-product/build.md?raw';
import catalog from '../../../docs/assets/catalog.json';

export function repoFile(relativePath) {
  return {
    'CHANGELOG.md': changelog,
    'CONTEXT.md': context,
    'docs/assets.md': assets,
    'docs/features.md': features,
    'docs/seams.md': seams,
    'docs/versioning.md': versioning,
    '.scratch/sopsdeck-product/build.md': roadmap,
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

export function roadmapPhases() {
  const section = roadmap.split('## Phase status')[1] ?? '';
  const table = section.split('## Ready build tickets')[0] ?? '';
  return table
    .split('\n')
    .map((line) => {
      const cells = line
        .trim()
        .replace(/^\|/, '')
        .replace(/\|$/, '')
        .split('|')
        .map((cell) => cell.trim());
      if (cells.length !== 5 || !/^\d+$/.test(cells[0])) return;
      return {
        phase: cells[0],
        slice: cells[1],
        status: cells[2].replaceAll('**', ''),
        proved: cells[3].replaceAll('`', ''),
        open: cells[4].replaceAll('`', ''),
      };
    })
    .filter(Boolean);
}
