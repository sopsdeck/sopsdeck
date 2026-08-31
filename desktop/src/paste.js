export function parsePastePayload(text, loneKey = '') {
  const raw = String(text ?? '');
  const trimmed = raw.trim();
  if (!trimmed) {
    throw new Error('empty paste');
  }

  if (trimmed.startsWith('{')) {
    const doc = JSON.parse(trimmed);
    if (!doc || typeof doc !== 'object' || Array.isArray(doc)) {
      throw new Error('invalid JSON');
    }

    return stringifyPasteMap(doc);
  }

  if (looksDotenv(raw)) {
    return dotenvMap(raw);
  }

  const yaml = parseYAMLMap(raw);
  if (yaml) {
    return yaml;
  }

  if (!loneKey) {
    const err = new Error('lone paste value requires KEY');
    err.code = 'LONE_KEY';
    throw err;
  }

  return { [loneKey]: raw };
}

export function classifyPasteKeys(current, incoming) {
  const adds = [];
  const changes = [];
  for (const key of Object.keys(incoming).sort()) {
    if (Object.hasOwn(current, key)) {
      changes.push(key);
    } else {
      adds.push(key);
    }
  }

  return { adds, changes };
}

export function pastePreviewText(adds, changes) {
  const lines = [`preview ${adds.length} add ${changes.length} change`];
  for (const key of adds) lines.push(`add ${key}`);
  for (const key of changes) lines.push(`change ${key}`);
  return lines.join('\n');
}

export function parseGitIdentity(text) {
  const raw = String(text ?? '').trim();
  const match = /^(.*?)\s+<([^<>]+)>\s*$/u.exec(raw);
  if (match) {
    return { name: match[1].trim(), email: match[2].trim() };
  }

  return { name: raw, email: '' };
}

export function classifyClipboard(text) {
  const trimmed = String(text ?? '').trim();
  if (!trimmed) return null;
  if (!trimmed.includes('\n') && (trimmed.startsWith('/') || /^[A-Za-z]:[\\/]/u.test(trimmed))) {
    return { kind: 'path', path: trimmed };
  }

  const age = trimmed.match(/\bage1[0-9a-z]{58}\b/u);
  if (age && (trimmed === age[0] || /^Name:/mu.test(trimmed) || /Age public key/u.test(trimmed))) {
    const named = trimmed.match(/^Name:\s*(.+)$/mu);
    const identity = parseGitIdentity(named?.[1] ?? '');
    return {
      kind: 'recipient',
      publicKey: age[0],
      name: identity.name,
      email: identity.email,
    };
  }

  try {
    const pairs = parsePastePayload(text);
    return { kind: 'bulk', pairs, names: Object.keys(pairs).sort() };
  } catch (err) {
    if (err?.code === 'LONE_KEY') return { kind: 'lone', value: String(text) };
    return null;
  }
}

function looksDotenv(text) {
  for (const line of text.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const eq = trimmed.indexOf('=');
    if (eq > 0) return true;
  }

  return false;
}

function dotenvMap(text) {
  const out = {};
  for (const line of text.split('\n')) {
    const eq = line.indexOf('=');
    if (eq < 1) continue;
    const key = line.slice(0, eq);
    const value = line.slice(eq + 1).replaceAll(String.raw`\n`, '\n');
    if (key) out[key] = value;
  }

  return out;
}

function parseYAMLMap(text) {
  const out = {};
  let found = false;
  for (const line of text.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const match = /^([^:#\s][^:]*)\s*:\s*(.*)$/.exec(trimmed);
    if (!match) return null;
    found = true;
    out[match[1].trim()] = unquoteYAML(match[2]);
  }

  return found ? out : null;
}

function unquoteYAML(value) {
  if (
    (value.startsWith('"') && value.endsWith('"')) ||
    (value.startsWith("'") && value.endsWith("'"))
  ) {
    return value.slice(1, -1);
  }

  return value;
}

function stringifyPasteMap(doc) {
  const out = {};
  for (const [key, raw] of Object.entries(doc)) {
    out[key] = typeof raw === 'string' ? raw : String(raw);
  }

  return out;
}
