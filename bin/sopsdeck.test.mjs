import process from 'node:process';
import { expect, test } from 'bun:test';
import { assetName, parseLauncherArgs } from './sopsdeck.mjs';

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
