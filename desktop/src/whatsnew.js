const noteTags = {
  feature: 'Feature',
  bugfix: 'Bug fix',
  performance: 'Performance',
  changed: 'Changed',
  removed: 'Removed',
  security: 'Security',
};

async function loadWhatsNew() {
  if (globalThis.__TAURI__?.core?.invoke) {
    return globalThis.__TAURI__.core.invoke('whats_new');
  }

  const response = await fetch('/whats-new.json');
  if (!response.ok) {
    throw new Error(response.statusText);
  }

  return response.json();
}

export async function showWhatsNew(onError) {
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
      list.append(buildNote(note));
    }

    dialog.showModal();
  } catch (err) {
    onError(err);
  }
}

function buildNote(note) {
  const item = document.createElement('li');
  item.className = 'whats-new-item';
  const text = typeof note === 'string' ? note : note.text;
  const type = typeof note === 'string' ? '' : note.type;
  const platforms = typeof note === 'string' ? [] : note.platforms || [];
  if (type) {
    const tag = document.createElement('span');
    tag.className = 'note-tag';
    tag.dataset.testid = 'whats-new-tag';
    tag.textContent = noteTags[type] || type;
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

  return item;
}
