#!/usr/bin/env node

import { createWriteStream, realpathSync } from 'node:fs';
import { access, chmod, mkdir, readFile, rename, stat, unlink } from 'node:fs/promises';
import process from 'node:process';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { arch, homedir, platform } from 'node:os';
import { spawn } from 'node:child_process';
import http from 'node:http';
import https from 'node:https';
import net from 'node:net';
import { PassThrough } from 'node:stream';
import { pipeline } from 'node:stream/promises';

const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const packageJson = JSON.parse(await readFile(path.join(packageRoot, 'package.json'), 'utf8'));
const defaultPort = 4174;
const cliCommands = new Set([
  'get',
  'set',
  'del',
  'lock',
  'unlock',
  'status',
  'copy',
  'run',
  'identity',
  'account',
  'robot',
  'sync',
  'review',
  'history',
  'restore',
  'recipient',
  'files',
  'project',
  'references',
  'unused',
  'rename',
  'scan',
  'drive',
  'team',
  'sops',
  'encrypt',
  'decrypt',
  'edit',
  'rotate',
  'updatekeys',
  'unset',
  'exec-env',
  'exec-file',
  'filestatus',
  'groups',
  'keyservice',
  'publish',
  'completion',
  'help',
  'h',
]);

function usage() {
  return `sopsdeck [PROJECT]
sopsdeck COMMAND [ARGS]

Open the local Sopsdeck workspace in your browser.

  sopsdeck .
  sopsdeck
  sd .

Options:
  --port PORT     listen on localhost PORT (default: ${defaultPort})
  --no-open       print the URL without opening a browser
  -h, --help      show this help

Commands:
  ${[...cliCommands].join(' ')}

  sync -f FILE   write selected secrets to configured Sync Targets
  encrypt, decrypt, edit, rotate, updatekeys, ...   use the installed SOPS CLI
  sops [ARGS]    pass any arguments directly to SOPS, including --help
  --version, -V, version   print the version

Install in a project with: npm install -D @sopsdeck/sopsdeck
`;
}

function assetName(os = platform(), cpu = arch()) {
  const names = {
    darwin: { arm64: 'sopsdeck-darwin-arm64', x64: 'sopsdeck-darwin-amd64' },
    linux: { arm64: 'sopsdeck-linux-arm64', x64: 'sopsdeck-linux-amd64' },
    win32: { x64: 'sopsdeck-windows-amd64.exe' },
  };
  const name = names[os]?.[cpu];
  if (!name) throw new Error(`unsupported platform: ${os}/${cpu}`);
  return name;
}

function parseLauncherArgs(args) {
  let project = null;
  let port = defaultPort;
  let open = true;
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--no-open') {
      open = false;
    } else if (arg === '--port' || arg.startsWith('--port=')) {
      const value = arg === '--port' ? args[++i] : arg.slice('--port='.length);
      port = Number(value);
      if (!Number.isInteger(port) || port < 1 || port > 65_535) {
        throw new Error('--port must be an integer between 1 and 65535');
      }
    } else if (arg.startsWith('-')) {
      throw new Error(`unknown option: ${arg}`);
    } else if (project) {
      throw new Error('only one project folder is supported');
    } else {
      project = path.resolve(arg);
    }
  }

  return { project: project ?? path.resolve('.'), port, open };
}

async function fileExists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

async function assertPortAvailable(port) {
  const server = net.createServer();
  await new Promise((resolve, reject) => {
    server.once('error', (error) => {
      if (error.code === 'EADDRINUSE') {
        reject(
          new Error(
            `Port ${port} is already in use. Stop the existing instance or choose --port PORT.`,
          ),
        );
      } else {
        reject(error);
      }
    });
    server.listen(port, '127.0.0.1', resolve);
  });
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()));
  });
}

function cachePath() {
  const root = process.env.SOPSDECK_CACHE_DIR || path.join(homedir(), '.cache', 'sopsdeck');
  const suffix = platform() === 'win32' ? '.exe' : '';
  return path.join(root, `v${packageJson.version}`, `${platform()}-${arch()}`, `sopsdeck${suffix}`);
}

function fetchBinary(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    const client = url.protocol === 'http:' ? http : https;
    const request = client.get(url, (response) => {
      if (
        response.statusCode >= 300 &&
        response.statusCode < 400 &&
        response.headers.location &&
        redirects < 5
      ) {
        response.resume();
        resolve(fetchBinary(new URL(response.headers.location, url), redirects + 1));

        return;
      }

      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`download failed (${response.statusCode}) from ${url}`));

        return;
      }

      resolve({ response, total: Number(response.headers['content-length'] || 0) });
    });
    request.on('error', reject);
  });
}

async function downloadBinary(target) {
  const asset = assetName();
  const base =
    process.env.SOPSDECK_RELEASE_BASE_URL ||
    `https://github.com/sopsdeck/sopsdeck/releases/download/v${packageJson.version}`;
  const url = process.env.SOPSDECK_BINARY_URL || `${base.replace(/\/$/, '')}/${asset}`;
  process.stderr.write(`sopsdeck: downloading ${asset}...\n`);
  const { response, total } = await fetchBinary(new URL(url));
  const temporary = `${target}.${process.pid}.tmp`;
  await mkdir(path.dirname(target), { recursive: true });
  const out = createWriteStream(temporary, { mode: 0o755 });
  const tracker = new PassThrough();
  let received = 0;
  let lastReport = 0;
  let timer;

  const stallMs = 30_000;
  const resetTimer = () => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      tracker.destroy(new Error(`download stalled (no data for ${stallMs / 1000}s) from ${url}`));
    }, stallMs);
  };

  tracker.on('data', (chunk) => {
    received += chunk.length;
    resetTimer();
    const now = Date.now();
    if (now - lastReport < 500) return;
    lastReport = now;
    const done = (received / 1_048_576).toFixed(1);
    const size = total ? `${(total / 1_048_576).toFixed(1)}MB` : '?';
    const percent = total ? ` ${Math.floor((received / total) * 100)}%` : '';
    process.stderr.write(`\rsopsdeck: downloading ${asset}... ${done}/${size}MB${percent}`);
  });
  resetTimer();
  try {
    await pipeline(response, tracker, out);
    clearTimeout(timer);
    process.stderr.write(
      `\rsopsdeck: downloaded ${asset} (${(received / 1_048_576).toFixed(1)}MB)      \n`,
    );
    await chmod(temporary, 0o755);
    await rename(temporary, target);
  } catch (error) {
    clearTimeout(timer);
    out.destroy();

    try {
      await unlink(temporary);
    } catch {}

    throw error;
  }

  return target;
}

async function runnerPath() {
  if (process.env.SOPSDECK_BIN) return process.env.SOPSDECK_BIN;
  const local = path.join(packageRoot, `sopsdeck${platform() === 'win32' ? '.exe' : ''}`);
  if (await fileExists(local)) return local;
  const cached = cachePath();
  if (await fileExists(cached)) return cached;
  return downloadBinary(cached);
}

function waitForChild(child) {
  return new Promise((resolve, reject) => {
    child.once('error', reject);
    child.once('exit', (code) => resolve(code ?? 1));
  });
}

function spawnRunner(binary, args, options = {}) {
  const child = spawn(binary, args, options);
  for (const signal of ['SIGINT', 'SIGTERM']) {
    process.once(signal, () => child.kill(signal));
  }

  return child;
}

async function waitForHealth(url, child) {
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error('server exited before it was ready');
    try {
      // eslint-disable-next-line no-await-in-loop
      const response = await fetch(`${url}/health`);
      if (response.ok) return;
    } catch {}

    // eslint-disable-next-line no-await-in-loop
    await new Promise((resolve) => {
      setTimeout(resolve, 50);
    });
  }

  throw new Error(`server did not start at ${url}`);
}

function openBrowser(url) {
  const command = platform() === 'darwin' ? 'open' : platform() === 'win32' ? 'cmd' : 'xdg-open';
  const args = platform() === 'win32' ? ['/c', 'start', '', url] : [url];
  const browser = spawn(command, args, { detached: true, stdio: 'ignore' });
  browser.once('error', () => {});
  browser.unref();
}

async function run(binary, args, options) {
  const child = spawnRunner(binary, args, options);
  return waitForChild(child);
}

async function main(args = process.argv.slice(2)) {
  if (args.length > 0 && (args[0] === '--help' || args[0] === '-h')) {
    process.stdout.write(usage());
    return 0;
  }

  if (args[0] === '--version' || args[0] === '-V' || args[0] === 'version') {
    process.stdout.write(`${packageJson.version}\n`);
    return 0;
  }

  if (
    cliCommands.has(args[0]) ||
    (args[0]?.startsWith('-') &&
      !['--port', '--no-open'].includes(args[0]) &&
      !args[0].startsWith('--port='))
  )
    return run(await runnerPath(), args, { stdio: 'inherit' });
  const options = parseLauncherArgs(args);

  let projectInfo;
  try {
    projectInfo = await stat(options.project);
  } catch (error) {
    if (error.code === 'ENOENT') {
      throw new Error(
        `unknown command or project folder: ${options.project}. Run sopsdeck --help.`,
      );
    }

    throw error;
  }

  if (!projectInfo.isDirectory()) return run(await runnerPath(), args, { stdio: 'inherit' });
  options.project = realpathSync(options.project);
  await assertPortAvailable(options.port);

  const binary = await runnerPath();
  const url = `http://127.0.0.1:${options.port}`;
  const child = spawnRunner(
    binary,
    [
      'drive',
      '--listen',
      `127.0.0.1:${options.port}`,
      '--ui',
      path.join(packageRoot, 'desktop', 'src'),
    ],
    {
      env: { ...process.env, SOPSDECK_DEV_PROJECT: options.project },
      stdio: ['inherit', 'pipe', 'inherit'],
    },
  );
  child.stdout.pipe(process.stdout);
  try {
    await waitForHealth(url, child);
  } catch (error) {
    if (child.exitCode === null) {
      child.kill('SIGTERM');
      try {
        await waitForChild(child);
      } catch {}
    }

    throw error;
  }

  process.stdout.write(`sopsdeck: open ${url}\n`);
  if (options.open) openBrowser(url);
  return waitForChild(child);
}

function isMainEntry() {
  if (!process.argv[1]) return false;
  let entry;
  try {
    entry = realpathSync(process.argv[1]);
  } catch {
    return false;
  }

  return pathToFileURL(entry).href === import.meta.url;
}

if (isMainEntry()) {
  try {
    process.exitCode = await main();
  } catch (error) {
    process.stderr.write(`sopsdeck: ${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  }
}

export { assertPortAvailable, assetName, parseLauncherArgs };
