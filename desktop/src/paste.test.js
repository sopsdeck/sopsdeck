import { expect, test } from 'bun:test';
import {
  classifyClipboard,
  classifyPasteKeys,
  parsePastePayload,
  pastePreviewText,
} from './paste.js';

test('parsePastePayload reads dotenv, JSON, YAML, and a lone value', () => {
  expect(parsePastePayload('NEW=pasted\n')).toEqual({ NEW: 'pasted' });
  expect(parsePastePayload('{"NEW":"fromjson"}')).toEqual({ NEW: 'fromjson' });
  expect(parsePastePayload('NEW: fromyaml\n')).toEqual({ NEW: 'fromyaml' });
  expect(parsePastePayload('lonely-secret', 'NEW')).toEqual({ NEW: 'lonely-secret' });
});

test('parsePastePayload refuses a lone value without a key', () => {
  expect(() => parsePastePayload('lonely-secret')).toThrow('lone paste value requires KEY');
});

test('paste preview lists names and never values', () => {
  const incoming = parsePastePayload('NEW=supersecretvalue\nHELLO=changed\n');
  const { adds, changes } = classifyPasteKeys({ HELLO: 'world' }, incoming);
  const text = pastePreviewText(adds, changes);
  expect(text).toContain('preview 1 add 1 change');
  expect(text).toContain('add NEW');
  expect(text).toContain('change HELLO');
  expect(text).not.toContain('supersecretvalue');
  expect(text).not.toContain('changed');
  expect(text).not.toContain('world');
});

test('classifyClipboard routes paths, recipients, bulk secrets, and lone values', () => {
  const recipient = `age1${'q'.repeat(58)}`;
  expect(classifyClipboard('/Users/alice/code/app')).toEqual({
    kind: 'path',
    path: '/Users/alice/code/app',
  });
  expect(classifyClipboard(recipient)).toEqual({ kind: 'recipient', publicKey: recipient });
  expect(classifyClipboard('TOKEN=supersecret\n')).toEqual({
    kind: 'bulk',
    pairs: { TOKEN: 'supersecret' },
    names: ['TOKEN'],
  });
  expect(classifyClipboard('supersecret')).toEqual({ kind: 'lone', value: 'supersecret' });
  expect(classifyClipboard('  ')).toBeNull();
});
