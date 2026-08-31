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

export function icon(kind) {
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
    lock: [
      svgEl('rect', { x: '5', y: '10', width: '14', height: '11', rx: '2', ...stroke }),
      svgEl('path', { d: 'M8 10V7a4 4 0 0 1 8 0v3', ...stroke }),
    ],
    unlock: [
      svgEl('rect', { x: '5', y: '10', width: '14', height: '11', rx: '2', ...stroke }),
      svgEl('path', { d: 'M8 10V7a4 4 0 0 1 7-2', ...stroke }),
    ],
    robot: [
      svgEl('rect', { x: '5', y: '7', width: '14', height: '12', rx: '3', ...stroke }),
      svgEl('path', { d: 'M12 3v4M8 12h.01M16 12h.01M9 16h6', ...stroke }),
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
    github: [
      svgEl('path', {
        d: 'M12 .5C5.65 .5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.57.1.78-.25.78-.55v-2.1c-3.2.7-3.87-1.35-3.87-1.35-.52-1.33-1.28-1.68-1.28-1.68-1.04-.71.08-.7.08-.7 1.15.08 1.76 1.18 1.76 1.18 1.03 1.76 2.7 1.25 3.36.96.1-.74.4-1.25.73-1.54-2.55-.29-5.23-1.28-5.23-5.68 0-1.25.45-2.27 1.18-3.07-.12-.29-.51-1.46.11-3.03 0 0 .96-.31 3.15 1.17A10.9 10.9 0 0 1 12 6.17c.97 0 1.94.13 2.85.38 2.19-1.48 3.15-1.17 3.15-1.17.62 1.57.23 2.74.11 3.03.73.8 1.18 1.82 1.18 3.07 0 4.41-2.69 5.38-5.25 5.67.41.36.78 1.08.78 2.18v3.23c0 .3.2.66.79.55A11.52 11.52 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5Z',
        fill: 'currentColor',
        stroke: 'none',
      }),
    ],
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

export function iconButton(testid, label, kind, onClick) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'icon-button';
  btn.dataset.testid = testid;
  btn.setAttribute('aria-label', label);
  btn.append(icon(kind));
  btn.addEventListener('click', onClick);
  return btn;
}
