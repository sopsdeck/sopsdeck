import { classifyClipboard } from './paste.js';

let lastClipboardText = '';

const labels = {
  path: 'Open this folder as a Project',
  recipient: 'Grant Access to the open Managed File',
  bulk: 'Paste these secrets into the open Managed File',
  lone: 'Paste this value into the open Managed File',
};

const descriptions = {
  path(item) {
    return `Folder path: ${item.path}`;
  },
  recipient(item) {
    return `Age recipient: ${item.publicKey.slice(0, 24)}…`;
  },
  bulk(item) {
    return `Secrets: ${item.names.join(', ')}`;
  },
  lone() {
    return 'A lone secret value';
  },
};

function descriptionOf(item) {
  return (descriptions[item.kind] ?? (() => ''))(item);
}

// Read the system clipboard on focus, classify it, and open a modal offering
// the matching action. actions maps each kind to an async fn.
export async function sniffClipboard(actions) {
  const clip = navigator.clipboard;
  if (!clip?.readText) return;
  let text;
  try {
    text = await clip.readText();
  } catch {
    return; // Permission denied or not a secure context.
  }

  if (!text || text === lastClipboardText) return;
  lastClipboardText = text;
  const item = classifyClipboard(text);
  if (!item) return;
  showClipboardModal(item, text, actions);
}

export function resetClipboardSeen() {
  lastClipboardText = '';
}

function showClipboardModal(item, raw, actions) {
  const dialog = document.getElementById('clipboard-dialog');
  const summary = document.getElementById('clipboard-summary');
  const actionBox = document.getElementById('clipboard-actions');
  actionBox.replaceChildren();
  summary.textContent = descriptionOf(item);
  const add = (label, handler) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'tool has-icon';
    btn.textContent = label;
    btn.addEventListener('click', async () => {
      dialog.close();
      try {
        await handler();
      } catch (err) {
        actions.onError(err);
      }
    });
    actionBox.append(btn);
  };

  const builders = {
    path() {
      add(labels.path, () => actions.path(item.path));
    },
    recipient() {
      add(labels.recipient, () => actions.recipient(item.publicKey));
    },
    bulk() {
      add(labels.bulk, () => actions.bulk(item.pairs));
    },
    lone() {
      add(labels.lone, () => actions.lone(raw));
    },
  };
  (builders[item.kind] ?? (() => {}))();
  dialog.showModal();
}
