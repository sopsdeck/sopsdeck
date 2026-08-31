export function isStructuredFormat(format) {
  return format === 'json' || format === 'yaml';
}

export function splitLeafPath(key) {
  const parts = [];
  let i = 0;
  const text = String(key || '');
  while (i < text.length) {
    const start = i;
    while (i < text.length && text[i] !== '.' && text[i] !== '[') i += 1;
    if (start === i) throw new Error(`invalid key path ${text}`);
    parts.push(text.slice(start, i));
    while (i < text.length && text[i] === '[') {
      const end = text.indexOf(']', i);
      if (end === -1) throw new Error(`invalid key path ${text}`);
      parts.push(text.slice(i + 1, end));
      i = end + 1;
    }

    if (i < text.length) {
      if (text[i] !== '.') throw new Error(`invalid key path ${text}`);
      i += 1;
    }
  }

  return parts;
}

export function nestLeaves(keys) {
  const root = { name: '', path: '', children: [] };
  for (const key of keys) {
    let node = root;
    const parts = splitLeafPath(key);
    let path = '';
    for (const [index, part] of parts.entries()) {
      path = path ? `${path}.${part}` : part;
      const leaf = index === parts.length - 1;
      let child = node.children.find((item) => item.name === part && Boolean(item.leaf) === leaf);
      if (!child) {
        child = { name: part, path: leaf ? key : path, leaf, children: [] };
        node.children.push(child);
      }

      node = child;
    }
  }

  return root.children;
}
