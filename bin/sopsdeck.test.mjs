import process from 'node:process';
import net from 'node:net';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test } from 'bun:test';
import { assertPortAvailable, assetName, parseLauncherArgs } from './sopsdeck.mjs';

test('launcher refuses a busy port instead of opening another instance', async () => {
  const server = net.createServer();
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const { port } = server.address();
  try {
    await expect(assertPortAvailable(port)).rejects.toThrow(`Port ${port} is already in use`);
  } finally {
    await new Promise((resolve) => {
      server.close(resolve);
    });
  }

  await assertPortAvailable(port);
});

test('launcher maps supported platforms to release assets', () => {
  expect(assetName('darwin', 'arm64')).toBe('sopsdeck-darwin-arm64');
  expect(assetName('darwin', 'x64')).toBe('sopsdeck-darwin-amd64');
  expect(assetName('linux', 'arm64')).toBe('sopsdeck-linux-arm64');
});

test('launcher defaults to the current folder and localhost port', () => {
  expect(parseLauncherArgs([])).toEqual({
    project: process.cwd(),
    port: 4174,
    open: true,
  });
  expect(parseLauncherArgs(['.'])).toEqual({
    project: process.cwd(),
    port: 4174,
    open: true,
  });
});

test('launcher rejects removed commands without starting or downloading a runner', () => {
  const cwd = mkdtempSync(path.join(tmpdir(), 'sopsdeck-launcher-'));
  try {
    for (const command of ['commit', 'mcp', 'configure_integration']) {
      const result = spawnSync(
        process.execPath,
        [fileURLToPath(new URL('sopsdeck.mjs', import.meta.url)), command],
        {
          cwd,
          env: { ...process.env, SOPSDECK_BIN: path.join(cwd, 'must-not-run') },
          encoding: 'utf8',
        },
      );
      expect(result.status).toBe(1);
      expect(result.stdout).toBe('');
      expect(result.stderr).toContain('unknown command or project folder:');
      expect(result.stderr).toContain(path.join(cwd, command));
      expect(result.stderr).toContain('sopsdeck --help');
    }
  } finally {
    rmSync(cwd, { recursive: true, force: true });
  }
});

test('launcher help lists current commands and explains Secret Sync', () => {
  const result = spawnSync(
    process.execPath,
    [fileURLToPath(new URL('sopsdeck.mjs', import.meta.url)), '--help'],
    { encoding: 'utf8' },
  );
  expect(result.status).toBe(0);
  expect(result.stderr).toBe('');
  expect(result.stdout).toContain('get set del');
  expect(result.stdout).toContain('sync -f FILE');
  expect(result.stdout).toContain('Sync Targets');
  expect(result.stdout).toContain('installed SOPS CLI');
  for (const command of ['commit', 'mcp', 'configure_integration']) {
    expect(result.stdout).not.toContain(command);
  }
});

test('launcher routes native SOPS commands, flags, and files to the runner', () => {
  const cwd = mkdtempSync(path.join(tmpdir(), 'sopsdeck-launcher-'));
  const runner = path.join(cwd, 'runner');
  writeFileSync(runner, '#!/bin/sh\nprintf "%s\\n" "$@"\n', { mode: 0o700 });
  writeFileSync(path.join(cwd, 'config.json'), '{}');
  try {
    for (const args of [
      ['decrypt', 'config.json'],
      ['--decrypt', 'config.json'],
      ['sops', '--help'],
      ['config.json'],
    ]) {
      const result = spawnSync(
        process.execPath,
        [fileURLToPath(new URL('sopsdeck.mjs', import.meta.url)), ...args],
        { cwd, env: { ...process.env, SOPSDECK_BIN: runner }, encoding: 'utf8' },
      );
      expect(result.status).toBe(0);
      expect(result.stdout).toBe(args.join('\n') + '\n');
      expect(result.stderr).toBe('');
    }
  } finally {
    rmSync(cwd, { recursive: true, force: true });
  }
});

test('launcher version command prints the version without starting a runner', () => {
  const result = spawnSync(
    process.execPath,
    [fileURLToPath(new URL('sopsdeck.mjs', import.meta.url)), 'version'],
    { encoding: 'utf8', env: { ...process.env, SOPSDECK_BIN: 'must-not-run' } },
  );
  expect(result.status).toBe(0);
  expect(result.stderr).toBe('');
  expect(result.stdout).toMatch(/^\d+\.\d+\.\d+\n$/);
});
