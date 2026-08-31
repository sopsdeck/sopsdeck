import {
  classifyPasteKeys,
  parseGitIdentity,
  parsePastePayload,
  pastePreviewText,
} from './paste.js';
import { resetClipboardSeen, sniffClipboard } from './clipboard.js';
import { icon, iconButton } from './icons.js';
import { showWhatsNew } from './whatsnew.js';

const invoke = invokeOverHTTP;
const THEME_KEY = 'sopsdeck-theme';
const INSPECTOR_KEY = 'sopsdeck-inspector';
const RECENTS_KEY = 'sopsdeck-recents';
const TREE_FOLDERS_KEY = 'sopsdeck-tree-folders';
const TREE_PROJECTS_KEY = 'sopsdeck-tree-projects';
const TREE_LIMIT = 8;
let unusedKeys = new Set();

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
const fileLockEl = () => document.getElementById('file-lock');
const copyFileEl = () => document.getElementById('copy-file');
const fileHistoryEl = () => document.getElementById('file-history');

const projects = [];
let selected = null;
let rows = [];
let revealed = false;
let commitAuto = true;
let lastAuto = '';
let access = [];
let accessFormOpen = false;
let focusedProject = false;
let canGrant = true;
let projectConfig = { path: '', name: '', owners: [], canGrant: true };
let account = {
  name: '',
  email: '',
  publicKey: '',
  hasIdentity: false,
};
let integration = {
  scope: 'repo',
  repo: '',
  org: '',
  environment: '',
  prefix: '',
  visibility: 'all',
};
let composerFocus = false;
let pendingPaste = null;
const treeShowAll = new Set();

function messageOf(err) {
  if (err instanceof Error) return err.message;
  return String(err);
}

function jsonValue(obj, key, fallback = '') {
  if (!obj || !Object.hasOwn(obj, key)) return fallback;
  return obj[key];
}

function accountFrom(raw = {}) {
  return {
    name: raw.name || '',
    email: raw.email || '',
    publicKey: raw.publicKey || jsonValue(raw, 'public_key'),
    hasIdentity: Boolean(raw.hasIdentity ?? jsonValue(raw, 'has_identity')),
  };
}

function withDialog(dialog, setup) {
  return new Promise((resolve) => {
    const ac = new AbortController();
    const { signal } = ac;
    let finished = false;
    const finish = (value) => {
      if (finished) return;
      finished = true;
      ac.abort();
      if (dialog.open) dialog.close();
      resolve(value);
    };

    setup({ signal, finish });
  });
}

function skipBoot() {
  return new URLSearchParams(location.search).has('empty');
}

async function copyText(text) {
  const value = String(text);
  let nativeError;
  try {
    await invoke('copy_text', { text: value });
    return true;
  } catch (err) {
    nativeError = err;
    // The server-side clipboard command may be unavailable in a browser-only setup.
  }

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value);
      return true;
    }

    const area = document.createElement('textarea');
    area.value = value;
    area.setAttribute('readonly', '');
    area.style.position = 'fixed';
    area.style.opacity = '0';
    document.body.append(area);
    try {
      area.select();
      if (!document.execCommand('copy')) throw new Error('Clipboard access is unavailable');
      return true;
    } finally {
      area.remove();
    }
  } catch {
    const detail = nativeError ? `: ${messageOf(nativeError)}` : '';
    showError(`Could not copy to the clipboard${detail}`);
    return false;
  }
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

function refreshUnusedBadge(kind, row) {
  const existing = kind.querySelector('.unused');
  if (row.unused && !row.added && !row.deleted) {
    if (existing) return;
    const badge = document.createElement('span');
    badge.className = 'unused';
    badge.textContent = 'unused';
    kind.append(badge);
  } else if (existing) {
    existing.remove();
  }
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

function removeProject(project) {
  const index = projects.findIndex((item) => item.path === project.path);
  if (index === -1) return;
  projects.splice(index, 1);
  localStorage.setItem(
    RECENTS_KEY,
    JSON.stringify(readJSON(RECENTS_KEY, []).filter((item) => item?.path !== project.path)),
  );
  if (selected?.project.path === project.path) {
    selected = null;
    rows = [];
    resetEditorChrome();
  }

  renderTree();
  renderWorkspace();
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
  if (added.length > 0) parts.push(`add ${added.join(', ')}`);
  if (changed.length > 0) parts.push(`update ${changed.join(', ')}`);
  if (removed.length > 0) parts.push(`remove ${removed.join(', ')}`);
  const file = selected
    ? `${selected.project?.name || 'project'}/${selected.name || selected.rel || 'managed file'}`
    : 'managed file';
  return parts.length > 0 ? `secrets(${file}): ${parts.join('; ')}` : '';
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
  fileLockEl().disabled = true;
  setFileLockState(true);
  copyFileEl().disabled = true;
  fileHistoryEl().disabled = true;
  access = [];
  accessFormOpen = false;
  integration = {
    scope: 'repo',
    repo: '',
    org: '',
    environment: '',
    prefix: '',
    visibility: 'all',
  };
  renderIntegrationSummary();
  renderAccess();
  setStatus('file', '');
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
  setRevealed(!revealed);
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
  if (focusedProject) return;
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
  const kicker = document.getElementById('sidebar-kicker');
  if (kicker) kicker.textContent = focusedProject ? 'Project' : 'Projects';
  renderRecents(nav);
  for (const [index, project] of projects.entries()) {
    const collapsed = projectCollapsed(project.path, index);
    const wrap = document.createElement('div');
    wrap.className = 'tree-project' + (collapsed ? ' collapsed' : '');
    const header = document.createElement('div');
    header.className = 'project-header';
    const title = document.createElement('button');
    title.type = 'button';
    title.className = 'project';
    title.dataset.testid = 'tree-project';
    title.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    const copy = document.createElement('span');
    copy.className = 'project-copy';
    const name = document.createElement('span');
    name.className = 'project-name';
    name.textContent = project.name;
    const hint = document.createElement('span');
    hint.className = 'project-path';
    hint.textContent = parentLabel(project.path);
    copy.append(name, hint);
    title.append(icon('chevron'), copy);
    title.addEventListener('click', () => {
      const next = readJSON(TREE_PROJECTS_KEY, {});
      next[project.path] = !collapsed;
      localStorage.setItem(TREE_PROJECTS_KEY, JSON.stringify(next));
      renderTree();
    });
    header.append(title);
    if (!focusedProject) {
      const actions = document.createElement('span');
      actions.className = 'project-actions';
      const move = iconButton('move-project', 'Choose a new Project path', 'folder', async () => {
        const path = pickProjectFolder();
        if (!path || path === project.path) return;
        await addProjectFromPath(path);
        removeProject(project);
      });
      const remove = iconButton('remove-project', 'Remove Project', 'trash', () =>
        removeProject(project),
      );
      actions.append(move, remove);
      header.append(actions);
    }

    wrap.append(header);
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
  access = [];
  accessFormOpen = false;
  renderAccess();
  revealed = false;
  commitAuto = true;
  lastAuto = '';
  commitEl().value = '';
  renderTree();
  crumbEl().textContent = displayPath(project, file);
  headlineEl().textContent = titleOf(file.name);
  showError('');
  setStatus('file', '');
  badgeEl().hidden = true;
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
      unused: false,
    }));
    fileLockEl().disabled = false;
    copyFileEl().disabled = false;
    fileHistoryEl().disabled = false;
    const status = await invoke('get_managed_file_status', { path: file.path });
    setFileLockState(Boolean(status.locked));
    document.getElementById('meta-enc').textContent = status.locked
      ? 'age + SOPS (locked)'
      : 'plaintext (unlocked)';
    sublineEl().textContent = `${rows.length} secrets · never uploaded`;
    setFileNote(file.name === 'eas.json' ? 'eas.json: EAS CLI will not read SOPS ciphertext' : '');
    renderKeys();
    await Promise.all([
      loadPublishMapping(file.path),
      loadAccess(file.path),
      loadUnusedKeys(file.path),
      loadProjectConfig(project.path),
    ]);
  } catch (err) {
    rows = [];
    keysEl().replaceChildren();
    saveEl().disabled = true;
    showEmpty('');
    showError(messageOf(err));
  }
}

async function copyFileContents() {
  if (!selected) return;
  try {
    const contents = await invoke('get_managed_file_contents', { path: selected.path });
    if (await copyText(contents)) setStatus('file', 'Copied unencrypted contents');
  } catch (err) {
    showError(messageOf(err));
  }
}

function setFileLockState(locked) {
  const button = fileLockEl();
  if (!button) return;
  button.replaceChildren(icon(locked ? 'unlock' : 'lock'));
  const action = locked ? 'Unlock' : 'Lock';
  button.setAttribute('aria-label', `${action} file`);
  button.title = `${action} file`;
  button.dataset.locked = locked ? 'true' : 'false';
  const badge = badgeEl();
  if (!badge) return;
  if (!selected) {
    badge.hidden = true;
    return;
  }

  badge.hidden = false;
  badge.textContent = locked ? 'Locked' : 'Unlocked';
  badge.classList.toggle('unlocked', !locked);
}

function setRevealed(value) {
  revealed = value;
  for (const row of rows) row.revealed = revealed;
  renderKeys();
}

async function unlockFileOnDisk() {
  if (!selected) return;
  if (dirtyCount()) {
    showError('Encrypt & save before Unlock');
    return;
  }

  showError('');
  try {
    await withBusy(fileLockEl(), 'Unlocking…', () =>
      invoke('unlock_managed_file', { path: selected.path }),
    );
    await openFile(selected.project, selected);
    setRevealed(true);
    setStatus('file', 'Plaintext is now on disk');
  } catch (err) {
    showError(messageOf(err));
  }
}

async function lockFileOnDisk() {
  if (!selected) return;
  if (dirtyCount()) {
    showError('Encrypt & save before Lock');
    return;
  }

  showError('');
  try {
    await withBusy(fileLockEl(), 'Locking…', () =>
      invoke('lock_managed_file', { path: selected.path }),
    );
    await openFile(selected.project, selected);
    setRevealed(false);
    setStatus('file', 'Encrypted on disk');
  } catch (err) {
    showError(messageOf(err));
  }
}

function toggleFileLock() {
  if (fileLockEl().dataset.locked === 'true') {
    return unlockFileOnDisk();
  }

  return lockFileOnDisk();
}

function historyItems(text) {
  return String(text || '')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const space = line.indexOf(' ');
      return {
        rev: space === -1 ? line : line.slice(0, space),
        subject: space === -1 ? line : line.slice(space + 1),
      };
    });
}

async function showFileHistory() {
  if (!selected) return;
  const dialog = document.getElementById('file-history-dialog');
  const list = document.getElementById('file-history-list');
  const preview = document.getElementById('file-history-preview');
  list.replaceChildren();
  preview.hidden = true;
  dialog.showModal();
  try {
    const items = historyItems(await invoke('history_managed_file', { path: selected.path }));
    if (items.length === 0) {
      const empty = document.createElement('li');
      empty.className = 'save-note';
      empty.textContent = 'No Git history for this file yet.';
      list.append(empty);
      return;
    }

    for (const item of items) {
      const li = document.createElement('li');
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = `${item.rev} ${item.subject}`;
      button.addEventListener('click', async () => {
        try {
          const pairs = await invoke('get_managed_file', { path: selected.path, at: item.rev });
          preview.hidden = false;
          preview.textContent =
            pairs.map((pair) => `${pair.key}=${pair.value}`).join('\n') ||
            'No secrets in this revision';
        } catch (err) {
          showError(messageOf(err));
        }
      });
      li.append(button);
      list.append(li);
    }
  } catch (err) {
    const error = document.createElement('li');
    error.className = 'control-error';
    error.textContent = messageOf(err);
    list.append(error);
  }
}

async function showSecretHistory(row) {
  if (!selected) return;
  const dialog = document.getElementById('secret-history-dialog');
  const list = document.getElementById('secret-history-list');
  const preview = document.getElementById('secret-history-preview');
  document.getElementById('secret-history-heading').textContent = row.key;
  list.replaceChildren();
  preview.hidden = true;
  dialog.showModal();
  try {
    const items = historyItems(await invoke('history_managed_file', { path: selected.path }));
    for (const item of items) {
      const li = document.createElement('li');
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = `${item.rev} ${item.subject}`;
      button.addEventListener('click', async () => {
        try {
          const pairs = await invoke('get_managed_file', { path: selected.path, at: item.rev });
          const match = pairs.find((pair) => pair.key === row.key);
          preview.hidden = false;
          preview.textContent = match
            ? `${item.rev}\n${row.key}=${match.value}`
            : `${item.rev}\n${row.key} was not present`;
        } catch (err) {
          showError(messageOf(err));
        }
      });
      li.append(button);
      list.append(li);
    }

    if (items.length === 0) {
      const empty = document.createElement('li');
      empty.className = 'save-note';
      empty.textContent = 'No Git history for this file yet.';
      list.append(empty);
    }
  } catch (err) {
    const error = document.createElement('li');
    error.className = 'control-error';
    error.textContent = messageOf(err);
    list.append(error);
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
    refreshUnusedBadge(kind, row);

    const mark = () => {
      line.classList.toggle('changed', rowDirty(row));
      kind.textContent = rowDirty(row) ? 'changed' : 'secret';
      refreshUnusedBadge(kind, row);

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
      iconButton('secret-history', 'Secret history', 'history', () => showSecretHistory(row)),
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
  let state = await invoke('inspect_project', { path });
  await ensureAccount(path);
  if (!state.initialized) {
    const selectedFiles = await chooseProjectFiles(path, state.candidates || []);
    if (selectedFiles) {
      await invoke('initialize_project', { path, files: selectedFiles });
      state = await invoke('inspect_project', { path });
    }
  }

  const files = state.managed || [];
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

  await loadProjectConfig(path);
}

function avatarURL(kind, seed) {
  const style = kind === 'robot' ? 'voxel-bot' : 'initials';
  return `https://api.dicebear.com/10.x/${style}/svg?seed=${encodeURIComponent(seed || 'Unknown')}`;
}

function renderAvatar(element, kind, seed, alt) {
  element.replaceChildren();
  const image = document.createElement('img');
  image.src = avatarURL(kind, seed);
  image.alt = alt;
  image.loading = 'lazy';
  element.append(image);
}

function renderAccount() {
  const label = account.name || account.email || 'Set up account';
  const button = document.getElementById('account');
  if (button) {
    button.title = `Open account · ${label}`;
    button.setAttribute('aria-label', `Open account · ${label}`);
  }

  const seed = account.name || account.email || 'Account';
  const avatar = document.getElementById('account-avatar');
  if (avatar) renderAvatar(avatar, 'person', seed, label);
  const accountLabel = document.getElementById('account-label');
  if (accountLabel) accountLabel.textContent = label;
  const large = document.getElementById('account-avatar-large');
  if (large) renderAvatar(large, 'person', seed, label);
  const previewName = document.getElementById('account-preview-name');
  if (previewName) previewName.textContent = account.name || 'Not configured';
  const previewEmail = document.getElementById('account-preview-email');
  if (previewEmail) previewEmail.textContent = account.email || 'Configure your Git identity';
  renderAccountKey();
  renderProjectPanel();
}

function renderAccountKey() {
  const keyStatus = document.getElementById('account-key-status');
  if (keyStatus) keyStatus.textContent = account.hasIdentity ? 'Configured' : 'Not configured';
  const createIdentity = document.getElementById('account-create-identity');
  if (createIdentity) {
    createIdentity.disabled = account.hasIdentity;
    createIdentity.textContent = account.hasIdentity ? 'Identity configured' : 'Create identity';
  }

  const keyField = document.getElementById('account-key-field');
  const keyInput = document.getElementById('account-public-key');
  if (keyField) keyField.hidden = !account.hasIdentity || !account.publicKey;
  if (keyInput) keyInput.value = account.publicKey || '';
}

function accountComplete() {
  return Boolean(account.name && account.email && account.hasIdentity);
}

async function openAccountDialog(required = false, path = '') {
  const dialog = document.getElementById('account-dialog');
  const nameInput = document.getElementById('account-name');
  const emailInput = document.getElementById('account-email');
  const error = document.getElementById('account-error');
  const currentPath = path || selected?.project.path || projects[0]?.path || '';
  try {
    account = accountFrom(await invoke('get_account', { path: currentPath }));
  } catch {
    account = accountFrom();
  }

  nameInput.value = account.name;
  emailInput.value = account.email;
  nameInput.readOnly = Boolean(account.name);
  emailInput.readOnly = Boolean(account.email);
  error.hidden = true;
  document.getElementById('account-later').hidden = !required;
  const saveAccount = document.getElementById('account-save');
  saveAccount.hidden = Boolean(account.name && account.email);
  saveAccount.textContent =
    account.name || account.email ? 'Fill missing Git identity' : 'Save Git identity';
  renderAccount();
  dialog.showModal();
  return withDialog(dialog, ({ signal, finish }) => {
    dialog.addEventListener('cancel', () => finish(false), { signal });
    document.getElementById('account-cancel').addEventListener('click', () => finish(false), {
      signal,
    });
    document.getElementById('account-later').addEventListener('click', () => finish(false), {
      signal,
    });
    document.getElementById('account-save').addEventListener(
      'click',
      async () => {
        const name = nameInput.value.trim();
        const email = emailInput.value.trim();
        if (!name || !email) {
          error.hidden = false;
          error.textContent = 'Enter the Git name and email used for commits';
          return;
        }

        try {
          account = accountFrom(
            await invoke('configure_account', { path: currentPath, name, email }),
          );
          renderAccount();
          if (required && !accountComplete()) {
            error.hidden = false;
            error.textContent = 'Create an Age identity before managing encrypted files';
            return;
          }

          finish(true);
        } catch (err) {
          error.hidden = false;
          error.textContent = messageOf(err);
        }
      },
      { signal },
    );

    document.getElementById('account-create-identity').addEventListener(
      'click',
      async () => {
        // eslint-disable-next-line no-alert -- identity backup confirmation is a destructive step.
        if (!window.confirm('Save the backup in your password manager before continuing?')) return;
        try {
          account = accountFrom(await invoke('create_user_identity', { path: currentPath }));
          renderAccount();
        } catch (err) {
          error.hidden = false;
          error.textContent = messageOf(err);
        }
      },
      { signal },
    );
  });
}

async function ensureAccount(path) {
  try {
    account = accountFrom(await invoke('get_account', { path }));
  } catch {
    account = accountFrom();
  }

  renderAccount();
  if (!accountComplete()) await openAccountDialog(true, path);
}

function chosenKeys(keyInputs) {
  return keyInputs.filter((keyInput) => keyInput.checked).map((keyInput) => keyInput.value);
}

function chooseProjectFiles(path, candidates, opts = {}) {
  const dialog = document.getElementById('setup-project-dialog');
  const list = document.getElementById('setup-project-files');
  const error = document.getElementById('setup-project-error');
  const selection = document.getElementById('setup-project-selection');
  const selectAll = document.getElementById('setup-project-select-all');
  const ignoreAll = document.getElementById('setup-project-ignore-all');
  const rows = [];
  list.replaceChildren();
  error.hidden = true;

  const formatFor = (rel) => {
    const lower = rel.toLowerCase();
    if (lower === '.env' || lower.includes('.env')) return 'Environment variables';
    if (lower.endsWith('.json')) return 'JSON';
    if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return 'YAML';
    return 'Configuration file';
  };

  const updateSelection = () => {
    let fileCount = 0;
    let pathCount = 0;
    for (const { input, label, state, keyInputs } of rows) {
      const selectedKeys = keyInputs.filter((keyInput) => keyInput.checked);
      const managed = keyInputs.length > 0 ? selectedKeys.length > 0 : input.checked;
      if (managed) fileCount += 1;
      pathCount += selectedKeys.length;
      input.checked = managed;
      input.indeterminate =
        keyInputs.length > 0 && selectedKeys.length > 0 && selectedKeys.length < keyInputs.length;
      label.classList.toggle('is-managed', managed);
      state.textContent =
        keyInputs.length > 0
          ? managed && selectedKeys.length < keyInputs.length
            ? `${selectedKeys.length}/${keyInputs.length}`
            : managed
              ? 'Manage all'
              : 'Ignore'
          : managed
            ? 'Manage'
            : 'Ignore';
    }

    selection.textContent =
      fileCount === 0
        ? 'No paths selected'
        : `${fileCount} file${fileCount === 1 ? '' : 's'} · ${pathCount || 'all'} paths selected`;
  };

  for (const file of candidates) {
    const rel = file.rel || file.path;
    const entry = document.createElement('div');
    entry.className = 'setup-project-entry';
    const label = document.createElement('label');
    label.className = 'setup-project-file';
    const input = document.createElement('input');
    input.type = 'checkbox';
    input.value = rel;
    input.checked = opts.manageAll === true;
    input.dataset.testid = 'setup-project-file-toggle';
    const copy = document.createElement('span');
    copy.className = 'setup-project-file-copy';
    const name = document.createElement('strong');
    name.textContent = rel;
    const meta = document.createElement('small');
    const keys = Array.isArray(file.keys) ? file.keys : [];
    meta.textContent =
      keys.length > 0
        ? `${formatFor(rel)} · ${keys.length} selectable path${keys.length === 1 ? '' : 's'}`
        : formatFor(rel);
    copy.append(name, meta);
    const state = document.createElement('span');
    state.className = 'setup-project-file-state';
    label.append(input, copy, state);
    const keyInputs = [];
    const keyList = document.createElement('div');
    keyList.className = 'setup-project-keys';
    for (const key of keys) {
      const keyLabel = document.createElement('label');
      keyLabel.className = 'setup-project-key';
      const keyInput = document.createElement('input');
      keyInput.type = 'checkbox';
      keyInput.value = key;
      keyInput.checked = opts.manageAll === true;
      keyInput.dataset.testid = 'setup-project-key-toggle';
      const keyName = document.createElement('code');
      keyName.textContent = key;
      keyLabel.append(keyInput, keyName);
      keyInput.addEventListener('change', updateSelection);
      keyList.append(keyLabel);
      keyInputs.push(keyInput);
    }

    input.addEventListener('change', () => {
      for (const keyInput of keyInputs) {
        keyInput.checked = input.checked;
      }

      updateSelection();
    });
    entry.append(label);
    if (keys.length > 0) entry.append(keyList);
    list.append(entry);
    rows.push({ input, label, state, keyInputs });
  }

  if (candidates.length === 0) {
    const empty = document.createElement('p');
    empty.className = 'save-note';
    empty.textContent = 'No supported configuration files found. You can add one later.';
    list.append(empty);
  }

  dialog.querySelector('h2').textContent =
    opts.heading || `Initialize ${path.split('/').findLast(Boolean) || path}?`;
  document.getElementById('setup-project-skip').textContent =
    opts.skipLabel || 'Open without initializing';
  document.getElementById('setup-project-init').textContent = opts.action || 'Initialize Project';
  updateSelection();
  dialog.showModal();
  return withDialog(dialog, ({ signal, finish }) => {
    selectAll.addEventListener(
      'click',
      () => {
        for (const { input, keyInputs } of rows) {
          input.checked = true;
          for (const keyInput of keyInputs) {
            keyInput.checked = true;
          }
        }

        updateSelection();
      },
      { signal },
    );
    ignoreAll.addEventListener(
      'click',
      () => {
        for (const { input, keyInputs } of rows) {
          input.checked = false;
          for (const keyInput of keyInputs) {
            keyInput.checked = false;
          }
        }

        updateSelection();
      },
      { signal },
    );
    dialog.addEventListener('cancel', () => finish(null), { signal });
    document.getElementById('setup-project-init').addEventListener(
      'click',
      () => {
        finish(
          rows
            .map(({ input, keyInputs }) => ({
              path: input.value,
              hasPaths: keyInputs.length > 0,
              keys: chosenKeys(keyInputs),
            }))
            .filter(({ hasPaths, keys }) => !hasPaths || keys.length > 0)
            .map(({ path, keys }) => ({ path, keys })),
        );
      },
      { signal },
    );
    document.getElementById('setup-project-skip').addEventListener('click', () => finish(null), {
      signal,
    });
  });
}

async function addProject() {
  showError('');
  try {
    const selectedPath = pickProjectFolder();
    if (!selectedPath) return;
    await addProjectFromPath(selectedPath);
  } catch (err) {
    showError(messageOf(err));
  }
}

function pickProjectFolder() {
  // eslint-disable-next-line no-alert -- browsers have no native path picker for local servers.
  return window.prompt('Project folder path', '')?.trim() || null;
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
  decorateButton('save', 'save');
  decorateButton('grant-access', 'grant');
  decorateButton('request-access', 'grant');
  for (const [id, kind] of [
    ['file-lock', 'unlock'],
    ['copy-file', 'copy'],
    ['file-history', 'history'],
    ['create-robot', 'robot'],
    ['project-copy-key', 'copy'],
    ['project-account', 'grant'],
  ]) {
    const button = document.getElementById(id);
    if (button) button.append(icon(kind));
  }

  document.querySelector('#github-integration .integration-logo')?.append(icon('github'));
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

function changePreview(current) {
  const lines = [];
  for (const row of current) {
    if (row.deleted && row.origKey) lines.push(`Remove ${row.origKey}`);
    else if (row.added && row.key) lines.push(`Add ${row.key}`);
    else if (row.key !== row.origKey) lines.push(`Rename ${row.origKey} → ${row.key}`);
    else if (row.value !== row.origValue) lines.push(`Update ${row.key}`);
  }

  return lines.join('\n');
}

function confirmSave(current) {
  const dialog = document.getElementById('save-preview-dialog');
  const renames = current.filter((row) => row.origKey && row.key !== row.origKey && !row.deleted);
  const choices = document.getElementById('rename-reference-choices');
  choices.replaceChildren();
  choices.hidden = renames.length === 0;
  for (const row of renames) {
    const label = document.createElement('label');
    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.value = row.origKey;
    label.append(checkbox, ` Rewrite references from ${row.origKey} to ${row.key}`);
    choices.append(label);
  }

  document.getElementById('save-preview-copy').textContent =
    `${selected?.name || 'Managed file'} will be encrypted and committed to Git.`;
  document.getElementById('save-preview').textContent = changePreview(current);
  dialog.showModal();
  return withDialog(dialog, ({ signal, finish }) => {
    document.getElementById('save-preview-cancel').addEventListener('click', () => finish(null), {
      signal,
    });
    document.getElementById('save-preview-confirm').addEventListener(
      'click',
      () => {
        finish({
          rewriteRefs: new Set(
            [...choices.querySelectorAll('input:checked')].map((input) => input.value),
          ),
        });
      },
      { signal },
    );
  });
}

async function saveFile() {
  if (!selected) return;
  showError('');
  const current = rows;
  if (dirtyCount() === 0) return;
  const decision = await confirmSave(current);
  if (!decision) return;
  const toDelete = rows.filter((r) => r.deleted && r.origKey);
  const live = rows.filter((r) => !r.deleted);
  const toSet = live.filter(
    (r) => r.key && (r.key !== r.origKey || r.value !== r.origValue || r.added),
  );
  const message = defaultCommitMessage(current);
  try {
    await withBusy(saveEl(), 'Saving…', async () => {
      for (const row of toDelete) {
        await invoke('del_managed_key', { path: selected.path, key: row.origKey });
      }

      for (const row of toSet) {
        const renamed = Boolean(row.origKey) && row.key !== row.origKey;
        if (renamed && decision.rewriteRefs.has(row.origKey)) {
          await invoke('rename_key', {
            path: selected.path,
            key: row.origKey,
            value: row.key,
            yes: true,
          });
        } else {
          if (renamed) {
            await invoke('del_managed_key', { path: selected.path, key: row.origKey });
          }

          await invoke('set_managed_key', { path: selected.path, key: row.key, value: row.value });
        }

        row.origKey = row.key;
        row.origValue = row.value;
        row.added = false;
      }

      rows = live;
      await invoke('commit_managed_file', { path: selected.path, message });
    });

    renderKeys();
    await loadUnusedKeys(selected.path);
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
    const rel = String(name || '')
      .trim()
      .replaceAll('\\', '/');
    const stateBefore = await invoke('inspect_project', { path: project.path });
    const existing = (stateBefore.candidates || []).find(
      (file) => safeRel(file.rel, file.name) === rel,
    );
    let keys = [];
    if (existing) {
      const selected = await chooseProjectFiles(project.path, [existing], {
        action: 'Add file',
        heading: `Manage ${rel}?`,
        manageAll: true,
        skipLabel: 'Cancel',
      });
      if (!selected?.length) return;
      keys = selected[0].keys;
    }

    await invoke('add_project_file', { path: project.path, file: rel, keys });
    const state = await invoke('inspect_project', { path: project.path });
    project.files = state.managed || [];
    const idx = projects.findIndex((item) => item.path === project.path);
    if (idx !== -1) projects[idx] = project;
    renderTree();
    const created = project.files.find((file) => safeRel(file.rel, file.name) === rel);
    if (created) {
      await openFile(project, created);
    }

    document.getElementById('add-file-name').value = '';
  } catch (err) {
    showError(messageOf(err));
  }
}

function renderIntegrationSummary() {
  const el = document.getElementById('github-integration');
  if (!el) return;
  let target;
  if (integration.scope === 'org') {
    target = integration.org ? `organization ${integration.org}` : 'organization not configured';
  } else if (integration.scope === 'environment') {
    target =
      integration.repo && integration.environment
        ? `${integration.repo} / ${integration.environment}`
        : 'repository environment not configured';
  } else {
    target = integration.repo || 'repository not configured';
  }

  const configured = !target.endsWith('not configured');
  el.classList.toggle('configured', configured);
  el.title = `GitHub · ${target}`;
  el.setAttribute('aria-label', `Configure GitHub integration (${target})`);
}

async function loadPublishMapping(path) {
  try {
    const mapping = await invoke('get_publish_mapping', { path });
    integration = {
      scope: mapping?.scope || (mapping?.environment ? 'environment' : 'repo'),
      repo: mapping?.repo || '',
      org: mapping?.org || '',
      environment: mapping?.environment || '',
      prefix: mapping?.prefix || '',
      visibility: mapping?.visibility || 'all',
    };
  } catch {
    integration = {
      scope: 'repo',
      repo: '',
      org: '',
      environment: '',
      prefix: '',
      visibility: 'all',
    };
  }

  renderIntegrationSummary();
}

function syncAccessChrome(empty, fileOpen) {
  const emptyActions = document.getElementById('access-empty-actions');
  const toolbar = document.getElementById('access-toolbar');
  const form = document.querySelector('.access-form');
  const list = document.getElementById('access-list');
  const count = document.getElementById('access-count');
  const request = document.getElementById('request-access');
  if (!emptyActions || !toolbar || !form || !list || !count) return false;
  emptyActions.hidden = !(empty && !accessFormOpen && canGrant && fileOpen);
  toolbar.hidden = empty;
  form.hidden = !canGrant || (empty && !accessFormOpen);
  list.hidden = empty;
  if (request) request.hidden = !fileOpen;
  if (empty && !accessFormOpen) {
    list.replaceChildren();
    return false;
  }

  return true;
}

function renderAccess() {
  const empty = access.length === 0;
  const fileOpen = Boolean(selected);
  if (!syncAccessChrome(empty, fileOpen)) return;
  const list = document.getElementById('access-list');
  const count = document.getElementById('access-count');
  count.textContent = `${access.length} other ${access.length === 1 ? 'recipient' : 'recipients'}`;
  list.replaceChildren();
  if (empty) {
    const note = document.createElement('p');
    note.className = 'save-note access-empty';
    note.textContent = 'No recipients found in this file.';
    list.append(note);
    return;
  }

  for (const recipient of access) {
    const row = document.createElement('div');
    row.className = 'access-row';
    const avatar = document.createElement('span');
    avatar.className = 'identity-avatar identity-avatar-small';
    const nameLabel =
      recipient.name || (recipient.kind === 'robot' ? 'Unnamed robot' : 'Unnamed user');
    renderAvatar(avatar, recipient.kind === 'robot' ? 'robot' : 'person', nameLabel, nameLabel);
    const details = document.createElement('div');
    details.className = 'access-details';
    const name = document.createElement('strong');
    name.textContent = nameLabel;
    details.append(name);
    if (recipient.email) {
      const email = document.createElement('span');
      email.className = 'access-email';
      email.textContent = recipient.email;
      details.append(email);
    }

    if (recipient.kind === 'robot') {
      const badge = document.createElement('span');
      badge.className = 'access-kind';
      badge.textContent = 'robot';
      details.append(badge);
    }

    const remove = iconButton(
      'remove-recipient',
      `Remove access for ${name.textContent}`,
      'trash',
      () => removeRecipient(recipient),
    );
    remove.classList.add('danger');
    row.append(avatar, details, remove);
    list.append(row);
  }
}

async function loadAccess(path) {
  try {
    const recipients = await invoke('list_file_access', { path });
    access = recipients.filter((recipient) => !recipient.self);
  } catch {
    // An unlocked file no longer contains SOPS recipient metadata; keep the last known list.
  }

  renderAccess();
}

function jsonFlag(obj, key, fallback = true) {
  if (!obj || !Object.hasOwn(obj, key)) return fallback;
  return Boolean(obj[key]);
}

async function loadProjectConfig(path) {
  const target = path || selected?.project.path || projects[0]?.path || '';
  if (!target) return;
  try {
    const next = await invoke('get_account', { path: target });
    projectConfig = {
      path: target,
      name: '',
      owners: next.owners || [],
      canGrant: jsonFlag(next, 'can_grant'),
    };
    account = accountFrom(next);
  } catch {
    projectConfig = { path: target, name: '', owners: [], canGrant: true };
  }

  canGrant = projectConfig.canGrant !== false;
  renderProjectPanel();
  renderAccess();
}

function renderProjectPanel() {
  const project = selected?.project || projects[0];
  const nameEl = document.getElementById('project-panel-name');
  const pathEl = document.getElementById('project-panel-path');
  const identityEl = document.getElementById('project-panel-identity');
  const ownersEl = document.getElementById('project-owners');
  const copyKey = document.getElementById('project-copy-key');
  if (nameEl) nameEl.textContent = project?.name || '—';
  if (pathEl) pathEl.textContent = project ? parentLabel(project.path) : projectConfig.path || '—';
  if (identityEl) {
    identityEl.textContent = account.name || account.email || 'Not configured';
  }

  if (copyKey) copyKey.disabled = !account.publicKey;
  if (!ownersEl) return;
  const owners = Array.isArray(projectConfig.owners) ? projectConfig.owners : [];
  if (owners.length === 0) {
    ownersEl.textContent =
      'No Project owners listed yet. Anyone with Access can add people until owners are recorded in .sopsdeck.toml.';
    return;
  }

  const names = owners.map((owner) => owner.name || owner.key).join(', ');
  ownersEl.textContent = canGrant
    ? `Owners: ${names}. You can grant Access on this Project.`
    : `Owners: ${names}. Ask an owner to add people, or copy a request.`;
}

async function copyPublicKey() {
  if (!account.publicKey) {
    showError('Create an Age identity first');
    return;
  }

  if (await copyText(account.publicKey)) {
    setStatus('access', 'Copied your Age public key');
  }
}

async function requestAccess() {
  if (!selected) return;
  if (!account.publicKey) {
    showError('Create an Age identity first, then copy a request', 'access');
    return;
  }

  const file = selected.rel || selected.name || selected.path;
  const project = selected.project?.name || projectConfig.name || 'this Project';
  const who = account.name ? `${account.name}${account.email ? ` <${account.email}>` : ''}` : 'me';
  const message = `Hi — please grant me Access to ${file} in ${project}.

Name: ${who}
Age public key:
${account.publicKey}

You can add this key in Sopsdeck (Access → Add recipient) if you are a Project owner.`;
  if (await copyText(message)) {
    setStatus('access', 'Copied an access request including your public key');
  }
}

async function addRecipient() {
  if (!selected) return;
  if (!canGrant) {
    showError('Only a Project owner can add Access. Copy a request instead.', 'access');
    return;
  }

  const entered = document.getElementById('recipient-name').value.trim();
  const key = document.getElementById('recipient-key').value.trim();
  const { name, email } = parseGitIdentity(entered);
  if (!name || !key) {
    showError('Enter a name or git identity and an Age public key', 'access');
    return;
  }

  try {
    await withBusy(document.getElementById('grant-access'), '', () =>
      invoke('add_recipient', {
        path: selected.path,
        publicKey: key,
        name,
        email,
        kind: 'person',
      }),
    );
    document.getElementById('recipient-name').value = '';
    document.getElementById('recipient-key').value = '';
    await loadAccess(selected.path);
    setStatus('access', `Access granted to ${name}`);
  } catch (err) {
    showError(messageOf(err), 'access');
  }
}

async function removeRecipient(recipient) {
  if (!selected) return;
  // eslint-disable-next-line no-alert -- removing Access is a destructive confirmation.
  if (!window.confirm(`Remove ${recipient.name || 'this recipient'} from this file?`)) return;
  try {
    await invoke('remove_recipient', { path: selected.path, publicKey: recipient.key });
    accessFormOpen = false;
    await loadAccess(selected.path);
    setStatus('access', `Access removed for ${recipient.name || 'recipient'}`);
  } catch (err) {
    showError(messageOf(err), 'access');
  }
}

function openTeamMemberForm() {
  accessFormOpen = true;
  renderAccess();
  document.getElementById('recipient-name').focus();
}

function openRobotDialog() {
  document.getElementById('robot-name').value = '';
  document.getElementById('robot-result').hidden = true;
  document.getElementById('robot-create').hidden = false;
  document.getElementById('robot-error').hidden = true;
  document.getElementById('robot-status').hidden = true;
  window.robotAccount = null;
  document.getElementById('robot-dialog').showModal();
}

function updateIntegrationFields() {
  const scope = document.getElementById('integration-scope').value;
  document.getElementById('integration-repo-field').hidden = scope === 'org';
  document.getElementById('integration-org-field').hidden = scope !== 'org';
  document.getElementById('integration-environment-field').hidden = scope !== 'environment';
  document.getElementById('integration-visibility-field').hidden = scope !== 'org';
}

function integrationValues() {
  return {
    scope: document.getElementById('integration-scope').value,
    repo: document.getElementById('integration-repo').value.trim(),
    org: document.getElementById('integration-org').value.trim(),
    environment: document.getElementById('integration-environment').value.trim(),
    prefix: document.getElementById('integration-prefix').value.trim(),
    visibility: document.getElementById('integration-visibility').value,
  };
}

function openIntegrationDialog() {
  const dialog = document.getElementById('integration-dialog');
  document.getElementById('integration-scope').value = integration.scope;
  document.getElementById('integration-repo').value = integration.repo;
  document.getElementById('integration-org').value = integration.org;
  document.getElementById('integration-environment').value = integration.environment;
  document.getElementById('integration-prefix').value = integration.prefix;
  document.getElementById('integration-visibility').value = integration.visibility;
  document.getElementById('integration-prune').checked = false;
  document.getElementById('integration-dialog-error').hidden = true;
  document.getElementById('integration-dialog-status').hidden = true;
  updateIntegrationFields();
  dialog.showModal();
}

async function saveIntegrationConfig() {
  if (!selected) return false;
  const next = integrationValues();
  if (next.scope === 'org' && !next.org) throw new Error('Enter a GitHub organization');
  if (next.scope !== 'org' && !next.repo) throw new Error('Enter a GitHub repository');
  if (next.scope === 'environment' && !next.environment)
    throw new Error('Enter a repository environment');
  await invoke('configure_integration', { path: selected.path, ...next });
  integration = next;
  renderIntegrationSummary();
  return true;
}

async function syncIntegration() {
  const dialog = document.getElementById('integration-dialog');
  const error = document.getElementById('integration-dialog-error');
  try {
    await saveIntegrationConfig();
    const values = integrationValues();
    const result = await invoke('publish_managed_file', {
      path: selected.path,
      ...values,
      yes: true,
      prune: document.getElementById('integration-prune').checked,
    });
    document.getElementById('integration-dialog-status').hidden = false;
    document.getElementById('integration-dialog-status').textContent = String(result || 'Synced');
    setStatus('publish', 'Synced to GitHub');
  } catch (err) {
    error.hidden = false;
    error.textContent = messageOf(err);
  }

  if (!dialog.open) renderIntegrationSummary();
}

async function createRobotAccount() {
  const name = document.getElementById('robot-name').value.trim();
  if (!name) {
    document.getElementById('robot-error').textContent = 'Name the robot account first';
    document.getElementById('robot-error').hidden = false;
    return;
  }

  try {
    const robot = await invoke('create_robot_identity', { name });
    renderAvatar(document.getElementById('robot-avatar'), 'robot', robot.name, robot.name);
    document.getElementById('robot-display-name').textContent = robot.name;
    document.getElementById('robot-private-key').value =
      robot.privateKey || jsonValue(robot, 'private_key');
    document.getElementById('robot-result').hidden = false;
    document.getElementById('robot-create').hidden = true;
    document.getElementById('robot-error').hidden = true;
    window.robotAccount = robot;
  } catch (err) {
    document.getElementById('robot-error').textContent = messageOf(err);
    document.getElementById('robot-error').hidden = false;
  }
}

async function addRobotToFile() {
  if (!selected || !window.robotAccount) return;
  try {
    await invoke('add_recipient', {
      path: selected.path,
      publicKey: window.robotAccount.publicKey || jsonValue(window.robotAccount, 'public_key'),
      name: window.robotAccount.name,
      kind: 'robot',
    });
    await loadAccess(selected.path);
    document.getElementById('robot-status').hidden = false;
    document.getElementById('robot-status').textContent = 'Robot added to this file';
  } catch (err) {
    document.getElementById('robot-error').textContent = messageOf(err);
    document.getElementById('robot-error').hidden = false;
  }
}

async function loadUnusedKeys(path) {
  unusedKeys = new Set();
  try {
    const list = await invoke('unused', { path });
    if (Array.isArray(list)) for (const key of list) unusedKeys.add(key);
  } catch {
    // Unused analysis is advisory; never block the editor.
  }

  for (const row of rows) row.unused = unusedKeys.has(row.origKey);
}

async function loadDemoHints() {
  try {
    const response = await fetch('/demo');
    if (!response.ok) return;
    const info = await response.json();
    const input = document.getElementById('recipient-key');
    const nameInput = document.getElementById('recipient-name');
    if (input && info.bobPublicKey) input.value = info.bobPublicKey;
    if (nameInput && info.teammateName) nameInput.value = info.teammateName;
    const extras = Array.isArray(info.projects) ? info.projects : [];
    for (const path of extras) {
      if (!path || projects.some((p) => p.path === path)) continue;
      await addProjectFromPath(path, { select: false });
    }
  } catch {
    // The normal browser server has no /demo endpoint.
  }
}

const clipboardActions = {
  path(folderPath) {
    return addProjectFromPath(folderPath, { select: true });
  },
  recipient(item) {
    const keyInput = document.getElementById('recipient-key');
    const nameInput = document.getElementById('recipient-name');
    if (keyInput) keyInput.value = item.publicKey;
    if (nameInput && item.name) {
      nameInput.value = item.email ? `${item.name} <${item.email}>` : item.name;
    }

    accessFormOpen = true;
    renderAccess();
    if (item.name) {
      document.getElementById('grant-access').click();
      return;
    }

    nameInput?.focus();
  },
  bulk(pairs) {
    if (!selected) return;
    const { adds, changes } = classifyPasteKeys(currentPasteKeys(), pairs);
    pendingPaste = { kind: 'bulk', pairs, adds, changes };
    renderKeys();
  },
  lone(raw) {
    if (!selected) return;
    pendingPaste = { kind: 'lone', value: raw };
    renderKeys();
  },
  onError(err) {
    showError(messageOf(err));
  },
};

window.addEventListener('DOMContentLoaded', async () => {
  decorateChrome();
  initInspector();
  renderAccount();
  document.addEventListener('paste', onEditorPaste);
  applyTheme(currentTheme());
  document.getElementById('whats-new').addEventListener('click', () => {
    showWhatsNew((err) => showError(messageOf(err)));
  });
  document.getElementById('whats-new-close').addEventListener('click', () => {
    document.getElementById('whats-new-dialog').close();
  });
  document.getElementById('theme-toggle').addEventListener('click', () => {
    applyTheme(currentTheme() === 'dark' ? 'light' : 'dark');
  });
  document.getElementById('account').addEventListener('click', () => openAccountDialog());
  document.getElementById('project-account').addEventListener('click', () => openAccountDialog());
  document.getElementById('account-copy-key').addEventListener('click', copyPublicKey);
  document.getElementById('project-copy-key').addEventListener('click', copyPublicKey);
  document.getElementById('request-access').addEventListener('click', requestAccess);
  document.getElementById('add-project').addEventListener('click', addProject);
  document.getElementById('add-file').addEventListener('click', addManagedFile);
  document.getElementById('add-file-name').addEventListener('keydown', (event) => {
    if (event.key !== 'Enter') return;
    event.preventDefault();
    addManagedFile();
  });
  saveEl().addEventListener('click', saveFile);
  fileLockEl().addEventListener('click', toggleFileLock);
  copyFileEl().addEventListener('click', copyFileContents);
  fileHistoryEl().addEventListener('click', showFileHistory);
  document.getElementById('grant-access').addEventListener('click', addRecipient);
  document.getElementById('create-robot').addEventListener('click', openRobotDialog);
  document.getElementById('add-team-member').addEventListener('click', openTeamMemberForm);
  document.getElementById('add-bot-account').addEventListener('click', openRobotDialog);
  document.getElementById('github-integration').addEventListener('click', openIntegrationDialog);
  document.getElementById('integration-scope').addEventListener('change', updateIntegrationFields);
  document
    .getElementById('integration-cancel')
    .addEventListener('click', () => document.getElementById('integration-dialog').close());
  document.getElementById('integration-save').addEventListener('click', async () => {
    try {
      await saveIntegrationConfig();
      document.getElementById('integration-dialog-status').hidden = false;
      document.getElementById('integration-dialog-status').textContent = 'Configuration saved';
    } catch (err) {
      const error = document.getElementById('integration-dialog-error');
      error.hidden = false;
      error.textContent = messageOf(err);
    }
  });
  document.getElementById('integration-sync').addEventListener('click', syncIntegration);
  document
    .getElementById('robot-cancel')
    .addEventListener('click', () => document.getElementById('robot-dialog').close());
  document.getElementById('robot-create').addEventListener('click', createRobotAccount);
  document
    .getElementById('robot-copy')
    .addEventListener('click', () => copyText(document.getElementById('robot-private-key').value));
  document.getElementById('robot-add').addEventListener('click', addRobotToFile);
  document
    .getElementById('file-history-close')
    .addEventListener('click', () => document.getElementById('file-history-dialog').close());
  document
    .getElementById('secret-history-close')
    .addEventListener('click', () => document.getElementById('secret-history-dialog').close());
  document.getElementById('clipboard-dismiss').addEventListener('click', () => {
    document.getElementById('clipboard-dialog').close();
  });
  window.addEventListener('focus', () => {
    setTimeout(() => sniffClipboard(clipboardActions), 200);
  });
  window.addEventListener('blur', () => {
    resetClipboardSeen();
  });
  renderWorkspace();
  if (skipBoot()) return;
  try {
    const boot = await invoke('boot_project');
    let demo = false;
    try {
      const response = await fetch('/demo');
      demo = response.ok;
    } catch {
      demo = false;
    }

    focusedProject = Boolean(boot) && !demo;
    document.body.classList.toggle('focused-project', focusedProject);
    if (boot) await addProjectFromPath(boot);
    if (!focusedProject) await loadDemoHints();
  } catch (err) {
    showError(messageOf(err));
  }
});
