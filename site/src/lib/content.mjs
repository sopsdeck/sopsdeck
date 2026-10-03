import { mdHeadings, mdToHtml } from '../../../scripts/site-pages.mjs';

import changelog from '../../../CHANGELOG.md?raw';
import cli from '../../../docs/cli.md?raw';
import guide from '../../../docs/guide.md?raw';

export function repoFile(relativePath) {
  return {
    'CHANGELOG.md': changelog,
    'docs/cli.md': cli,
    'docs/guide.md': guide,
  }[relativePath];
}

export function docHtml(relativePath) {
  return mdToHtml(repoFile(relativePath).replace(/^# .+\n+/, ''));
}

export { changelog };

export function guideHeadings() {
  return mdHeadings(guide);
}

export function cliHeadings() {
  return mdHeadings(cli);
}
