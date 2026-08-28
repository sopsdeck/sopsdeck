export function escapeHtml(text) {
  return text
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

export function rewriteDocHrefs(href) {
  const trimmed = href.trim();
  const hash = trimmed.indexOf('#');
  const path = hash === -1 ? trimmed : trimmed.slice(0, hash);
  const suffix = hash === -1 ? '' : trimmed.slice(hash);
  const base = path.split('/').pop() ?? path;
  if (base === 'CONTEXT.md') {
    return `glossary.html${suffix}`;
  }

  if (base.endsWith('.md') && !path.includes('.scratch')) {
    return `${base.replace(/\.md$/, '.html')}${suffix}`;
  }

  if (path.startsWith('assets/')) {
    return `../../docs/${path}${suffix}`;
  }

  return trimmed;
}

function inline(text) {
  let out = escapeHtml(text);
  out = out.replaceAll(/`([^`]+)`/g, '<code>$1</code>');
  out = out.replaceAll(
    /\[([^\]]+)\]\(([^)]+)\)/g,
    (_, label, href) => `<a href="${escapeHtml(rewriteDocHrefs(href))}">${label}</a>`,
  );
  out = out.replaceAll(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  out = out.replaceAll(/_([^_]+)_/g, '<em>$1</em>');
  return out;
}

function parseTable(lines, start) {
  const rows = [];
  let i = start;
  while (i < lines.length && lines[i].includes('|')) {
    const cells = lines[i]
      .split('|')
      .slice(1, -1)
      .map((cell) => cell.trim());
    if (!cells.every((cell) => /^[-:]+$/.test(cell))) {
      rows.push(cells);
    }

    i += 1;
  }

  if (rows.length === 0) {
    return null;
  }

  const [header, ...body] = rows;
  const head = header.map((cell) => `<th>${inline(cell)}</th>`).join('');
  const trs = body
    .map((row) => `<tr>${row.map((cell) => `<td>${inline(cell)}</td>`).join('')}</tr>`)
    .join('\n');
  return {
    html: `<table>\n<thead><tr>${head}</tr></thead>\n<tbody>\n${trs}\n</tbody>\n</table>`,
    next: i,
  };
}

export function mdToHtml(md) {
  const prepared = md.replaceAll(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt, src) => {
    const name = src.split('/').pop();
    return `[${alt || name}](${src})`;
  });
  const lines = prepared.split('\n');
  const out = [];
  let i = 0;
  let list = null;
  let fence = null;

  const closeList = () => {
    if (!list) {
      return;
    }

    out.push(`</${list}>`);
    list = null;
  };

  while (i < lines.length) {
    const line = lines[i];
    if (fence !== null) {
      if (line.startsWith('```')) {
        out.push(`<pre><code>${escapeHtml(fence.join('\n'))}</code></pre>`);
        fence = null;
      } else {
        fence.push(line);
      }

      i += 1;
      continue;
    }

    if (line.startsWith('```')) {
      closeList();
      fence = [];
      i += 1;
      continue;
    }

    if (line.includes('|') && i + 1 < lines.length && /^\s*\|?\s*[-:| ]+\s*$/.test(lines[i + 1])) {
      closeList();
      const table = parseTable(lines, i);
      if (table) {
        out.push(table.html);
        i = table.next;
        continue;
      }
    }

    const heading = /^(#{1,3}) (.+)$/.exec(line);
    if (heading) {
      closeList();
      const level = heading[1].length;
      out.push(`<h${level}>${inline(heading[2])}</h${level}>`);
      i += 1;
      continue;
    }

    const ul = /^[-*] (.+)$/.exec(line);
    if (ul) {
      if (list !== 'ul') {
        closeList();
        out.push('<ul>');
        list = 'ul';
      }

      out.push(`<li>${inline(ul[1])}</li>`);
      i += 1;
      continue;
    }

    if (line.trim() === '') {
      closeList();
      i += 1;
      continue;
    }

    closeList();
    out.push(`<p>${inline(line)}</p>`);
    i += 1;
  }

  closeList();
  if (fence !== null) {
    out.push(`<pre><code>${escapeHtml(fence.join('\n'))}</code></pre>`);
  }

  return out.join('\n');
}

const logo = `<svg class="mark" viewBox="0 0 64 64" aria-hidden="true">
            <path
              d="M11 13h36l6 7-13 12 13 12-6 7H11l13-19L11 13Z"
              fill="none"
              stroke="currentColor"
              stroke-width="6"
              stroke-linejoin="round"
            />
            <path d="M25 32h15" stroke="#46D6A8" stroke-width="6" stroke-linecap="round" />
          </svg>`;

export function sitePage({ title, kicker, heading, lede, body, base, active }) {
  const root = base ?? '';
  const productHref = `${root}index.html`;
  const docsHref = `${root}docs/`;
  const notesHref = `${root}changelog.html`;
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>${escapeHtml(title)}</title>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
      href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@500;600&family=Manrope:wght@400;500;600;700;800&display=swap"
      rel="stylesheet"
    />
    <style>
      :root {
        --ink: #101828;
        --blue: #3157f6;
        --mint: #46d6a8;
        --paper: #f7f9fc;
        --slate: #475467;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        color: var(--ink);
        background: var(--paper);
        font: 15px/1.5 Manrope, system-ui, sans-serif;
      }
      a { color: inherit; }
      a:hover { color: var(--blue); }
      .wrap { width: min(800px, calc(100% - 40px)); margin: 0 auto; }
      header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        padding: 22px 0;
      }
      .logo {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 18px;
        font-weight: 800;
        letter-spacing: -0.04em;
        text-decoration: none;
      }
      .mark { width: 28px; height: 28px; }
      nav {
        display: flex;
        gap: 22px;
        color: var(--slate);
        font-size: 13px;
        font-weight: 600;
      }
      nav a { text-decoration: none; }
      nav a[aria-current='page'] { color: var(--ink); }
      .hero {
        padding: 36px 40px 40px;
        color: var(--paper);
        border-radius: 24px;
        background: var(--ink);
      }
      .kicker {
        color: var(--mint);
        font: 600 11px 'IBM Plex Mono', ui-monospace, monospace;
        letter-spacing: 0.08em;
        text-transform: uppercase;
      }
      .hero h1 {
        margin: 10px 0 8px;
        font-size: clamp(32px, 6vw, 48px);
        letter-spacing: -0.05em;
      }
      .hero p { margin: 0; max-width: 52ch; color: rgba(247, 249, 252, 0.72); }
      .hero a { color: var(--mint); }
      .prose { margin: 36px 0 48px; color: var(--slate); }
      .prose h1, .prose h2, .prose h3 {
        color: var(--ink);
        letter-spacing: -0.04em;
      }
      .prose h1 { font-size: 28px; }
      .prose h2 { margin-top: 32px; font-size: 22px; }
      .prose code, .prose pre {
        font-family: 'IBM Plex Mono', ui-monospace, monospace;
        font-size: 13px;
      }
      .prose pre {
        overflow: auto;
        padding: 14px 16px;
        border-radius: 14px;
        background: #fff;
        border: 1px solid rgba(16, 24, 40, 0.08);
      }
      .prose table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;
      }
      .prose th, .prose td {
        text-align: left;
        vertical-align: top;
        padding: 8px 10px;
        border-bottom: 1px solid rgba(16, 24, 40, 0.08);
      }
      .prose ul { padding-left: 1.2em; }
      .doc-index { list-style: none; padding: 0; }
      .doc-index li {
        margin: 0 0 10px;
        padding: 14px 16px;
        border: 1px solid rgba(16, 24, 40, 0.08);
        border-radius: 14px;
        background: #fff;
      }
      .release { margin: 0 0 48px; }
      .release h2 {
        margin: 6px 0 18px;
        font-size: 28px;
        letter-spacing: -0.04em;
        color: var(--ink);
      }
      .release > .kicker { color: var(--slate); }
      .notes {
        margin: 0;
        padding: 0;
        list-style: none;
        counter-reset: note;
      }
      .notes li {
        counter-increment: note;
        display: grid;
        grid-template-columns: 2.2em 1fr;
        gap: 12px;
        margin: 0 0 10px;
        padding: 14px 16px;
        border: 1px solid rgba(16, 24, 40, 0.08);
        border-radius: 14px;
        background: #fff;
        color: var(--slate);
        line-height: 1.45;
      }
      .notes li:before {
        content: counter(note, decimal-leading-zero);
        color: var(--blue);
        font: 600 11px 'IBM Plex Mono', ui-monospace, monospace;
      }
      footer { padding: 8px 0 48px; color: var(--slate); font-size: 13px; }
    </style>
  </head>
  <body>
    <div class="wrap">
      <header>
        <a class="logo" href="${productHref}">
          ${logo}
          sopsdeck
        </a>
        <nav>
          <a href="${productHref}"${active === 'product' ? ' aria-current="page"' : ''}>Product</a>
          <a href="${docsHref}"${active === 'docs' ? ' aria-current="page"' : ''}>Docs</a>
          <a href="${notesHref}"${active === 'notes' ? ' aria-current="page"' : ''}>Notes</a>
        </nav>
      </header>
      <section class="hero">
        <p class="kicker">${escapeHtml(kicker)}</p>
        <h1>${escapeHtml(heading)}</h1>
        <p>${lede}</p>
      </section>
      <div class="prose">
${body}
      </div>
      <footer>Canonical domain sopsdeck.com. Generated pages stay in sync via ./scripts/docs.</footer>
    </div>
  </body>
</html>
`;
}
