import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

export const minClipSeconds = 1;

export function mediaDurationSeconds(path) {
  if (path.endsWith('.cast')) {
    return castDurationSeconds(readFileSync(path, 'utf8'));
  }

  const probed = ffprobeSeconds(path);
  if (Number.isFinite(probed) && probed > 0) {
    return probed;
  }

  return webmDurationSeconds(readFileSync(path));
}

function ffprobeSeconds(path) {
  const probed = spawnSync(
    'ffprobe',
    ['-v', 'error', '-show_entries', 'format=duration', '-of', 'csv=p=0', path],
    { encoding: 'utf8' },
  );
  if (probed.status !== 0) {
    return Number.NaN;
  }

  return Number.parseFloat(probed.stdout.trim());
}

function castDurationSeconds(text) {
  let last = 0;
  for (const line of text.split('\n')) {
    if (!line.startsWith('[')) {
      continue;
    }

    let row;
    try {
      row = JSON.parse(line);
    } catch {
      continue;
    }

    if (Array.isArray(row) && typeof row[0] === 'number') {
      last = row[0];
    }
  }

  return last;
}

function readVint(buf, offset) {
  if (offset >= buf.length) {
    return null;
  }

  const first = buf[offset];
  let width = 1;
  let mask = 0x80;
  while (width <= 8 && (first & mask) === 0) {
    width += 1;
    mask >>= 1;
  }

  if (width > 8 || offset + width > buf.length) {
    return null;
  }

  let value = first & (mask - 1);
  for (let i = 1; i < width; i += 1) {
    value = value * 256 + buf[offset + i];
  }

  return { value, length: width };
}

export function webmDurationSeconds(buf) {
  const limit = Math.min(buf.length - 4, 16_384);
  for (let i = 0; i < limit; i += 1) {
    if (buf[i] !== 0x44 || buf[i + 1] !== 0x89) {
      continue;
    }

    const size = readVint(buf, i + 2);
    if (!size) {
      continue;
    }

    const start = i + 2 + size.length;
    if (start + size.value > buf.length) {
      continue;
    }

    let raw = Number.NaN;
    if (size.value === 4) {
      raw = buf.readFloatBE(start);
    } else if (size.value === 8) {
      raw = buf.readDoubleBE(start);
    } else {
      continue;
    }

    if (!Number.isFinite(raw) || raw <= 0) {
      continue;
    }

    return raw > 100 ? raw / 1000 : raw;
  }

  return Number.NaN;
}
