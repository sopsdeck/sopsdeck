import { classifyPasteKeys, parsePastePayload, pastePreviewText } from './paste.js';

const invoke = globalThis.__TAURI__?.core?.invoke ?? invokeOverHTTP;
const THEME_KEY = 'sopsdeck-theme';
const INSPECTOR_KEY = 'sopsdeck-inspector';
const RECENTS_KEY = 'sopsdeck-recents';
const TREE_FOLDERS_KEY = 'sopsdeck-tree-folders';
const TREE_PROJECTS_KEY = 'sopsdeck-tree-projects';
const TREE_LIMIT = 8;

async function invokeOverHTTP(cmd, args = {}) {
  const response = await fetch('/invoke', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ cmd, ...args }),
  });
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.error || response.statusText);
  }

  return payload.result;
}

const treeEl = () => document.getElementById('tree');
const keysEl = () => document.getElementById('keys');
const emptyEl = () => document.getElementById('empty-state');
const crumbEl = () => document.getElementById('breadcrumb');
const headlineEl = () => document.getElementById('headline');
const sublineEl = () => document.getElementById('subline');
const errorEl = () => document.getElementById('error');
const gitErrorEl = () => document.getElementById('git-error');
const saveErrorEl = () => document.getElementById('save-error');
const accessErrorEl = () => document.getElementById('access-error');
const publishErrorEl = () => document.getElementById('publish-error');
const badgeEl = () => document.getElementById('badge');
const saveEl = () => document.getElementById('save');
const commitEl = () => document.getElementById('commit-message');

const projects = [];
let selected = null;
let rows = [];
let revealed = false;
let commitAuto = true;
let lastAuto = '';
let historyRev = '';
let composerFocus = false;
let pendingPaste = null;
const treeShowAll = new Set();

function messageOf(err) {
  if (err instanceof Error) return err.message;
  return String(err);
}

function skipBoot() {
  return new URLSearchParams(location.search).has('empty');
}

const stroke = {
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': '1.8',
  'stroke-linecap': 'round',
  'stroke-linejoin': 'round',
};

function svgEl(name, attrs, children = []) {
  const el = document.createElementNS('http://www.w3.org/2000/svg', name);
  for (const [key, value] of Object.entries(attrs)) {
    el.setAttribute(key, value);
  }

  for (const child of children) el.append(child);
  return el;
}

function icon(kind) {
  const parts = {
    eye: [
      svgEl('path', { d: 'M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12Z', ...stroke }),
      svgEl('circle', { cx: '12', cy: '12', r: '2.5', ...stroke }),
    ],
    'eye-off': [
      svgEl('path', { d: 'M3 3l18 18', ...stroke }),
      svgEl('path', { d: 'M10.6 10.6a2.5 2.5 0 0 0 3.5 3.5', ...stroke }),
      svgEl('path', {
        d: 'M9.9 5.1A11 11 0 0 1 12 5c6.5 0 10 7 10 7a18 18 0 0 1-3.2 3.8',
        ...stroke,
      }),
      svgEl('path', { d: 'M6.1 6.1C3.6 8 2 12 2 12s3.5 7 10 7a10 10 0 0 0 4.2-.9', ...stroke }),
    ],
    copy: [
      svgEl('rect', { x: '9', y: '9', width: '13', height: '13', rx: '2', ...stroke }),
      svgEl('path', { d: 'M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1', ...stroke }),
    ],
    trash: [
      svgEl('path', { d: 'M4 7h16', ...stroke }),
      svgEl('path', { d: 'M10 11v6M14 11v6', ...stroke }),
      svgEl('path', { d: 'M6 7l1 14h10l1-14', ...stroke }),
      svgEl('path', { d: 'M9 7V4h6v3', ...stroke }),
    ],
    plus: [svgEl('path', { d: 'M12 5v14M5 12h14', ...stroke })],
    folder: [
      svgEl('path', { d: 'M3 7h6l2 2h10v10H3z', ...stroke }),
      svgEl('path', { d: 'M3 7V5h5l2 2', ...stroke }),
    ],
    file: [
      svgEl('path', { d: 'M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V9z', ...stroke }),
      svgEl('path', { d: 'M14 3v6h6', ...stroke }),
    ],
    spark: [
      svgEl('path', { d: 'M12 3v4M12 17v4M3 12h4M17 12h4', ...stroke }),
      svgEl('circle', { cx: '12', cy: '12', r: '3', ...stroke }),
    ],
    save: [
      svgEl('path', { d: 'M5 5h10l4 4v10H5z', ...stroke }),
      svgEl('path', { d: 'M8 5v5h8', ...stroke }),
    ],
    commit: [
      svgEl('circle', { cx: '12', cy: '12', r: '3', ...stroke }),
      svgEl('path', { d: 'M12 5v4M12 15v4', ...stroke }),
    ],
    sync: [
      svgEl('path', { d: 'M4 12a8 8 0 0 1 13-5.5L19 9', ...stroke }),
      svgEl('path', { d: 'M20 12a8 8 0 0 1-13 5.5L5 15', ...stroke }),
    ],
    review: [svgEl('path', { d: 'M4 6h16M4 12h10M4 18h13', ...stroke })],
    history: [
      svgEl('circle', { cx: '12', cy: '12', r: '9', ...stroke }),
      svgEl('path', { d: 'M12 7v5l3 2', ...stroke }),
    ],
    restore: [
      svgEl('path', { d: 'M3 12a9 9 0 1 0 3-6.7', ...stroke }),
      svgEl('path', { d: 'M3 4v5h5', ...stroke }),
    ],
    grant: [
      svgEl('circle', { cx: '9', cy: '8', r: '3', ...stroke }),
      svgEl('path', { d: 'M3 19c1-4 4-6 6-6s5 2 6 6', ...stroke }),
      svgEl('path', { d: 'M19 8v6M16 11h6', ...stroke }),
    ],
    drop: [
      svgEl('circle', { cx: '9', cy: '8', r: '3', ...stroke }),
      svgEl('path', { d: 'M3 19c1-4 4-6 6-6s5 2 6 6M16 11h6', ...stroke }),
    ],
    publish: [svgEl('path', { d: 'M12 19V5M6 11l6-6 6 6', ...stroke })],
    sun: [
      svgEl('circle', { cx: '12', cy: '12', r: '4', ...stroke }),
      svgEl('path', {
        d: 'M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
        ...stroke,
      }),
    ],
    moon: [svgEl('path', { d: 'M21 14.5A8.5 8.5 0 1 1 9.5 3 7 7 0 0 0 21 14.5z', ...stroke })],
    chevron: [svgEl('path', { d: 'M8 10l4 4 4-4', ...stroke })],
  };
  return svgEl(
    'svg',
    { viewBox: '0 0 24 24', width: '14', height: '14', 'aria-hidden': 'true' },
    parts[kind] || [],
  );
}

function iconButton(testid, label, kind, onClick) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'icon-button';
  btn.dataset.testid = testid;
  btn.setAttribute('aria-label', label);
  btn.append(icon(kind));
  btn.addEventListener('click', onClick);
  return btn;
}

function copyText(text) {
  const clip = navigator.clipboard;
  if (!clip?.writeText) return;
  (async () => {
    try {
      await clip.writeText(String(text));
    } catch {
      // Clipboard can fail without permission or a secure context.
    }
  })();
}

function normalizePosix(path) {
  const parts = [];
  for (const part of String(path || '').split('/')) {
    if (!part || part === '.') continue;
    if (part === '..') {
      parts.pop();
      continue;
    }

    parts.push(part);
  }

  return `/${parts.join('/')}`;
}

function resolveManagedPath(projectPath, name) {
  const raw = String(name || '')
    .trim()
    .replaceAll('\\', '/');
  if (!raw) {
    throw new Error('Name a Managed File');
  }

  const root = String(projectPath || '').replace(/\/+$/u, '');
  if (!root) {
    throw new Error('Open a Project first');
  }

  if (raw.split('/').includes('..')) {
    throw new Error('Path stays inside the Project');
  }

  const isAbs = raw.startsWith('/');
  const abs = normalizePosix(isAbs ? raw : `${root}/${raw}`);
  if (abs !== root && !abs.startsWith(`${root}/`)) {
    throw new Error('Path stays inside the Project');
  }

  return abs;
}

function currentPasteKeys() {
  const out = {};
  for (const row of visibleRows()) {
    if (row.key) out[row.key] = row.value;
  }

  return out;
}

function applyPastePairs(pairs) {
  for (const key of Object.keys(pairs).sort()) {
    const value = pairs[key];
    const existing = rows.find((r) => !r.deleted && r.key === key);
    if (existing) {
      existing.value = value;
      continue;
    }

    rows.push({
      key,
      value,
      origKey: '',
      origValue: '',
      added: true,
      deleted: false,
      revealed: true,
    });
  }
}

function renderPasteChrome(box) {
  if (!pendingPaste) return;
  const preview = document.createElement('div');
  preview.className = 'paste-preview';
  if (pendingPaste.kind === 'bulk') {
    preview.dataset.testid = 'paste-preview';
    const body = document.createElement('pre');
    body.textContent = pastePreviewText(pendingPaste.adds, pendingPaste.changes);
    const confirm = document.createElement('button');
    confirm.type = 'button';
    confirm.className = 'tool';
    confirm.dataset.testid = 'paste-confirm';
    confirm.textContent = 'Apply paste';
    confirm.addEventListener('click', () => {
      applyPastePairs(pendingPaste.pairs);
      pendingPaste = null;
      renderKeys();
    });
    const cancel = document.createElement('button');
    cancel.type = 'button';
    cancel.className = 'tool';
    cancel.dataset.testid = 'paste-cancel';
    cancel.textContent = 'Cancel';
    cancel.addEventListener('click', () => {
      pendingPaste = null;
      renderKeys();
    });
    const actions = document.createElement('div');
    actions.className = 'paste-preview-actions';
    actions.append(confirm, cancel);
    preview.append(body, actions);
    box.append(preview);
    return;
  }

  preview.dataset.testid = 'paste-key-prompt';
  const input = document.createElement('input');
  input.type = 'text';
  input.dataset.testid = 'paste-key-name';
  input.placeholder = 'Key for paste';
  input.autocomplete = 'off';
  input.spellcheck = false;
  const apply = () => {
    const key = input.value.trim();
    if (!key) return;
    applyPastePairs({ [key]: pendingPaste.value });
    pendingPaste = null;
    renderKeys();
  };

  input.addEventListener('keydown', (event) => {
    if (event.key !== 'Enter') return;
    event.preventDefault();
    apply();
  });
  const confirm = document.createElement('button');
  confirm.type = 'button';
  confirm.className = 'tool';
  confirm.dataset.testid = 'paste-key-apply';
  confirm.textContent = 'Add';
  confirm.addEventListener('click', apply);
  preview.append(input, confirm);
  box.append(preview);
}

function applyLoneToFocused(input, text) {
  const line = input.closest('[data-testid="key-row"]');
  const name = line?.querySelector('[data-testid="key-name"]')?.value;
  const row = rows.find((r) => !r.deleted && r.key === name);
  if (!row) return;
  row.value = text;
  row.revealed = true;
  renderKeys();
}

function onEditorPaste(event) {
  if (!selected) return;
  const box = keysEl();
  if (!box || box.hidden) return;
  const { target } = event;
  if (!(target instanceof Node) || !box.contains(target)) return;

  const text = event.clipboardData?.getData('text/plain') ?? '';
  if (!text.trim()) return;

  try {
    const pairs = parsePastePayload(text);
    if (Object.keys(pairs).length === 0) return;
    event.preventDefault();
    const { adds, changes } = classifyPasteKeys(currentPasteKeys(), pairs);
    pendingPaste = { kind: 'bulk', pairs, adds, changes };
    renderKeys();
  } catch (err) {
    if (err?.code !== 'LONE_KEY') {
      if (text.trim().startsWith('{')) event.preventDefault();
      return;
    }

    event.preventDefault();
    if (target?.dataset?.testid === 'key-value') {
      applyLoneToFocused(target, text);
      return;
    }

    pendingPaste = { kind: 'lone', value: text };
    renderKeys();
  }
}

function parseComposerLine(text) {
  const line = String(text || '')
    .split('\n')[0]
    .trim();
  if (!line) return null;
  const eq = line.indexOf('=');
  if (eq === -1) {
    return { key: line, value: '' };
  }

  return { key: line.slice(0, eq).trim(), value: line.slice(eq + 1) };
}

function rowDirty(row) {
  return Boolean(
    row.deleted || row.added || row.key !== row.origKey || row.value !== row.origValue,
  );
}

function visibleRows() {
  return rows.filter((row) => !row.deleted);
}

function clearErrors() {
  for (const el of [errorEl(), gitErrorEl(), saveErrorEl(), accessErrorEl(), publishErrorEl()]) {
    if (!el) continue;
    el.hidden = true;
    el.textContent = '';
  }
}

function showError(msg, region = 'editor') {
  clearErrors();
  if (!msg) return;
  const el =
    {
      git: gitErrorEl(),
      save: saveErrorEl(),
      access: accessErrorEl(),
      publish: publishErrorEl(),
    }[region] ?? errorEl();
  el.hidden = false;
  el.textContent = msg;
}

function setStatus(kind, text) {
  const el = document.getElementById(`${kind}-status`);
  if (!el) return;
  el.hidden = !text;
  el.textContent = text || '';
}

function formatOf(path) {
  const base = path.split('/').pop() || path;
  if (base === '.env' || base.startsWith('.env.') || base.toLowerCase().endsWith('.env')) {
    return 'dotenv';
  }

  if (base.toLowerCase().endsWith('.json')) return 'json';
  if (base.toLowerCase().endsWith('.yaml') || base.toLowerCase().endsWith('.yml')) return 'yaml';
  return 'unknown';
}

function titleOf(name) {
  if (name.startsWith('.env.')) {
    const rest = name.slice(5);
    return rest.charAt(0).toUpperCase() + rest.slice(1);
  }

  return name;
}

function parentLabel(path) {
  const parts = path.split('/').filter(Boolean);
  if (parts.length < 2) return path;
  return `~/${parts.at(-2)}`;
}

function safeRel(rel, name) {
  const raw = String(rel || name || '').replaceAll('\\', '/');
  const cleaned = raw
    .split('/')
    .filter((part) => part && part !== '.' && part !== '..')
    .join('/');
  return cleaned || name || '';
}

function fileLabel(rel, name) {
  const cleaned = safeRel(rel, name);
  const parts = cleaned.split('/');
  return parts.at(-1) || cleaned;
}

function fileDir(rel, name) {
  const parts = safeRel(rel, name).split('/');
  if (parts.length < 2) return '';
  return parts.slice(0, -1).join('/');
}

function readJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return fallback;
    const parsed = JSON.parse(raw);
    return parsed ?? fallback;
  } catch {
    return fallback;
  }
}

function rememberRecent(project) {
  const items = readJSON(RECENTS_KEY, []).filter(
    (item) => item && item.path && item.path !== project.path,
  );
  items.unshift({ path: project.path, name: project.name });
  localStorage.setItem(RECENTS_KEY, JSON.stringify(items.slice(0, TREE_LIMIT)));
}

function displayPath(project, file) {
  const name = project?.name || 'project';
  return `~/${name}/${safeRel(file.rel, file.name)}`;
}

function dirtyCount() {
  return rows.filter((r) => rowDirty(r)).length;
}

function defaultCommitMessage(current) {
  const added = [];
  const changed = [];
  const removed = [];
  for (const row of current) {
    if (row.deleted) {
      if (row.origKey) removed.push(row.origKey);
      continue;
    }

    if (row.added && row.key) {
      added.push(row.key);
      continue;
    }

    if (row.key !== row.origKey || row.value !== row.origValue) {
      changed.push(row.key || row.origKey);
    }
  }

  const parts = [];
  if (added.length > 0) parts.push(`Add ${added.join(', ')}`);
  if (changed.length > 0) parts.push(`Change ${changed.join(', ')}`);
  if (removed.length > 0) parts.push(`Remove ${removed.join(', ')}`);
  return parts.join('; ');
}

function syncCommitMessage() {
  if (!commitAuto) return;
  const next = defaultCommitMessage(rows);
  if (next === '') return;
  lastAuto = next;
  commitEl().value = lastAuto;
}

function setFileNote(text) {
  const el = document.getElementById('file-note');
  if (!el) return;
  el.hidden = !text;
  el.textContent = text || '';
}

function showEmpty(message) {
  const empty = emptyEl();
  empty.hidden = !message;
  empty.textContent = message || '';
}

function resetEditorChrome() {
  selected = null;
  rows = [];
  revealed = false;
  badgeEl().hidden = true;
  keysEl().hidden = true;
  keysEl().replaceChildren();
  document.getElementById('meta-path').textContent = '—';
  document.getElementById('meta-format').textContent = '—';
  document.getElementById('meta-enc').textContent = '—';
  saveEl().disabled = true;
  setFileNote('');
}

function renderWorkspace() {
  if (projects.length === 0) {
    resetEditorChrome();
    renderTree();
    crumbEl().textContent = 'Add a project folder to begin';
    headlineEl().textContent = 'Sopsdeck';
    sublineEl().textContent = 'Encrypted files stay on this machine';
    showEmpty('No Project yet. Add a folder from disk.');
    return;
  }

  const project = selected?.project ?? projects[0];
  if (!selected && (!project.files || project.files.length === 0)) {
    resetEditorChrome();
    crumbEl().textContent = `~/${project.name}`;
    headlineEl().textContent = project.name;
    sublineEl().textContent = 'No Managed Files in this Project';
    showEmpty('This Project has no Managed Files.');
  }
}

function currentTheme() {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
}

function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  localStorage.setItem(THEME_KEY, theme);
  const btn = document.getElementById('theme-toggle');
  if (!btn) return;
  const next = theme === 'dark' ? 'light' : 'dark';
  btn.replaceChildren(icon(theme === 'dark' ? 'sun' : 'moon'));
  btn.setAttribute('aria-label', `Switch to ${next} theme`);
}

function readInspectorState() {
  try {
    const raw = JSON.parse(localStorage.getItem(INSPECTOR_KEY) || '{}');
    return raw && typeof raw === 'object' ? raw : {};
  } catch {
    return {};
  }
}

function writeInspectorState(state) {
  const collapsed = {};
  for (const [id, value] of Object.entries(state)) {
    if (value) collapsed[id] = true;
  }

  localStorage.setItem(INSPECTOR_KEY, JSON.stringify(collapsed));
}

function applyInspectorCollapsed(section, collapsed) {
  const toggle = section.querySelector('.inspect-toggle');
  section.classList.toggle('collapsed', collapsed);
  if (toggle) toggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
}

function initInspector() {
  const stored = readInspectorState();
  for (const section of document.querySelectorAll('.inspect-section[data-section]')) {
    const id = section.dataset.section;
    const toggle = section.querySelector('.inspect-toggle');
    if (!toggle) continue;
    applyInspectorCollapsed(section, Boolean(stored[id]));
    toggle.prepend(icon('chevron'));
    toggle.addEventListener('click', () => {
      const next = !section.classList.contains('collapsed');
      applyInspectorCollapsed(section, next);
      stored[id] = next;
      writeInspectorState(stored);
    });
  }
}

function toggleReveal() {
  revealed = !revealed;
  for (const row of rows) row.revealed = revealed;
  renderKeys();
}

function groupFiles(files) {
  const groups = new Map();
  const sorted = [...files].sort((a, b) =>
    safeRel(a.rel, a.name).localeCompare(safeRel(b.rel, b.name)),
  );
  for (const file of sorted) {
    const dir = fileDir(file.rel, file.name);
    const list = groups.get(dir) || [];
    list.push(file);
    groups.set(dir, list);
  }

  return groups;
}

function folderStorageKey(projectPath, dir) {
  return `${projectPath}\n${dir}`;
}

function projectCollapsed(path, index) {
  const stored = readJSON(TREE_PROJECTS_KEY, {});
  if (Object.hasOwn(stored, path)) return Boolean(stored[path]);
  return index > 0;
}

function appendManagedFile(parent, project, file) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'file' + (selected && file.path === selected.path ? ' selected' : '');
  btn.dataset.testid = 'managed-file';
  btn.textContent = fileLabel(file.rel, file.name);
  btn.addEventListener('click', () => openFile(project, file));
  parent.append(btn);
}

function appendFileGroup(parent, project, dir, files) {
  const key = folderStorageKey(project.path, dir);
  const stored = readJSON(TREE_FOLDERS_KEY, {});
  const collapsed = Boolean(dir && stored[key]);
  let body = parent;
  if (dir) {
    const folder = document.createElement('div');
    folder.className = 'tree-folder' + (collapsed ? ' collapsed' : '');
    const toggle = document.createElement('button');
    toggle.type = 'button';
    toggle.className = 'tree-folder-toggle';
    toggle.dataset.testid = 'tree-folder';
    toggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    toggle.append(icon('chevron'), dir);
    toggle.addEventListener('click', () => {
      const next = readJSON(TREE_FOLDERS_KEY, {});
      if (collapsed) {
        delete next[key];
      } else {
        next[key] = true;
      }

      localStorage.setItem(TREE_FOLDERS_KEY, JSON.stringify(next));
      renderTree();
    });
    folder.append(toggle);
    body = document.createElement('div');
    body.className = 'files';
    folder.append(body);
    parent.append(folder);
    if (collapsed) return;
  }

  const showAll = treeShowAll.has(key);
  const visible = showAll ? files : files.slice(0, TREE_LIMIT);
  for (const file of visible) appendManagedFile(body, project, file);
  if (!showAll && files.length > TREE_LIMIT) {
    const more = document.createElement('button');
    more.type = 'button';
    more.className = 'tree-show-more';
    more.dataset.testid = 'tree-show-more';
    more.textContent = `Show ${files.length - TREE_LIMIT} more`;
    more.addEventListener('click', () => {
      treeShowAll.add(key);
      renderTree();
    });
    body.append(more);
  }
}

function renderRecents(nav) {
  const items = readJSON(RECENTS_KEY, []).filter((item) => item?.path && item?.name);
  if (items.length === 0) return;
  const wrap = document.createElement('div');
  wrap.className = 'recents';
  wrap.dataset.testid = 'recents';
  const title = document.createElement('div');
  title.className = 'kicker';
  title.textContent = 'Recents';
  wrap.append(title);
  for (const item of items) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'recent-project';
    btn.dataset.testid = 'recent-project';
    btn.textContent = item.name;
    btn.addEventListener('click', () => addProjectFromPath(item.path));
    wrap.append(btn);
  }

  nav.append(wrap);
}

function renderTree() {
  const nav = treeEl();
  nav.replaceChildren();
  renderRecents(nav);
  for (const [index, project] of projects.entries()) {
    const collapsed = projectCollapsed(project.path, index);
    const wrap = document.createElement('div');
    wrap.className = 'tree-project' + (collapsed ? ' collapsed' : '');
    const title = document.createElement('button');
    title.type = 'button';
    title.className = 'project';
    title.dataset.testid = 'tree-project';
    title.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    title.append(icon('chevron'), project.name);
    const hint = document.createElement('span');
    hint.className = 'project-path';
    hint.textContent = parentLabel(project.path);
    title.append(hint);
    title.addEventListener('click', () => {
      const next = readJSON(TREE_PROJECTS_KEY, {});
      next[project.path] = !collapsed;
      localStorage.setItem(TREE_PROJECTS_KEY, JSON.stringify(next));
      renderTree();
    });
    wrap.append(title);
    if (!collapsed) {
      const files = document.createElement('div');
      files.className = 'files';
      const groups = groupFiles(project.files || []);
      const dirs = [...groups.keys()].sort((a, b) => a.localeCompare(b));
      for (const dir of dirs) {
        appendFileGroup(files, project, dir, groups.get(dir));
      }

      wrap.append(files);
    }

    nav.append(wrap);
  }
}

async function openFile(project, file) {
  selected = { project, ...file };
  pendingPaste = null;
  revealed = false;
  commitAuto = true;
  lastAuto = '';
  commitEl().value = '';
  renderTree();
  crumbEl().textContent = displayPath(project, file);
  headlineEl().textContent = titleOf(file.name);
  showError('');
  badgeEl().hidden = false;
  document.getElementById('meta-path').textContent = displayPath(project, file);
  document.getElementById('meta-format').textContent = formatOf(file.path);
  document.getElementById('meta-enc').textContent = 'age + SOPS';
  try {
    const pairs = await invoke('get_managed_file', { path: file.path });
    rows = pairs.map((p) => ({
      key: p.key,
      value: p.value,
      origKey: p.key,
      origValue: p.value,
      added: false,
      deleted: false,
      revealed: false,
    }));
    sublineEl().textContent = `${rows.length} secrets · never uploaded`;
    setFileNote(file.name === 'eas.json' ? 'eas.json: EAS CLI will not read SOPS ciphertext' : '');
    renderKeys();
    await loadPublishMapping(file.path);
  } catch (err) {
    rows = [];
    keysEl().replaceChildren();
    saveEl().disabled = true;
    showEmpty('');
    showError(messageOf(err));
  }
}

function renderKeys() {
  if (!selected) {
    keysEl().hidden = true;
    renderWorkspace();
    return;
  }

  const visible = visibleRows();
  showEmpty(visible.length === 0 ? 'No keys in this file.' : '');
  const box = keysEl();
  box.hidden = false;
  box.replaceChildren();
  renderPasteChrome(box);
  const head = document.createElement('div');
  head.className = 'key-head';
  const keyHead = document.createElement('span');
  keyHead.textContent = 'Key';
  const valueHead = document.createElement('span');
  valueHead.className = 'value-head';
  valueHead.append('Value');
  valueHead.append(
    iconButton(
      'reveal',
      revealed ? 'Hide values' : 'Reveal values',
      revealed ? 'eye-off' : 'eye',
      toggleReveal,
    ),
  );
  const typeHead = document.createElement('span');
  typeHead.textContent = 'Type';
  const actionsHead = document.createElement('span');
  head.append(keyHead, valueHead, typeHead, actionsHead);

  box.append(head);

  for (const row of visible) {
    const line = document.createElement('div');
    const changed = rowDirty(row);
    line.className = 'key-row' + (changed ? ' changed' : '');
    line.dataset.testid = 'key-row';
    const keyCell = document.createElement('div');
    keyCell.className = 'key-cell';
    const name = document.createElement('input');
    name.className = 'key-name';
    name.dataset.testid = 'key-name';
    name.value = row.key;
    name.autocomplete = 'off';
    name.spellcheck = false;
    const kind = document.createElement('span');
    kind.className = 'kind';
    kind.textContent = changed ? 'changed' : 'secret';
    const mark = () => {
      line.classList.toggle('changed', rowDirty(row));
      kind.textContent = rowDirty(row) ? 'changed' : 'secret';
      saveEl().disabled = dirtyCount() === 0;
      syncCommitMessage();
    };

    name.addEventListener('input', () => {
      row.key = name.value;
      mark();
    });
    keyCell.append(
      name,
      iconButton('copy-key', 'Copy key', 'copy', () => copyText(row.key)),
    );
    const valueCell = document.createElement('div');
    valueCell.className = 'value-cell';
    const input = document.createElement('input');
    const shown = Boolean(row.revealed);
    input.className = 'value' + (shown ? '' : ' masked');
    input.dataset.testid = 'key-value';
    input.value = shown ? row.value : '••••••••••••••••••••••••••••';
    input.readOnly = !shown;
    input.autocomplete = 'off';
    input.spellcheck = false;
    input.addEventListener('input', () => {
      row.value = input.value;
      mark();
    });
    valueCell.append(input);
    const actions = document.createElement('div');
    actions.className = 'key-actions';
    const remove = iconButton('delete-key', 'Delete key', 'trash', () => {
      if (row.added && !row.origKey) {
        rows = rows.filter((item) => item !== row);
      } else {
        row.deleted = true;
      }

      renderKeys();
    });
    remove.classList.add('danger');
    actions.append(
      iconButton(
        'reveal-key',
        shown ? 'Hide value' : 'Reveal value',
        shown ? 'eye-off' : 'eye',
        () => {
          row.revealed = !row.revealed;
          renderKeys();
        },
      ),
      iconButton('copy-value', 'Copy value', 'copy', () => copyText(row.value)),
      remove,
    );
    line.append(keyCell, valueCell, kind, actions);
    box.append(line);
  }

  const composer = document.createElement('div');
  composer.className = 'key-composer';
  const composerInput = document.createElement('input');
  composerInput.type = 'text';
  composerInput.dataset.testid = 'key-composer';
  composerInput.placeholder = 'KEY=value';
  composerInput.autocomplete = 'off';
  composerInput.spellcheck = false;
  const addFromComposer = () => {
    const parsed = parseComposerLine(composerInput.value);
    if (!parsed) return;
    rows.push({
      key: parsed.key,
      value: parsed.value,
      origKey: '',
      origValue: '',
      added: true,
      deleted: false,
      revealed: true,
    });
    composerFocus = true;
    renderKeys();
  };

  composerInput.addEventListener('keydown', (event) => {
    if (event.key !== 'Enter') return;
    event.preventDefault();
    addFromComposer();
  });
  composer.append(composerInput, iconButton('composer-add', 'Add key', 'plus', addFromComposer));
  box.append(composer);
  saveEl().disabled = dirtyCount() === 0;
  syncCommitMessage();
  sublineEl().textContent = `${visible.length} secrets · ${dirtyCount() ? 'edited locally' : 'never uploaded'}`;
  if (composerFocus) {
    composerFocus = false;
    composerInput.focus();
  }
}

async function addProjectFromPath(path, opts = {}) {
  const select = opts.select !== false;
  const files = await invoke('list_managed_files', { path });
  const name = path.split('/').findLast(Boolean) || path;
  const existing = projects.findIndex((p) => p.path === path);
  const project = { name, path, files };
  rememberRecent(project);
  if (existing === -1) {
    projects.push(project);
  } else {
    projects[existing] = project;
  }

  renderTree();
  if (files[0] && select) {
    await openFile(project, files[0]);
    return;
  }

  if (select && !files[0]) {
    selected = null;
    renderWorkspace();
  }
}

async function addProject() {
  showError('');
  try {
    const selectedPath = await invoke('pick_project_folder');
    if (!selectedPath) return;
    await addProjectFromPath(selectedPath);
  } catch (err) {
    showError(messageOf(err));
  }
}

function buttonLabel(el) {
  return el.querySelector('.btn-label') || el;
}

function decorateButton(id, kind) {
  const el = document.getElementById(id);
  if (!el || el.querySelector('.btn-label')) return;
  const label = document.createElement('span');
  label.className = 'btn-label';
  label.textContent = el.textContent.trim();
  el.replaceChildren(icon(kind), label);
  el.classList.add('has-icon');
}

function decorateChrome() {
  decorateButton('add-file', 'file');
  decorateButton('whats-new', 'spark');
  decorateButton('add-project', 'folder');
  decorateButton('grant-access', 'grant');
  decorateButton('remove-access', 'drop');
  decorateButton('publish', 'review');
  decorateButton('publish-yes', 'publish');
  decorateButton('commit', 'commit');
  decorateButton('sync', 'sync');
  decorateButton('review', 'review');
  decorateButton('history', 'history');
  decorateButton('restore', 'restore');
  decorateButton('save', 'save');
}

async function withBusy(el, label, fn) {
  const textEl = buttonLabel(el);
  const origText = textEl.textContent;
  const origDisabled = el.disabled;
  el.setAttribute('aria-busy', 'true');
  el.disabled = true;
  if (label) textEl.textContent = label;
  try {
    return await fn();
  } finally {
    el.removeAttribute('aria-busy');
    textEl.textContent = origText;
    el.disabled = origDisabled;
  }
}

async function saveFile() {
  if (!selected) return;
  showError('');
  const toDelete = rows.filter((r) => r.deleted && r.origKey);
  const live = rows.filter((r) => !r.deleted);
  const toSet = live.filter(
    (r) => r.key && (r.key !== r.origKey || r.value !== r.origValue || r.added),
  );
  try {
    await withBusy(saveEl(), 'Saving…', async () => {
      for (const row of toDelete) {
        await invoke('del_managed_key', { path: selected.path, key: row.origKey });
      }

      for (const row of toSet) {
        const renamed = Boolean(row.origKey) && row.key !== row.origKey;
        if (renamed) {
          await invoke('del_managed_key', { path: selected.path, key: row.origKey });
        }

        await invoke('set_managed_key', { path: selected.path, key: row.key, value: row.value });
        row.origKey = row.key;
        row.origValue = row.value;
        row.added = false;
      }

      rows = live;
    });

    renderKeys();
  } catch (err) {
    showError(messageOf(err), 'save');
    saveEl().disabled = dirtyCount() === 0;
  }
}

function currentProject() {
  return selected?.project ?? projects[0];
}

async function addManagedFile() {
  const project = currentProject();
  showError('');
  if (!project) {
    showError('Open a Project first');
    return;
  }

  const name = document.getElementById('add-file-name').value;
  try {
    const abs = resolveManagedPath(project.path, name);
    await invoke('create_managed_file', { path: abs });
    const files = await invoke('list_managed_files', { path: project.path });
    project.files = files;
    const idx = projects.findIndex((item) => item.path === project.path);
    if (idx !== -1) projects[idx] = project;
    renderTree();
    const rel = String(name || '')
      .trim()
      .replaceAll('\\', '/');
    const created =
      files.find((file) => file.path === abs) ||
      files.find((file) => safeRel(file.rel, file.name) === rel);
    if (created) {
      await openFile(project, created);
    }

    document.getElementById('add-file-name').value = '';
  } catch (err) {
    showError(messageOf(err));
  }
}

async function loadPublishMapping(path) {
  const prefix = document.getElementById('publish-prefix');
  const repo = document.getElementById('publish-repo');
  const environment = document.getElementById('publish-environment');
  try {
    const mapping = await invoke('get_publish_mapping', { path });
    prefix.value = mapping?.prefix || '';
    repo.textContent = mapping?.repo || '—';
    environment.textContent = mapping?.environment || '—';
  } catch {
    prefix.value = '';
    repo.textContent = '—';
    environment.textContent = '—';
  }
}

async function publishFile(yes) {
  if (!selected) return;
  showError('');
  setStatus('publish', '');
  const btn = document.getElementById(yes ? 'publish-yes' : 'publish');
  const prefix = document.getElementById('publish-prefix').value.trim();
  const prune = document.getElementById('publish-prune').checked;
  try {
    const result = await withBusy(btn, yes ? 'Publishing…' : 'Checking…', () =>
      invoke('publish_managed_file', {
        path: selected.path,
        prefix,
        yes,
        prune,
      }),
    );
    setStatus('publish', String(result || '').trim());
  } catch (err) {
    showError(messageOf(err), 'publish');
  }
}

async function loadDemoHints() {
  try {
    const response = await fetch('/demo');
    if (!response.ok) return;
    const info = await response.json();
    const input = document.getElementById('recipient-key');
    if (input && info.bobPublicKey) input.value = info.bobPublicKey;
    const extras = Array.isArray(info.projects) ? info.projects : [];
    for (const path of extras) {
      if (!path || projects.some((p) => p.path === path)) continue;
      await addProjectFromPath(path, { select: false });
    }
  } catch {
    // Tauri has no /demo endpoint.
  }
}

async function loadWhatsNew() {
  if (globalThis.__TAURI__?.core?.invoke) {
    return invoke('whats_new');
  }

  const response = await fetch('/whats-new.json');
  if (!response.ok) {
    throw new Error(response.statusText);
  }

  return response.json();
}

async function showWhatsNew() {
  const dialog = document.getElementById('whats-new-dialog');
  try {
    const payload = await loadWhatsNew();
    document.getElementById('whats-new-heading').textContent = payload.heading || "What's new";
    const versionBits = [];
    if (payload.version) versionBits.push(`Sopsdeck ${payload.version}`);
    if (payload.date) versionBits.push(payload.date);
    document.getElementById('whats-new-version').textContent = versionBits.join(' · ');
    const list = document.getElementById('whats-new-list');
    list.replaceChildren();
    for (const note of payload.notes || []) {
      const item = document.createElement('li');
      item.className = 'whats-new-item';
      const text = typeof note === 'string' ? note : note.text;
      const type = typeof note === 'string' ? '' : note.type;
      const platforms = typeof note === 'string' ? [] : note.platforms || [];
      if (type) {
        const tag = document.createElement('span');
        tag.className = 'note-tag';
        tag.dataset.testid = 'whats-new-tag';
        tag.textContent =
          {
            feature: 'Feature',
            bugfix: 'Bug fix',
            performance: 'Performance',
            changed: 'Changed',
            removed: 'Removed',
            security: 'Security',
          }[type] || type;
        item.append(tag);
      }

      const body = document.createElement('span');
      body.className = 'whats-new-text';
      body.textContent = text;
      item.append(body);
      for (const name of platforms) {
        const plat = document.createElement('span');
        plat.className = 'note-platform';
        plat.dataset.testid = 'whats-new-platform';
        plat.textContent = name;
        item.append(plat);
      }

      list.append(item);
    }

    dialog.showModal();
  } catch (err) {
    showError(messageOf(err));
  }
}

window.addEventListener('DOMContentLoaded', async () => {
  decorateChrome();
  initInspector();
  document.addEventListener('paste', onEditorPaste);
  applyTheme(currentTheme());
  document.getElementById('whats-new').addEventListener('click', () => {
    showWhatsNew();
  });
  document.getElementById('whats-new-close').addEventListener('click', () => {
    document.getElementById('whats-new-dialog').close();
  });
  document.getElementById('theme-toggle').addEventListener('click', () => {
    applyTheme(currentTheme() === 'dark' ? 'light' : 'dark');
  });
  commitEl().addEventListener('input', () => {
    const { value } = commitEl();
    commitAuto = value === '' || value === lastAuto;
  });
  document.getElementById('add-project').addEventListener('click', addProject);
  document.getElementById('add-file').addEventListener('click', addManagedFile);
  document.getElementById('add-file-name').addEventListener('keydown', (event) => {
    if (event.key !== 'Enter') return;
    event.preventDefault();
    addManagedFile();
  });
  saveEl().addEventListener('click', saveFile);
  document.getElementById('commit').addEventListener('click', async () => {
    if (!selected) return;
    if (dirtyCount()) {
      showError('Encrypt & save before commit', 'git');
      return;
    }

    showError('');
    try {
      const message = commitEl().value;
      await invoke('commit_managed_file', { path: selected.path, message });
      commitEl().value = '';
      commitAuto = true;
      lastAuto = '';
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  document.getElementById('sync').addEventListener('click', async () => {
    if (!selected) return;
    showError('');
    try {
      await withBusy(document.getElementById('sync'), 'Syncing…', () =>
        invoke('sync_project', { path: selected.path }),
      );
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  document.getElementById('review').addEventListener('click', async () => {
    if (!selected) return;
    showError('');
    setStatus('git', '');
    try {
      const text = await withBusy(document.getElementById('review'), 'Reviewing…', () =>
        invoke('review_managed_file', { path: selected.path }),
      );
      const out = document.getElementById('review-out');
      out.hidden = false;
      out.textContent = text && String(text).trim() ? text : 'No uncommitted secret changes';
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  document.getElementById('history').addEventListener('click', async () => {
    if (!selected) return;
    showError('');
    setStatus('git', '');
    try {
      const text = await withBusy(document.getElementById('history'), 'Loading…', () =>
        invoke('history_managed_file', { path: selected.path }),
      );
      const list = document.getElementById('history-list');
      list.hidden = false;
      list.replaceChildren();
      historyRev = '';
      for (const line of String(text || '')
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean)) {
        const space = line.indexOf(' ');
        const rev = space === -1 ? line : line.slice(0, space);
        const subject = space === -1 ? line : line.slice(space + 1);
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.textContent = line;
        btn.dataset.rev = rev;
        btn.addEventListener('click', async () => {
          historyRev = rev;
          for (const other of list.querySelectorAll('button')) {
            other.removeAttribute('aria-current');
          }

          btn.setAttribute('aria-current', 'true');

          try {
            const pairs = await invoke('get_managed_file', { path: selected.path, at: rev });
            const out = document.getElementById('review-out');
            out.hidden = false;
            out.textContent = (pairs || [])
              .filter((p) => p.key && p.key !== 'sops' && !String(p.key).startsWith('sops_'))
              .map((p) => `${p.key}=${p.value}`)
              .join('\n');
            setStatus('git', subject);
          } catch (err) {
            showError(messageOf(err), 'git');
          }
        });
        list.append(btn);
      }
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  document.getElementById('restore').addEventListener('click', async () => {
    if (!selected) return;
    if (!historyRev) {
      showError('Pick a revision from History', 'git');
      return;
    }

    showError('');
    try {
      await withBusy(document.getElementById('restore'), 'Restoring…', () =>
        invoke('restore_managed_file', { path: selected.path, at: historyRev }),
      );
      await openFile(selected.project, selected);
      setStatus('git', 'Restored. Commit to keep it.');
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  document.getElementById('grant-access').addEventListener('click', async () => {
    if (!selected) return;
    const key = document.getElementById('recipient-key').value.trim();
    showError('');
    setStatus('access', '');
    try {
      await withBusy(document.getElementById('grant-access'), 'Granting…', () =>
        invoke('add_recipient', { path: selected.path, publicKey: key }),
      );
      setStatus('access', 'Access granted');
    } catch (err) {
      showError(messageOf(err), 'access');
    }
  });
  document.getElementById('remove-access').addEventListener('click', async () => {
    if (!selected) return;
    const key = document.getElementById('recipient-key').value.trim();
    showError('');
    setStatus('access', '');
    try {
      const note = await withBusy(document.getElementById('remove-access'), 'Removing…', () =>
        invoke('remove_recipient', { path: selected.path, publicKey: key }),
      );
      const text =
        typeof note === 'string' && note
          ? note.replace(/^recipient remove:\s*/u, '')
          : 'Access dropped. Git history and copies they already have still decrypt.';
      setStatus('access', text);
    } catch (err) {
      showError(messageOf(err), 'access');
    }
  });
  document.getElementById('publish').addEventListener('click', () => publishFile(false));
  document.getElementById('publish-yes').addEventListener('click', () => publishFile(true));
  renderWorkspace();
  if (skipBoot()) return;
  try {
    const boot = await invoke('boot_project');
    if (boot) await addProjectFromPath(boot);
    await loadDemoHints();
  } catch (err) {
    showError(messageOf(err));
  }
});
