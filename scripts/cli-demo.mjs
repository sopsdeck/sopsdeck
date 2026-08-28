#!/usr/bin/env bun

import { spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const outDir = join(root, 'docs/assets');
const bin = join(root, 'test-results', 'sopsdeck-cli-demo');
const age = join(root, 'testdata', 'age.txt');
const hello = join(root, 'testdata', 'hello.env');

mkdirSync(join(root, 'test-results'), { recursive: true });
mkdirSync(outDir, { recursive: true });
const scratch = join(root, 'test-results', 'cli-demo');
mkdirSync(scratch, { recursive: true });

const built = spawnSync('go', ['build', '-o', bin, './cmd/sopsdeck'], {
  cwd: root,
  encoding: 'utf8',
});
if (built.status !== 0) {
  process.stderr.write(built.stderr || built.stdout);
  process.exit(1);
}

function run(args, cwd) {
  const result = spawnSync(bin, args, {
    cwd,
    encoding: 'utf8',
    env: { ...process.env, SOPS_AGE_KEY_FILE: age },
  });
  if (result.status !== 0) {
    throw new Error(
      `sopsdeck ${args.join(' ')} exited ${result.status}\n${result.stderr}${result.stdout}`,
    );
  }

  return result;
}

function git(cwd, args) {
  const result = spawnSync('git', args, {
    cwd,
    encoding: 'utf8',
    env: { ...process.env, GIT_TEMPLATE_DIR: '' },
  });
  if (result.status !== 0) {
    throw new Error(`git ${args.join(' ')}: ${result.stderr}${result.stdout}`);
  }

  return result.stdout;
}

function initRepo(dir) {
  git(dir, ['init']);
  git(dir, ['config', 'user.email', 'demo@sopsdeck.example']);
  git(dir, ['config', 'user.name', 'Sopsdeck Demo']);
  git(dir, ['checkout', '-b', 'main']);
}

class Cast {
  constructor(title) {
    this.title = title;
    this.events = [];
    this.t = 0.12;
  }

  typeLine(text) {
    this.events.push([round(this.t), 'o', '$ ']);
    this.t += 0.18;
    for (const ch of text) {
      this.events.push([round(this.t), 'o', ch]);
      this.t += 0.045;
    }

    this.events.push([round(this.t), 'o', '\r\n']);
    this.t += 0.25;
  }

  output(text) {
    if (!text) {
      this.t += 0.45;
      return;
    }

    const body = text.endsWith('\n') ? text : `${text}\n`;
    const normalized = body.replaceAll('\n', '\r\n');
    this.events.push([round(this.t), 'o', normalized]);
    this.t += 0.55 + Math.min(1.2, normalized.length / 80);
  }

  hold() {
    this.t += 1.6;
    this.events.push([round(this.t), 'o', '']);
  }

  write(name) {
    const header = JSON.stringify({
      version: 2,
      width: 80,
      height: 24,
      title: this.title,
      env: { TERM: 'xterm-256color' },
    });
    const body = this.events.map((row) => JSON.stringify(row)).join('\n');
    writeFileSync(join(outDir, name), `${header}\n${body}\n`);
  }
}

function round(n) {
  return Math.round(n * 1000) / 1000;
}

function recordGet() {
  const cwd = mkdtempSync(join(scratch, 'get-'));
  copyFileSync(hello, join(cwd, 'hello.env'));
  const got = run(['get', 'HELLO', '-f', 'hello.env'], cwd);
  const cast = new Cast('sopsdeck get');
  cast.typeLine('sopsdeck get HELLO -f hello.env');
  cast.output(got.stdout);
  cast.hold();
  cast.write('cli-get.cast');
}

function recordSet() {
  const cwd = mkdtempSync(join(scratch, 'set-'));
  copyFileSync(hello, join(cwd, 'hello.env'));
  const set = run(['set', 'API_TOKEN', 'sk_demo', '-f', 'hello.env'], cwd);
  const got = run(['get', 'API_TOKEN', '-f', 'hello.env'], cwd);
  const cast = new Cast('sopsdeck set');
  cast.typeLine('sopsdeck set API_TOKEN sk_demo -f hello.env');
  cast.output(set.stdout);
  cast.typeLine('sopsdeck get API_TOKEN -f hello.env');
  cast.output(got.stdout);
  cast.hold();
  cast.write('cli-set.cast');
}

function recordCommit() {
  const cwd = mkdtempSync(join(scratch, 'commit-'));
  initRepo(cwd);
  copyFileSync(hello, join(cwd, 'hello.env'));
  const committed = run(['commit', '-m', 'add production secrets', '-f', 'hello.env'], cwd);
  const subject = git(cwd, ['log', '-1', '--pretty=%s']);
  const cast = new Cast('sopsdeck commit');
  cast.typeLine('sopsdeck commit -m "add production secrets" -f hello.env');
  cast.output(committed.stdout);
  cast.typeLine('git log -1 --pretty=%s');
  cast.output(subject);
  cast.hold();
  cast.write('cli-commit.cast');
}

function recordSync() {
  const bare = mkdtempSync(join(scratch, 'origin-'));
  git(bare, ['init', '--bare']);
  git(bare, ['symbolic-ref', 'HEAD', 'refs/heads/main']);

  const cwd = mkdtempSync(join(scratch, 'sync-'));
  initRepo(cwd);
  git(cwd, ['remote', 'add', 'origin', bare]);
  copyFileSync(hello, join(cwd, 'hello.env'));
  run(['commit', '-m', 'first', '-f', 'hello.env'], cwd);
  git(cwd, ['push', '-u', 'origin', 'main']);
  copyFileSync(join(root, 'testdata', 'hello.json'), join(cwd, 'hello.json'));
  run(['commit', '-m', 'add hello.json', '-f', 'hello.json'], cwd);
  const synced = run(['sync'], cwd);
  const log = git(cwd, ['log', 'origin/main', '--pretty=%s']);
  const cast = new Cast('sopsdeck sync');
  cast.typeLine('sopsdeck sync');
  cast.output(synced.stdout);
  cast.typeLine('git log origin/main --pretty=%s');
  cast.output(log);
  cast.hold();
  cast.write('cli-sync.cast');
}

recordGet();
recordSet();
recordCommit();
recordSync();
process.stdout.write('wrote docs/assets/cli-get.cast cli-set.cast cli-commit.cast cli-sync.cast\n');
