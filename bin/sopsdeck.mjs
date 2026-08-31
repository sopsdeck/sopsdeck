#!/usr/bin/env node

import { access, chmod, mkdir, readFile, rename, stat, writeFile } from 'node:fs/promises';
import { Buffer } from 'node:buffer';
import process from 'node:process';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { arch, homedir, platform } from 'node:os';
import { spawn } from 'node:child_process';

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
  'configure_integration',
  'commit',
  'sync',
  'review',
  'history',
  'restore',
  'recipient',
  'publish',
  'files',
  'project',
  'references',
  'unused',
  'rename',
  'scan',
  'mcp',
]);

function usage() {
  return `sopsdeck [PROJECT]

Open the local Sopsdeck workspace in your browser.

  sopsdeck .
  sopsdeck
  sd .

Options:
  --port PORT     listen on localhost PORT (default: ${defaultPort})
  --no-open       print the URL without opening a browser
  -h, --help      show this help

Install in a project with: npm install -D sopsdeck
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

function cachePath() {
  const root = process.env.SOPSDECK_CACHE_DIR || path.join(homedir(), '.cache', 'sopsdeck');
  const suffix = platform() === 'win32' ? '.exe' : '';
  return path.join(root, `v${packageJson.version}`, `${platform()}-${arch()}`, `sopsdeck${suffix}`);
}

async function downloadBinary(target) {
  const asset = assetName();
  const base =
    process.env.SOPSDECK_RELEASE_BASE_URL ||
    `https://github.com/sopsdeck/sopsdeck/releases/download/v${packageJson.version}`;
  const url = process.env.SOPSDECK_BINARY_URL || `${base.replace(/\/$/, '')}/${asset}`;
  process.stderr.write(`sopsdeck: downloading ${asset}...\n`);
  const response = await fetch(url);
  if (!response.ok) throw new Error(`download failed (${response.status}) from ${url}`);
  const temporary = `${target}.${process.pid}.tmp`;
  await mkdir(path.dirname(target), { recursive: true });
  await writeFile(temporary, Buffer.from(await response.arrayBuffer()), { mode: 0o755 });
  await chmod(temporary, 0o755);
  await rename(temporary, target);
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

  if (args[0] === '--version' || args[0] === '-V') {
    process.stdout.write(`${packageJson.version}\n`);
    return 0;
  }

  if (cliCommands.has(args[0])) return run(await runnerPath(), args, { stdio: 'inherit' });
  const options = parseLauncherArgs(args);

  const projectInfo = await stat(options.project);
  if (!projectInfo.isDirectory()) throw new Error(`${options.project} is not a folder`);

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

if (import.meta.url === pathToFileURL(path.resolve(process.argv[1] || '')).href) {
  try {
    process.exitCode = await main();
  } catch (error) {
    process.stderr.write(`sopsdeck: ${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  }
}

export { assetName, parseLauncherArgs };
