import { expect, test } from 'bun:test';
import { isStructuredFormat, nestLeaves, splitLeafPath } from './tree.js';

test('splitLeafPath walks dotted JSON paths and indexes', () => {
  expect(splitLeafPath('build.env.SECRET')).toEqual(['build', 'env', 'SECRET']);
  expect(splitLeafPath('items[0].name')).toEqual(['items', '0', 'name']);
});

test('nestLeaves builds a tree from leaf paths', () => {
  const tree = nestLeaves(['build.env.SECRET', 'build.env.PUBLIC', 'cli.version']);
  expect(tree.map((node) => node.name)).toEqual(['build', 'cli']);
  expect(tree[0].children[0].children.map((node) => node.name)).toEqual(['SECRET', 'PUBLIC']);
  expect(tree[0].children[0].children[0].leaf).toBe(true);
  expect(tree[0].children[0].children[0].path).toBe('build.env.SECRET');
});

test('isStructuredFormat is json and yaml only', () => {
  expect(isStructuredFormat('json')).toBe(true);
  expect(isStructuredFormat('yaml')).toBe(true);
  expect(isStructuredFormat('dotenv')).toBe(false);
});
