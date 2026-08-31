import { icon, iconButton } from './icons.js';
import { nestLeaves } from './tree.js';

function valueInput(row, mark) {
  const shown = Boolean(row.revealed);
  const input = document.createElement('textarea');
  input.rows = 1;
  input.className = 'value' + (shown ? '' : ' masked');
  input.dataset.testid = 'key-value';
  input.value = shown ? row.value : '••••••••••••••••••••••••••••';
  input.readOnly = !shown;
  input.autocomplete = 'off';
  input.spellcheck = false;
  const autosize = () => {
    input.style.height = 'auto';
    input.style.height = `${Math.min(Math.max(input.scrollHeight, 22), 240)}px`;
  };

  input.addEventListener('input', () => {
    if (!shown) return;
    row.value = input.value;
    mark();
    autosize();
  });
  queueMicrotask(autosize);
  return input;
}

function rowActions(row, shown, ui) {
  const actions = document.createElement('div');
  actions.className = 'key-actions';
  const remove = iconButton('delete-key', 'Delete key', 'trash', () => ui.deleteRow(row));
  remove.classList.add('danger');
  actions.append(
    iconButton('reveal-key', shown ? 'Hide value' : 'Reveal value', shown ? 'eye-off' : 'eye', () =>
      ui.toggleRowReveal(row),
    ),
    iconButton('copy-value', 'Copy value', 'copy', () => ui.copyText(row.value)),
    iconButton('secret-history', 'Secret history', 'history', () => ui.showSecretHistory(row)),
    remove,
  );
  return actions;
}

function appendComposer(box, structured, ui) {
  const composer = document.createElement('div');
  composer.className = 'key-composer';
  const composerInput = document.createElement('input');
  composerInput.type = 'text';
  composerInput.dataset.testid = 'key-composer';
  composerInput.placeholder = structured ? 'build.env.TOKEN=value' : 'KEY=value';
  composerInput.autocomplete = 'off';
  composerInput.spellcheck = false;
  const addFromComposer = () => {
    const parsed = ui.parseComposerLine(composerInput.value);
    if (!parsed) return;
    ui.addRow(parsed);
  };

  composerInput.addEventListener('keydown', (event) => {
    if (event.key !== 'Enter') return;
    event.preventDefault();
    addFromComposer();
  });
  composer.append(composerInput, iconButton('composer-add', 'Add key', 'plus', addFromComposer));
  box.append(composer);
  if (ui.consumeComposerFocus()) composerInput.focus();
}

function renderKeyHead(box, structured, ui) {
  const head = document.createElement('div');
  head.className = 'key-head' + (structured ? ' key-head-tree' : '');
  const keyHead = document.createElement('span');
  keyHead.textContent = structured ? 'Path' : 'Key';
  const valueHead = document.createElement('span');
  valueHead.className = 'value-head';
  valueHead.append('Value');
  valueHead.append(
    iconButton(
      'reveal',
      ui.revealed ? 'Hide values' : 'Reveal values',
      ui.revealed ? 'eye-off' : 'eye',
      ui.toggleReveal,
    ),
  );
  const actionsHead = document.createElement('span');
  if (structured) {
    const encryptHead = document.createElement('span');
    encryptHead.textContent = 'Encrypt';
    head.append(encryptHead, keyHead, valueHead, actionsHead);
  } else {
    head.append(keyHead, valueHead, actionsHead);
  }

  box.append(head);
}

function markRow(line, row, kind, ui) {
  line.classList.toggle('changed', ui.rowDirty(row));
  if (kind) {
    kind.textContent = ui.rowDirty(row) ? 'changed' : 'secret';
    ui.refreshUnusedBadge(kind, row);
  }

  ui.onDirty();
}

function renderDotenvRow(box, row, ui) {
  const line = document.createElement('div');
  const changed = ui.rowDirty(row);
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
  ui.refreshUnusedBadge(kind, row);
  const mark = () => markRow(line, row, kind, ui);
  name.addEventListener('input', () => {
    row.key = name.value;
    mark();
  });
  keyCell.append(
    name,
    kind,
    iconButton('copy-key', 'Copy key', 'copy', () => ui.copyText(row.key)),
  );
  const valueCell = document.createElement('div');
  valueCell.className = 'value-cell';
  valueCell.append(valueInput(row, mark));
  line.append(keyCell, valueCell, rowActions(row, Boolean(row.revealed), ui));
  box.append(line);
}

function renderTreeNodes(box, nodes, byKey, ui) {
  const depth = ui.depth ?? 0;
  for (const node of nodes) {
    if (node.leaf) {
      const row = byKey.get(node.path);
      if (!row) continue;
      const line = document.createElement('div');
      line.className = 'key-row json-leaf' + (ui.rowDirty(row) ? ' changed' : '');
      line.dataset.testid = 'key-row';
      line.style.setProperty('--tree-depth', String(depth));
      const encrypt = document.createElement('button');
      encrypt.type = 'button';
      encrypt.className = 'icon-button encrypt-toggle' + (row.encrypted ? ' is-on' : '');
      encrypt.dataset.testid = 'encrypt-toggle';
      encrypt.setAttribute('aria-pressed', row.encrypted ? 'true' : 'false');
      encrypt.title = row.encrypted ? 'Stop encrypting this path' : 'Encrypt this path';
      encrypt.append(icon(row.encrypted ? 'lock' : 'unlock'));
      const mark = () => markRow(line, row, null, ui);
      encrypt.addEventListener('click', () => ui.toggleEncrypted(row));
      const keyCell = document.createElement('div');
      keyCell.className = 'key-cell';
      const name = document.createElement('code');
      name.className = 'key-name json-key';
      name.textContent = node.name;
      name.title = row.key;
      const hiddenName = document.createElement('input');
      hiddenName.type = 'hidden';
      hiddenName.dataset.testid = 'key-name';
      hiddenName.value = row.key;
      keyCell.append(name, hiddenName);
      const valueCell = document.createElement('div');
      valueCell.className = 'value-cell';
      valueCell.append(valueInput(row, mark));
      line.append(encrypt, keyCell, valueCell, rowActions(row, Boolean(row.revealed), ui));
      box.append(line);
      continue;
    }

    const folder = document.createElement('div');
    folder.className = 'json-folder';
    folder.style.setProperty('--tree-depth', String(depth));
    const label = document.createElement('span');
    label.className = 'json-folder-name';
    label.textContent = node.name;
    folder.append(label);
    box.append(folder);
    renderTreeNodes(box, node.children, byKey, { ...ui, depth: depth + 1 });
  }
}

export function renderKeyRows(box, visible, structured, ui) {
  renderKeyHead(box, structured, ui);
  if (structured) {
    const byKey = new Map(visible.map((row) => [row.key, row]));
    const wrap = document.createElement('div');
    wrap.className = 'json-tree';
    wrap.dataset.testid = 'json-tree';
    renderTreeNodes(wrap, nestLeaves(visible.map((row) => row.key)), byKey, { ...ui, depth: 0 });
    box.append(wrap);
  } else {
    for (const row of visible) renderDotenvRow(box, row, ui);
  }

  appendComposer(box, structured, ui);
}

export function appendSetupKeyTree(list, nodes, state) {
  for (const node of nodes) {
    if (node.leaf) {
      const keyLabel = document.createElement('label');
      keyLabel.className = 'setup-project-key';
      keyLabel.style.setProperty('--tree-depth', String(state.depth));
      const keyInput = document.createElement('input');
      keyInput.type = 'checkbox';
      keyInput.value = node.path;
      keyInput.checked = state.checked;
      keyInput.dataset.testid = 'setup-project-key-toggle';
      const keyName = document.createElement('code');
      keyName.textContent = node.name;
      keyName.title = node.path;
      keyLabel.append(keyInput, keyName);
      list.append(keyLabel);
      state.keyInputs.push(keyInput);
      continue;
    }

    const folder = document.createElement('div');
    folder.className = 'setup-project-folder';
    folder.style.setProperty('--tree-depth', String(state.depth));
    folder.textContent = node.name;
    list.append(folder);
    appendSetupKeyTree(list, node.children, { ...state, depth: state.depth + 1 });
  }
}
