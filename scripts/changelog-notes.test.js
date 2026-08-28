import { expect, test } from 'bun:test';

import { parseChangelog, parseHeading, platformsOf, typeFromHeading } from './changelog-notes.mjs';

test('parseHeading reads version and date', () => {
  expect(parseHeading('## Unreleased')).toEqual({ heading: 'Unreleased', date: '' });
  expect(parseHeading('## 1.0.0 - 2026-08-28')).toEqual({
    heading: '1.0.0',
    date: '2026-08-28',
  });
  expect(parseHeading('## [1.0.0] - 2026-08-28')).toEqual({
    heading: '1.0.0',
    date: '2026-08-28',
  });
});

test('Keep a Changelog headings map to type tags', () => {
  expect(typeFromHeading('Added')).toBe('feature');
  expect(typeFromHeading('Fixed')).toBe('bugfix');
  expect(typeFromHeading('Changed')).toBe('changed');
  expect(typeFromHeading('Performance')).toBe('performance');
});

test('platformsOf only tags named platforms', () => {
  expect(platformsOf('Works on macOS and Linux')).toEqual(['macOS', 'Linux']);
  expect(platformsOf('Inspector collapse')).toEqual([]);
});

test('parseChangelog groups by version and type', () => {
  const sections = parseChangelog(`## Unreleased

### Added
- Nested folders
- Signed macOS builds

### Fixed
- Folder picker hang

### Performance
- Faster Sync
`);
  expect(sections).toHaveLength(1);
  expect(sections[0].heading).toBe('Unreleased');
  expect(sections[0].notes.map((note) => note.type)).toEqual([
    'feature',
    'feature',
    'bugfix',
    'performance',
  ]);
  expect(sections[0].notes[1].platforms).toEqual(['macOS']);
});
