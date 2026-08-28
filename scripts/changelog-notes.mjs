const TYPE_FROM_HEADING = {
  added: 'feature',
  fixed: 'bugfix',
  changed: 'changed',
  deprecated: 'changed',
  removed: 'removed',
  security: 'security',
  performance: 'performance',
};

export const TYPE_LABEL = {
  feature: 'Feature',
  bugfix: 'Bug fix',
  performance: 'Performance',
  changed: 'Changed',
  removed: 'Removed',
  security: 'Security',
};

const TYPE_ORDER = ['feature', 'bugfix', 'performance', 'changed', 'removed', 'security'];

export function platformsOf(text) {
  const out = [];
  if (/\bmac\s*os\b/i.test(text) || /\bmacos\b/i.test(text)) out.push('macOS');
  if (/\bwindows\b/i.test(text)) out.push('Windows');
  if (/\blinux\b/i.test(text)) out.push('Linux');
  return out;
}

export function typeFromHeading(heading) {
  return (
    TYPE_FROM_HEADING[
      String(heading || '')
        .trim()
        .toLowerCase()
    ] || 'changed'
  );
}

export function parseHeading(line) {
  const match = /^##\s+(?:\[([^\]]+)\]|(\S+))(?:\s+-\s+(\d{4}-\d{2}-\d{2}))?/.exec(
    String(line || '').trim(),
  );
  if (!match) return;
  return { heading: match[1] || match[2], date: match[3] || '' };
}

function taggedNote(text, type) {
  return { text, type, platforms: platformsOf(text) };
}

export function parseChangelog(md) {
  const lines = String(md || '').split('\n');
  const sections = [];
  let current;
  let type = 'changed';
  for (const raw of lines) {
    const line = raw.trim();
    const heading = parseHeading(line);
    if (heading && line.startsWith('## ')) {
      current = { heading: heading.heading, date: heading.date, notes: [] };
      type = 'changed';
      sections.push(current);
      continue;
    }

    if (!current) continue;
    if (line.startsWith('### ')) {
      type = typeFromHeading(line.slice(4));
      continue;
    }

    if (line.startsWith('- ')) {
      current.notes.push(taggedNote(line.slice(2).trim(), type));
    }
  }

  for (const section of sections) {
    section.notes.sort((a, b) => TYPE_ORDER.indexOf(a.type) - TYPE_ORDER.indexOf(b.type));
  }

  return sections;
}

export function changelogSectionNotes(md, heading) {
  return parseChangelog(md).find((section) => section.heading === heading);
}

export function typeLabel(type) {
  return TYPE_LABEL[type] || TYPE_LABEL.changed;
}
