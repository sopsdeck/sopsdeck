#!/usr/bin/env bun

import { copyFileSync, existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const bin = join(root, 'test-results', 'sopsdeck-castkit');
const output = join(root, 'docs/assets/cli-overview.mp4');
const renderer = process.env.CASTKIT_RENDERER_HOME || join(root, 'renderer-runtime');
const castkitHome = mkdtempSync(join(tmpdir(), 'sopsdeck-castkit-home-'));
const browserCache =
  process.env.PLAYWRIGHT_BROWSERS_PATH ||
  join(process.env.HOME || '', 'Library/Caches/ms-playwright');
const env = {
  ...process.env,
  HOME: castkitHome,
  CASTKIT_HOME: castkitHome,
  PLAYWRIGHT_BROWSERS_PATH: browserCache,
  CASTKIT_RENDERER_HOME: renderer,
};

try {
  if (!existsSync(join(renderer, 'render.mjs'))) {
    fail(`Castkit renderer not found at ${renderer}; set CASTKIT_RENDERER_HOME`);
  }

  mkdirSync(join(root, 'test-results'), { recursive: true });
  run('go', ['build', '-o', bin, './cmd/sopsdeck']);

  const init = run('castkit', ['handoff', 'init', bin, '--json'], env);
  const session = JSON.parse(init.stdout).session_id;
  const planDir = mkdtempSync(join(root, 'test-results', 'castkit-plan-'));
  const plan = join(planDir, 'cli.json');
  writeFileSync(plan, JSON.stringify(demoPlan(), null, 2));

  run('castkit', ['validate', '--session', session, '--script', plan, '--json'], env);
  run(
    'castkit',
    [
      'execute',
      '--session',
      session,
      '--script',
      plan,
      '--non-interactive',
      '--preset',
      'polished',
      '--output',
      output,
      '--json',
    ],
    env,
  );
  copyFileSync(output, join(root, 'site/public/assets/cli-overview.mp4'));
  rmSync(planDir, { recursive: true, force: true });
  process.stdout.write(`wrote ${output}\n`);
} finally {
  rmSync(castkitHome, { recursive: true, force: true });
}

function demoPlan() {
  const file = (name) => shQuote(join(root, 'testdata', name));
  const target = shQuote(bin);
  return {
    version: '1',
    mode: 'terminal',
    setup: [],
    scenes: [
      scene('overview', 'Sopsdeck CLI', `${target} --help`, 'get set del run'),
      scene('files', 'Find Managed Files', `${target} files ${shQuote(root)}`, 'hello.env'),
      scene(
        'get',
        'Read a Secret',
        `SOPS_AGE_KEY_FILE=${file('age.txt')} ${target} get HELLO -f ${file('hello.json')}`,
        'world',
      ),
    ],
    checks: [],
    cleanup: [],
    redactions: [],
    audio: { typing: true, music_path: null },
    branding: { title: 'sopsdeck', watermark_text: 'sopsdeck.com' },
  };
}

function scene(id, title, command, contains) {
  return {
    id,
    title,
    steps: [
      {
        id,
        run: id === 'overview' ? `SOPSDECK_COLOR=always ${command}` : command,
        expect: { contains, regex: null, exit_code: 0 },
        timeout_ms: 120000,
        source_refs: ['ref_help_0001'],
        manual_step: true,
        manual_reason: 'Run the locally built Sopsdeck binary.',
        artifacts: [],
      },
    ],
  };
}

function shQuote(value) {
  return `'${value.replaceAll("'", "'\\''")}'`;
}

function run(command, args, commandEnv = process.env) {
  const result = spawnSync(command, args, { cwd: root, encoding: 'utf8', env: commandEnv });
  if (result.status !== 0) {
    fail(`${command} ${args.join(' ')}\n${result.stderr}${result.stdout}`);
  }
  return result;
}

function fail(message) {
  throw new Error(message);
}
