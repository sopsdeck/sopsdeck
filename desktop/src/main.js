const { invoke } = window.__TAURI__.core;

const treeEl = () => document.getElementById("tree");
const keysEl = () => document.getElementById("keys");
const crumbEl = () => document.getElementById("breadcrumb");
const headlineEl = () => document.getElementById("headline");
const sublineEl = () => document.getElementById("subline");
const errorEl = () => document.getElementById("error");
const badgeEl = () => document.getElementById("badge");
const toolbarEl = () => document.getElementById("toolbar");
const saveEl = () => document.getElementById("save");
const revealEl = () => document.getElementById("reveal");

const projects = [];
let selected = null;
let rows = [];
let revealed = false;

function showError(msg) {
  const el = errorEl();
  if (!msg) {
    el.hidden = true;
    el.textContent = "";
    return;
  }
  el.hidden = false;
  el.textContent = msg;
}

function formatOf(path) {
  const base = path.split("/").pop() || path;
  if (base === ".env" || base.startsWith(".env.") || base.toLowerCase().endsWith(".env")) {
    return "dotenv";
  }
  if (base.toLowerCase().endsWith(".json")) return "json";
  if (base.toLowerCase().endsWith(".yaml") || base.toLowerCase().endsWith(".yml")) return "yaml";
  return "unknown";
}

function titleOf(name) {
  if (name.startsWith(".env.")) {
    const rest = name.slice(5);
    return rest.charAt(0).toUpperCase() + rest.slice(1);
  }
  return name;
}

function parentLabel(path) {
  const parts = path.split("/").filter(Boolean);
  if (parts.length < 2) return path;
  return "~/" + parts[parts.length - 2];
}

function dirtyCount() {
  return rows.filter((r) => r.key !== r.origKey || r.value !== r.origValue || r.added).length;
}

function renderTree() {
  const nav = treeEl();
  nav.replaceChildren();
  for (const project of projects) {
    const wrap = document.createElement("div");
    const title = document.createElement("div");
    title.className = "project";
    title.append(project.name);
    const hint = document.createElement("span");
    hint.className = "project-path";
    hint.textContent = parentLabel(project.path);
    title.append(hint);
    wrap.append(title);
    const files = document.createElement("div");
    files.className = "files";
    for (const file of project.files) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "file" + (selected && file.path === selected.path ? " selected" : "");
      btn.textContent = file.rel || file.name;
      btn.addEventListener("click", () => openFile(project, file));
      files.append(btn);
    }
    wrap.append(files);
    nav.append(wrap);
  }
}

async function openFile(project, file) {
  selected = { project, ...file };
  revealed = false;
  revealEl().textContent = "Reveal values";
  renderTree();
  crumbEl().textContent = file.path;
  headlineEl().textContent = titleOf(file.name);
  showError("");
  badgeEl().hidden = false;
  toolbarEl().hidden = false;
  document.getElementById("meta-path").textContent = file.rel || file.name;
  document.getElementById("meta-format").textContent = formatOf(file.path);
  document.getElementById("meta-enc").textContent = "age + SOPS";
  try {
    const pairs = await invoke("get_managed_file", { path: file.path });
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
    showError(String(err));
  }
}

function renderKeys() {
  const box = keysEl();
  box.replaceChildren();
  const head = document.createElement("div");
  head.className = "key-head";
  head.innerHTML = "<span>Key</span><span>Value</span><span>Type</span>";
  box.append(head);
  if (!rows.length) {
    const empty = document.createElement("p");
    empty.className = "subline";
    empty.textContent = "No keys in this file.";
    box.append(empty);
  }
  for (const row of rows) {
    const line = document.createElement("div");
    const changed = row.key !== row.origKey || row.value !== row.origValue || row.added;
    line.className = "key-row" + (changed ? " changed" : "");
    const name = document.createElement("input");
    name.className = "key-name";
    name.value = row.key;
    name.readOnly = !row.added;
    name.addEventListener("input", () => {
      row.key = name.value;
      const dirty = row.key !== row.origKey || row.value !== row.origValue || row.added;
      line.classList.toggle("changed", dirty);
      kind.textContent = dirty ? "changed" : "secret";
      saveEl().disabled = dirtyCount() === 0;
    });
    const input = document.createElement("input");
    input.className = "value" + (revealed ? "" : " masked");
    input.value = revealed ? row.value : "••••••••••••••••••••••••••••";
    input.readOnly = !revealed;
    input.addEventListener("input", () => {
      row.value = input.value;
      const dirty = row.key !== row.origKey || row.value !== row.origValue || row.added;
      line.classList.toggle("changed", dirty);
      kind.textContent = dirty ? "changed" : "secret";
      saveEl().disabled = dirtyCount() === 0;
    });
    const kind = document.createElement("span");
    kind.className = "kind";
    kind.textContent = changed ? "changed" : "secret";
    line.append(name, input, kind);
    box.append(line);
  }
  saveEl().disabled = dirtyCount() === 0;
  sublineEl().textContent = selected
    ? `${rows.length} secrets · ${dirtyCount() ? "edited locally" : "never uploaded"}`
    : sublineEl().textContent;
}

async function addProjectFromPath(path) {
  const files = await invoke("list_managed_files", { path });
  const name = path.split("/").filter(Boolean).pop() || path;
  const existing = projects.findIndex((p) => p.path === path);
  const project = { name, path, files };
  if (existing >= 0) projects[existing] = project;
  else projects.push(project);
  renderTree();
  if (files[0]) openFile(project, files[0]);
}

async function addProject() {
  showError("");
  try {
    const selectedPath = await invoke("pick_project_folder");
    if (!selectedPath) return;
    await addProjectFromPath(selectedPath);
  } catch (err) {
    showError(String(err));
  }
}

async function saveFile() {
  if (!selected) return;
  showError("");
  const pending = rows.filter((r) => r.key && (r.key !== r.origKey || r.value !== r.origValue || r.added));
  try {
    for (const row of pending) {
      await invoke("set_managed_key", { path: selected.path, key: row.key, value: row.value });
      row.origKey = row.key;
      row.origValue = row.value;
      row.added = false;
    }
    renderKeys();
  } catch (err) {
    showError(String(err));
  }
}

window.addEventListener("DOMContentLoaded", async () => {
  document.getElementById("add-project").addEventListener("click", addProject);
  document.getElementById("add-secret").addEventListener("click", () => {
    if (!selected) return;
    rows.push({ key: "", value: "", origKey: "", origValue: "", added: true });
    revealed = true;
    revealEl().textContent = "Hide values";
    renderKeys();
  });
  revealEl().addEventListener("click", () => {
    revealed = !revealed;
    revealEl().textContent = revealed ? "Hide values" : "Reveal values";
    renderKeys();
  });
  saveEl().addEventListener("click", saveFile);
  try {
    const boot = await invoke("boot_project");
    if (boot) await addProjectFromPath(boot);
  } catch (err) {
    showError(String(err));
  }
});
