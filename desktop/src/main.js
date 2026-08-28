const invoke = globalThis.__TAURI__?.core?.invoke ?? invokeOverHTTP;
const THEME_KEY = 'sopsdeck-theme';

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
const toolbarEl = () => document.getElementById('toolbar');
const saveEl = () => document.getElementById('save');
const revealEl = () => document.getElementById('reveal');
const commitEl = () => document.getElementById('commit-message');

const projects = [];
let selected = null;
let rows = [];
let revealed = false;
let commitAuto = true;
let lastAuto = '';

function messageOf(err) {
  if (err instanceof Error) return err.message;
  return String(err);
}

function skipBoot() {
  return new URLSearchParams(location.search).has('empty');
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

function displayPath(project, file) {
  const name = project?.name || 'project';
  return `~/${name}/${safeRel(file.rel, file.name)}`;
}

function dirtyCount() {
  return rows.filter((r) => r.key !== r.origKey || r.value !== r.origValue || r.added).length;
}

function defaultCommitMessage(current) {
  const added = [];
  const changed = [];
  for (const row of current) {
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
  return parts.join('; ');
}

function syncCommitMessage() {
  if (!commitAuto) return;
  const next = defaultCommitMessage(rows);
  if (next === '') return;
  lastAuto = next;
  commitEl().value = lastAuto;
}

function showEmpty(message) {
  const empty = emptyEl();
  empty.hidden = !message;
  empty.textContent = message || '';
  keysEl().hidden = Boolean(message);
}

function resetEditorChrome() {
  selected = null;
  rows = [];
  revealed = false;
  badgeEl().hidden = true;
  toolbarEl().hidden = true;
  document.getElementById('meta-path').textContent = '—';
  document.getElementById('meta-format').textContent = '—';
  document.getElementById('meta-enc').textContent = '—';
  saveEl().disabled = true;
}

function renderWorkspace() {
  if (projects.length === 0) {
    resetEditorChrome();
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
  if (btn) btn.textContent = theme === 'dark' ? 'Light mode' : 'Dark mode';
}

function renderTree() {
  const nav = treeEl();
  nav.replaceChildren();
  for (const project of projects) {
    const wrap = document.createElement('div');
    const title = document.createElement('div');
    title.className = 'project';
    title.append(project.name);
    const hint = document.createElement('span');
    hint.className = 'project-path';
    hint.textContent = parentLabel(project.path);
    title.append(hint);
    wrap.append(title);
    const files = document.createElement('div');
    files.className = 'files';
    for (const file of project.files) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'file' + (selected && file.path === selected.path ? ' selected' : '');
      btn.dataset.testid = 'managed-file';
      btn.textContent = safeRel(file.rel, file.name);
      btn.addEventListener('click', () => openFile(project, file));
      files.append(btn);
    }

    wrap.append(files);
    nav.append(wrap);
  }
}

async function openFile(project, file) {
  selected = { project, ...file };
  revealed = false;
  commitAuto = true;
  lastAuto = '';
  commitEl().value = '';
  revealEl().textContent = 'Reveal values';
  renderTree();
  crumbEl().textContent = displayPath(project, file);
  headlineEl().textContent = titleOf(file.name);
  showError('');
  badgeEl().hidden = false;
  toolbarEl().hidden = false;
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
    }));
    sublineEl().textContent = `${rows.length} secrets · never uploaded`;
    renderKeys();
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
    renderWorkspace();
    return;
  }

  if (rows.length === 0) {
    showEmpty('No keys in this file.');
    saveEl().disabled = true;
    sublineEl().textContent = '0 secrets · never uploaded';
    return;
  }

  showEmpty('');
  const box = keysEl();
  box.hidden = false;
  box.replaceChildren();
  const head = document.createElement('div');
  head.className = 'key-head';
  for (const label of ['Key', 'Value', 'Type']) {
    const span = document.createElement('span');
    span.textContent = label;
    head.append(span);
  }

  box.append(head);

  for (const row of rows) {
    const line = document.createElement('div');
    const changed = row.key !== row.origKey || row.value !== row.origValue || row.added;
    line.className = 'key-row' + (changed ? ' changed' : '');
    line.dataset.testid = 'key-row';
    const name = document.createElement('input');
    name.className = 'key-name';
    name.dataset.testid = 'key-name';
    name.value = row.key;
    name.readOnly = !row.added;
    name.addEventListener('input', () => {
      row.key = name.value;
      const dirty = row.key !== row.origKey || row.value !== row.origValue || row.added;
      line.classList.toggle('changed', dirty);
      kind.textContent = dirty ? 'changed' : 'secret';
      saveEl().disabled = dirtyCount() === 0;
      syncCommitMessage();
    });
    const input = document.createElement('input');
    input.className = 'value' + (revealed ? '' : ' masked');
    input.dataset.testid = 'key-value';
    input.value = revealed ? row.value : '••••••••••••••••••••••••••••';
    input.readOnly = !revealed;
    input.addEventListener('input', () => {
      row.value = input.value;
      const dirty = row.key !== row.origKey || row.value !== row.origValue || row.added;
      line.classList.toggle('changed', dirty);
      kind.textContent = dirty ? 'changed' : 'secret';
      saveEl().disabled = dirtyCount() === 0;
      syncCommitMessage();
    });
    const kind = document.createElement('span');
    kind.className = 'kind';
    kind.textContent = changed ? 'changed' : 'secret';
    line.append(name, input, kind);
    box.append(line);
  }

  saveEl().disabled = dirtyCount() === 0;
  syncCommitMessage();
  sublineEl().textContent = selected
    ? `${rows.length} secrets · ${dirtyCount() ? 'edited locally' : 'never uploaded'}`
    : sublineEl().textContent;
}

async function addProjectFromPath(path) {
  const files = await invoke('list_managed_files', { path });
  const name = path.split('/').findLast(Boolean) || path;
  const existing = projects.findIndex((p) => p.path === path);
  const project = { name, path, files };
  if (existing === -1) {
    projects.push(project);
  } else {
    projects[existing] = project;
  }

  renderTree();
  if (files[0]) {
    await openFile(project, files[0]);
    return;
  }

  selected = null;
  renderWorkspace();
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

async function withBusy(el, label, fn) {
  const origText = el.textContent;
  const origDisabled = el.disabled;
  el.setAttribute('aria-busy', 'true');
  el.disabled = true;
  if (label) el.textContent = label;
  try {
    return await fn();
  } finally {
    el.removeAttribute('aria-busy');
    el.textContent = origText;
    el.disabled = origDisabled;
  }
}

async function saveFile() {
  if (!selected) return;
  showError('');
  const pending = rows.filter(
    (r) => r.key && (r.key !== r.origKey || r.value !== r.origValue || r.added),
  );
  try {
    await withBusy(saveEl(), 'Saving…', async () => {
      for (const row of pending) {
        await invoke('set_managed_key', { path: selected.path, key: row.key, value: row.value });
        row.origKey = row.key;
        row.origValue = row.value;
        row.added = false;
      }
    });

    renderKeys();
  } catch (err) {
    showError(messageOf(err), 'save');
    saveEl().disabled = dirtyCount() === 0;
  }
}

async function publishFile(yes) {
  if (!selected) return;
  showError('');
  setStatus('publish', '');
  const btn = document.getElementById(yes ? 'publish-yes' : 'publish');
  try {
    const result = await withBusy(btn, yes ? 'Publishing…' : 'Checking…', () =>
      invoke('publish_managed_file', {
        path: selected.path,
        prefix: 'SD_',
        yes,
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
    document.getElementById('whats-new-version').textContent = payload.version
      ? `Sopsdeck ${payload.version}`
      : '';
    const list = document.getElementById('whats-new-list');
    list.replaceChildren();
    for (const note of payload.notes || []) {
      const item = document.createElement('li');
      item.textContent = note;
      list.append(item);
    }

    dialog.showModal();
  } catch (err) {
    showError(messageOf(err));
  }
}

window.addEventListener('DOMContentLoaded', async () => {
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
  document.getElementById('add-secret').addEventListener('click', () => {
    if (!selected) return;
    rows.push({ key: '', value: '', origKey: '', origValue: '', added: true });
    revealed = true;
    revealEl().textContent = 'Hide values';
    renderKeys();
  });
  revealEl().addEventListener('click', () => {
    revealed = !revealed;
    revealEl().textContent = revealed ? 'Hide values' : 'Reveal values';
    renderKeys();
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
