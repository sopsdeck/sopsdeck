const invoke = globalThis.__TAURI__?.core?.invoke ?? invokeOverHTTP;

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
const crumbEl = () => document.getElementById('breadcrumb');
const headlineEl = () => document.getElementById('headline');
const sublineEl = () => document.getElementById('subline');
const errorEl = () => document.getElementById('error');
const gitErrorEl = () => document.getElementById('git-error');
const saveErrorEl = () => document.getElementById('save-error');
const badgeEl = () => document.getElementById('badge');
const toolbarEl = () => document.getElementById('toolbar');
const saveEl = () => document.getElementById('save');
const revealEl = () => document.getElementById('reveal');

const projects = [];
let selected = null;
let rows = [];
let revealed = false;

function messageOf(err) {
  if (err instanceof Error) return err.message;
  return String(err);
}

function clearErrors() {
  for (const el of [errorEl(), gitErrorEl(), saveErrorEl()]) {
    if (!el) continue;
    el.hidden = true;
    el.textContent = '';
  }
}

function showError(msg, region = 'editor') {
  clearErrors();
  if (!msg) return;
  let el = errorEl();
  if (region === 'git') el = gitErrorEl();
  if (region === 'save') el = saveErrorEl();
  el.hidden = false;
  el.textContent = msg;
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

function dirtyCount() {
  return rows.filter((r) => r.key !== r.origKey || r.value !== r.origValue || r.added).length;
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
      btn.textContent = file.rel || file.name;
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
  revealEl().textContent = 'Reveal values';
  renderTree();
  crumbEl().textContent = file.path;
  headlineEl().textContent = titleOf(file.name);
  showError('');
  badgeEl().hidden = false;
  toolbarEl().hidden = false;
  document.getElementById('meta-path').textContent = file.rel || file.name;
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
    showError(messageOf(err));
  }
}

function renderKeys() {
  const box = keysEl();
  box.replaceChildren();
  const head = document.createElement('div');
  head.className = 'key-head';
  for (const label of ['Key', 'Value', 'Type']) {
    const span = document.createElement('span');
    span.textContent = label;
    head.append(span);
  }

  box.append(head);
  if (rows.length === 0) {
    const empty = document.createElement('p');
    empty.className = 'subline';
    empty.textContent = 'No keys in this file.';
    box.append(empty);
  }

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
    });
    const kind = document.createElement('span');
    kind.className = 'kind';
    kind.textContent = changed ? 'changed' : 'secret';
    line.append(name, input, kind);
    box.append(line);
  }

  saveEl().disabled = dirtyCount() === 0;
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
  if (files[0]) openFile(project, files[0]);
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

async function saveFile() {
  if (!selected) return;
  showError('');
  const pending = rows.filter(
    (r) => r.key && (r.key !== r.origKey || r.value !== r.origValue || r.added),
  );
  try {
    for (const row of pending) {
      await invoke('set_managed_key', { path: selected.path, key: row.key, value: row.value });
      row.origKey = row.key;
      row.origValue = row.value;
      row.added = false;
    }

    renderKeys();
  } catch (err) {
    showError(messageOf(err), 'save');
  }
}

window.addEventListener('DOMContentLoaded', async () => {
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
      const message = document.getElementById('commit-message').value;
      await invoke('commit_managed_file', { path: selected.path, message });
      document.getElementById('commit-message').value = '';
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  document.getElementById('sync').addEventListener('click', async () => {
    if (!selected) return;
    showError('');
    try {
      await invoke('sync_project', { path: selected.path });
    } catch (err) {
      showError(messageOf(err), 'git');
    }
  });
  try {
    const boot = await invoke('boot_project');
    if (boot) await addProjectFromPath(boot);
  } catch (err) {
    showError(messageOf(err));
  }
});
