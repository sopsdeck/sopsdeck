import { expect, test } from 'bun:test';
import { clipboardFingerprint, dismissClipboard, wasClipboardDismissed } from './clipboard.js';

function memoryStorage(initial = {}) {
  const data = { ...initial };
  return {
    getItem(key) {
      return Object.hasOwn(data, key) ? data[key] : null;
    },
    setItem(key, value) {
      data[key] = String(value);
    },
  };
}

test('dismissed clipboard text is remembered by fingerprint, not plaintext', () => {
  const storage = memoryStorage();
  const secret = 'sk_live_super_secret_value';
  expect(wasClipboardDismissed(secret, storage)).toBe(false);
  dismissClipboard(secret, storage);
  expect(wasClipboardDismissed(secret, storage)).toBe(true);
  expect(wasClipboardDismissed('sk_live_other', storage)).toBe(false);
  expect(storage.getItem('sopsdeck-clipboard-dismissed')).not.toContain(secret);
  expect(clipboardFingerprint(secret)).toMatch(/^[\da-f]+$/);
});
